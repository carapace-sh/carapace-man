package man

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carapace-sh/carapace/pkg/xdg"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

// dbSearchPaths returns directories where compiled DBs are searched.
// Uses $XDG_CONFIG_HOME/carapace/man/db/ (alongside the YAML man/ tree).
// Can be overridden with CARAPACE_MAN_DB_PATH (path-list-separated).
func dbSearchPaths() ([]string, error) {
	if env := os.Getenv("CARAPACE_MAN_DB_PATH"); env != "" {
		paths := strings.Split(env, string(os.PathListSeparator))
		for i := range paths {
			paths[i] = filepath.ToSlash(paths[i])
		}
		return paths, nil
	}

	configDir, err := xdg.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("%v/carapace/man/db", configDir)}, nil
}

// hostIndex maps (scheme, host) → db path.
type hostIndex map[string]string // key = scheme + "\x00" + host

func indexKey(scheme, host string) string {
	return scheme + "\x00" + host
}

func openBadgerRead(dbPath string) (*badger.DB, error) {
	opts := badger.DefaultOptions(dbPath).
		WithReadOnly(true).
		WithCompression(options.ZSTD).
		WithLogger(nil)
	return badger.Open(opts)
}

// isBadgerDir reports whether path looks like a badger database directory.
func isBadgerDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	// Badger stores data in .sst/.vlog files; MANIFEST identifies a DB dir.
	if _, err := os.Stat(filepath.Join(path, "MANIFEST")); err == nil {
		return true
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".sst") || strings.HasSuffix(name, ".vlog") {
			return true
		}
	}
	return false
}

// buildHostIndex scans all badger DB directories in the search paths and builds
// a (scheme, host) → dbPath index. First DB wins on conflicts.
func buildHostIndex() (hostIndex, error) {
	searchPaths, err := dbSearchPaths()
	if err != nil {
		return nil, err
	}

	var dbPaths []string
	for _, sp := range searchPaths {
		entries, err := os.ReadDir(sp)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			p := filepath.Join(sp, entry.Name())
			if isBadgerDir(p) {
				dbPaths = append(dbPaths, p)
			}
		}
	}
	sort.Strings(dbPaths)

	index := hostIndex{}
	for _, dbPath := range dbPaths {
		db, err := openBadgerRead(dbPath)
		if err != nil {
			continue
		}
		err = db.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchValues = false
			it := txn.NewIterator(opts)
			defer it.Close()
			var currentScheme, currentHost string
			for it.Rewind(); it.Valid(); it.Next() {
				key := string(it.Item().Key())
				scheme, rest, ok := strings.Cut(key, "\x00")
				if !ok {
					continue
				}
				host, _, ok := strings.Cut(rest, "\x00")
				if !ok {
					continue
				}
				if scheme == currentScheme && host == currentHost {
					continue
				}
				currentScheme, currentHost = scheme, host
				k := indexKey(scheme, host)
				if _, exists := index[k]; !exists {
					index[k] = dbPath
				}
				it.Seek(dbSchemePrefix(scheme + "\x00" + host + "\x01")) // jump past this host
			}
			return nil
		})
		db.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", dbPath, err)
		}
	}
	return index, nil
}

// lookupDB opens the DB that contains (scheme, host) read-only and runs fn
// against it. Returns false if no DB contains the (scheme, host).
func lookupDB(scheme, host string, fn func(*badger.DB) error) (bool, error) {
	index, err := buildHostIndex()
	if err != nil {
		return false, err
	}
	dbPath, ok := index[indexKey(scheme, host)]
	if !ok {
		return false, nil
	}
	return lookupDBFile(dbPath, scheme, host, fn)
}

// lookupDBFile opens a specific DB read-only and runs fn if (scheme, host)
// exists. Returns false if the scheme/host is not present.
func lookupDBFile(dbPath, scheme, host string, fn func(*badger.DB) error) (bool, error) {
	db, err := openBadgerRead(dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	found := false
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := dbHostPrefix(scheme, host)
		it.Seek(prefix)
		if it.ValidForPrefix(prefix) {
			found = true
		}
		return nil
	})
	if err != nil || !found {
		return found, err
	}
	return true, fn(db)
}

