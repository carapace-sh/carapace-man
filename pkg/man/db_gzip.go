package man

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/carapace-sh/carapace/pkg/xdg"
	bolt "go.etcd.io/bbolt"
)

func isGzipDB(path string) bool {
	return strings.HasSuffix(path, ".db.gz")
}

// isDBFile reports whether a path is a supported database file (.db or .db.gz).
func isDBFile(name string) bool {
	return strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".db.gz")
}

func dbCacheDir() (string, error) {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "carapace", "man", "db"), nil
	}
	cacheDir, err := xdg.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "carapace", "man", "db"), nil
}

// cachePrefix returns the cache directory and filename prefix for a source DB.
// Cache entries are keyed by basename (with .db.gz/.gz stripped), assuming no
// name clashes across different source paths. Freshness is derived from the
// source file's mtime and size (hashed), so timestamp-preserving replacements
// with identical mtime and size may reuse a stale cached copy.
func cachePrefix(src string) (string, string, error) {
	cacheDir, err := dbCacheDir()
	if err != nil {
		return "", "", err
	}
	base := strings.TrimSuffix(filepath.Base(src), ".db.gz")
	base = strings.TrimSuffix(base, ".gz")
	prefix := base + "."
	return cacheDir, prefix, nil
}

func cacheFileName(prefix string, info fs.FileInfo) string {
	sum := sha1.Sum([]byte(strconv.FormatInt(info.ModTime().UnixNano(), 10) + "." + strconv.FormatInt(info.Size(), 10)))
	return prefix + hex.EncodeToString(sum[:8]) + ".db"
}

func findFreshCache(cacheDir, prefix string, info fs.FileInfo) (string, error) {
	want := cacheFileName(prefix, info)
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".db") {
			continue
		}
		path := filepath.Join(cacheDir, name)
		if name == filepath.Base(want) {
			return path, nil
		}
		// stale cache entry for this source
		_ = os.Remove(path)
	}
	return "", nil
}

func decompressGzipDB(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	gr, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gr.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".carapace-db-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		tmp.Close()
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := io.Copy(tmp, gr); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return err
	}
	ok = true
	return nil
}

func ensureDecompressedDB(src string) (string, error) {
	info, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	cacheDir, prefix, err := cachePrefix(src)
	if err != nil {
		return "", err
	}
	cachePath, err := findFreshCache(cacheDir, prefix, info)
	if err != nil {
		return "", err
	}
	if cachePath != "" {
		return cachePath, nil
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(cacheDir, filepath.Base(cacheFileName(prefix, info)))
	if err := decompressGzipDB(src, target); err != nil {
		// another process may have created a fresh cache concurrently
		if fresh, ferr := findFreshCache(cacheDir, prefix, info); ferr == nil && fresh != "" {
			return fresh, nil
		}
		return "", err
	}
	return target, nil
}

// openReadDB opens a database file read-only. For .db.gz files it uses a
// decompressed cache copy since bbolt requires a seekable file.
func openReadDB(path string) (*bolt.DB, error) {
	actual := path
	if isGzipDB(path) {
		var err error
		actual, err = ensureDecompressedDB(path)
		if err != nil {
			return nil, err
		}
	}
	return bolt.Open(actual, 0o400, &bolt.Options{ReadOnly: true})
}

func compressGzipDB(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".carapace-db-gz-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		tmp.Close()
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()

	gw := gzip.NewWriter(tmp)
	if _, err := io.Copy(gw, in); err != nil {
		gw.Close()
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return err
	}
	ok = true
	return nil
}

// prepareWriteDB prepares a writable bbolt database path for compilation.
// It returns the actual path to open with bbolt and a finalize function that
// must be called after the bbolt DB is closed (noop for plain .db files).
// For .db.gz targets, compilation happens in a temporary file which is
// compressed to target on finalize. With amend on .db.gz, the existing target
// is decompressed into the working file first.
func prepareWriteDB(target string, amend bool) (workPath string, finalize func() error, err error) {
	if !isGzipDB(target) {
		if !amend {
			if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return "", nil, err
			}
		}
		return target, func() error { return nil }, nil
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", nil, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".carapace-db-work-*.db")
	if err != nil {
		return "", nil, err
	}
	workPath = tmp.Name()
	tmp.Close()

	if amend {
		if _, err := os.Stat(target); err == nil {
			if err := decompressToPath(target, workPath); err != nil {
				_ = os.Remove(workPath)
				return "", nil, err
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			_ = os.Remove(workPath)
			return "", nil, err
		} else {
			// no existing db to amend; start fresh
			_ = os.Remove(workPath)
		}
	} else {
		_ = os.Remove(workPath)
	}

	finalize = func() error {
		defer os.Remove(workPath)
		return compressGzipDB(workPath, target)
	}
	return workPath, finalize, nil
}

// withWritableDB prepares a writable bbolt database, runs fn against it, and
// finalizes the target on success. For .db.gz targets the working file is
// removed if fn or finalization fails.
func withWritableDB(target string, amend bool, fn func(*bolt.DB) error) error {
	workPath, finalize, err := prepareWriteDB(target, amend)
	if err != nil {
		return err
	}

	db, err := bolt.Open(workPath, 0o644, nil)
	if err != nil {
		if isGzipDB(target) {
			_ = os.Remove(workPath)
		}
		return err
	}

	runErr := fn(db)
	if closeErr := db.Close(); closeErr != nil && runErr == nil {
		runErr = closeErr
	}
	if runErr != nil {
		if isGzipDB(target) {
			_ = os.Remove(workPath)
		}
		return runErr
	}
	return finalize()
}

func decompressToPath(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	gr, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gr.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		out.Close()
		if !ok {
			_ = os.Remove(dst)
		}
	}()
	if _, err := io.Copy(out, gr); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}