func getDoc(db *badger.DB, scheme, host, key string) (string, bool, error) {
	var result string
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(dbKey(scheme, host, key))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			result = string(val)
			return nil
		})
	})
	if err != nil {
		return "", false, err
	}
	if result == "" {
		return "", false, nil
	}
	return result, true, nil
}

// cmdDB looks up a cmd:// UID from compiled DBs.
// Returns ("", false, nil) if not found in any DB (fall through to YAML).
func cmdDB(uid *url.URL) (string, bool, error) {
	path := strings.TrimPrefix(uid.Path, "/")
	keyPrefix := path
	if keyPrefix != "" {
		keyPrefix += "/"
	}

	var key string
	if q := uid.Query(); q.Has("flag") {
		key = keyPrefix + "flag/" + q.Get("flag")
	} else {
		key = keyPrefix + "command"
	}

	var result string
	found, err := lookupDB("cmd", uid.Host, func(db *badger.DB) error {
		var innerErr error
		result, _, innerErr = getDoc(db, "cmd", uid.Host, key)
		return innerErr
	})
	if err != nil {
		return "", false, err
	}
	if !found || result == "" {
		return "", false, nil
	}
	return result, true, nil
}

// describeDB looks up a static scheme UID from compiled DBs.
// Returns ("", false, nil) if not found in any DB (fall through to YAML).
func describeDB(uid *url.URL) (string, bool, error) {
	key := strings.TrimPrefix(uid.Path, "/")

	var result string
	found, err := lookupDB(uid.Scheme, uid.Host, func(db *badger.DB) error {
		var innerErr error
		result, _, innerErr = getDoc(db, uid.Scheme, uid.Host, key)
		return innerErr
	})
	if err != nil {
		return "", false, err
	}
	if !found || result == "" {
		return "", false, nil
	}
	return result, true, nil
}

// SchemesFromDBs returns all scheme names found in compiled DBs.
func SchemesFromDBs() ([]string, error) {
	index, err := buildHostIndex()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for k := range index {
		scheme, _, _ := strings.Cut(k, "\x00")
		seen[scheme] = true
	}
	schemes := make([]string, 0, len(seen))
	for s := range seen {
		schemes = append(schemes, s)
	}
	sort.Strings(schemes)
	return schemes, nil
}

// HostsFromDBs returns all host names for a scheme found in compiled DBs.
func HostsFromDBs(scheme string) ([]string, error) {
	index, err := buildHostIndex()
	if err != nil {
		return nil, err
	}
	hosts := make([]string, 0)
	for k := range index {
		s, h, ok := strings.Cut(k, "\x00")
		if !ok {
			continue
		}
		if s == scheme {
			hosts = append(hosts, h)
		}
	}
	sort.Strings(hosts)
	return hosts, nil
}

// UidsFromDBs returns all UIDs for a (scheme, host) found in compiled DBs.
func UidsFromDBs(scheme, host string) ([]*url.URL, error) {
	var uids []*url.URL
	_, err := lookupDB(scheme, host, func(db *badger.DB) error {
		var innerErr error
		uids, innerErr = enumerateUids(db, scheme, host)
		return innerErr
	})
	if err != nil {
		return nil, err
	}
	return uids, nil
}

// enumerateUids scans a host prefix and reconstructs UIDs from its keys.
func enumerateUids(db *badger.DB, scheme, host string) ([]*url.URL, error) {
	var uids []*url.URL
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := dbHostPrefix(scheme, host)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := strings.TrimPrefix(string(it.Item().Key()), string(prefix))
			if scheme == "cmd" {
				path, field, ok := splitCmdKey(key)
				if !ok {
					continue
				}
				uidStr := fmt.Sprintf("cmd://%s", host)
				if path != "" {
					uidStr += "/" + path
				}
				if flagName, ok := strings.CutPrefix(field, "flag/"); ok {
					uidStr += "?flag=" + flagName
				} else if field != "command" {
					continue
				}
				uid, err := url.Parse(uidStr)
				if err != nil {
					return err
				}
				uids = append(uids, uid)
			} else {
				uidStr := fmt.Sprintf("%s://%s/%s", scheme, host, key)
				uid, err := url.Parse(uidStr)
				if err != nil {
					return err
				}
				uids = append(uids, uid)
			}
		}
		return nil
	})
	return uids, err
}

// splitCmdKey splits a cmd key into (path, field).
// e.g. "commit/flag/message" → ("commit", "flag/message")
//
//	"command"             → ("", "command")
//	"commit/diff/command" → ("commit/diff", "command")
func splitCmdKey(key string) (path, field string, ok bool) {
	parts := strings.Split(key, "/")
	if len(parts) == 0 {
		return "", "", false
	}
	last := parts[len(parts)-1]
	if last == "command" {
		if len(parts) == 1 {
			return "", "command", true
		}
		return strings.Join(parts[:len(parts)-1], "/"), "command", true
	}
	if len(parts) >= 2 {
		fieldPrefix := parts[len(parts)-2]
		switch fieldPrefix {
		case "flag", "positional", "dash", "example":
			field = parts[len(parts)-2] + "/" + parts[len(parts)-1]
			if len(parts) == 2 {
				return "", field, true
			}
			return strings.Join(parts[:len(parts)-2], "/"), field, true
		}
	}
	return "", "", false
}

// InspectSchemes returns all scheme names in a specific DB.
func InspectSchemes(dbPath string) ([]string, error) {
	db, err := openBadgerRead(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	seen := map[string]bool{}
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			key := string(it.Item().Key())
			scheme, _, ok := strings.Cut(key, "\x00")
			if !ok {
				continue
			}
			if seen[scheme] {
				continue
			}
			seen[scheme] = true
			it.Seek(dbSchemePrefix(scheme + "\x01"))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	schemes := make([]string, 0, len(seen))
	for s := range seen {
		schemes = append(schemes, s)
	}
	sort.Strings(schemes)
	return schemes, nil
}

// InspectHosts returns all host names for a scheme in a specific DB.
func InspectHosts(dbPath, scheme string) ([]string, error) {
	db, err := openBadgerRead(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	seen := map[string]bool{}
	prefix := dbSchemePrefix(scheme)
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := strings.TrimPrefix(string(it.Item().Key()), string(prefix))
			host, _, ok := strings.Cut(key, "\x00")
			if !ok || seen[host] {
				continue
			}
			seen[host] = true
			it.Seek(dbSchemePrefix(scheme + "\x00" + host + "\x01"))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	hosts := make([]string, 0, len(seen))
	for h := range seen {
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)
	return hosts, nil
}

// InspectUids returns all UIDs for a (scheme, host) in a specific DB.
func InspectUids(dbPath, scheme, host string) ([]*url.URL, error) {
	var uids []*url.URL
	found, err := lookupDBFile(dbPath, scheme, host, func(db *badger.DB) error {
		var innerErr error
		uids, innerErr = enumerateUids(db, scheme, host)
		return innerErr
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("scheme %q host %q not found in %s", scheme, host, dbPath)
	}
	return uids, nil
}

// InspectDescribe looks up a UID in a specific DB and returns the doc.
// Returns ("", false, nil) if the key is not present.
func InspectDescribe(dbPath string, uid *url.URL) (string, bool, error) {
	scheme := uid.Scheme
	host := uid.Host

	var key string
	if scheme == "cmd" {
		path := strings.TrimPrefix(uid.Path, "/")
		keyPrefix := path
		if keyPrefix != "" {
			keyPrefix += "/"
		}
		if q := uid.Query(); q.Has("flag") {
			key = keyPrefix + "flag/" + q.Get("flag")
		} else {
			key = keyPrefix + "command"
		}
	} else {
		key = strings.TrimPrefix(uid.Path, "/")
	}

	var result string
	found, err := lookupDBFile(dbPath, scheme, host, func(db *badger.DB) error {
		var ok bool
		var innerErr error
		result, ok, innerErr = getDoc(db, scheme, host, key)
		if !ok {
			result = ""
		}
		return innerErr
	})
	if err != nil {
		return "", false, err
	}
	return result, found && result != "", nil
}
