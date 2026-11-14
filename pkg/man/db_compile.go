package man

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/carapace-sh/carapace/pkg/command"
	bolt "go.etcd.io/bbolt"
	"go.yaml.in/yaml/v3"
)

type compileOptions struct {
	amend bool
}

// CompileOption customizes compilation.
type CompileOption func(*compileOptions)

// WithAmend updates an existing database instead of rebuilding it from
// scratch: processed host buckets are replaced, others are preserved.
func WithAmend(amend bool) CompileOption {
	return func(o *compileOptions) { o.amend = amend }
}

func newCompileOptions(opts []CompileOption) compileOptions {
	var o compileOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// truncateDB removes an existing database file unless amending.
func truncateDB(dbPath string, amend bool) error {
	if amend {
		return nil
	}
	if err := os.Remove(dbPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

const commandSchemaURL = "https://carapace.sh/schemas/command.json"

var commandKeys = []string{
	"name", "aliases", "parsing", "flags", "persistentflags",
	"exclusiveflags", "run", "completion", "commands", "documentation", "examples",
}

func hasCommandSchemaHeader(content []byte) bool {
	return strings.Contains(string(content), "yaml-language-server: $schema="+commandSchemaURL)
}

// looksLikeCommand reports whether top-level keys of the YAML document
// resemble a carapace-spec command (headerless fallback).
func looksLikeCommand(content []byte) (bool, error) {
	var m map[string]yaml.Node
	if err := yaml.Unmarshal(content, &m); err != nil {
		return false, err
	}
	for _, key := range commandKeys {
		if _, ok := m[key]; ok {
			return true, nil
		}
	}
	return false, nil
}

// CompileSpec compiles a carapace-spec command spec into a bbolt DB.
// The host is derived from the spec's root command name (first word).
// If the spec has no name, host is derived from the filename (without .yaml).
// A spec is recognized as a command by its schema header or command-like
// content; generic scheme docs must be compiled as part of a man directory
// tree (CompileManDir) instead.
func CompileSpec(specPath, dbPath string, opts ...CompileOption) error {
	content, err := os.ReadFile(specPath)
	if err != nil {
		return err
	}
	o := newCompileOptions(opts)
	isCommand := hasCommandSchemaHeader(content)
	if !isCommand {
		isCommand, err = looksLikeCommand(content)
		if err != nil {
			return err
		}
	}
	if !isCommand {
		return fmt.Errorf("%v: not a command spec (missing %v schema header); generic scheme docs must be compiled within their man directory tree", specPath, commandSchemaURL)
	}
	var cmd command.Command
	if err := yaml.Unmarshal(content, &cmd); err != nil {
		return err
	}
	host, _, _ := strings.Cut(cmd.Name, " ")
	if host == "" {
		host = strings.TrimSuffix(filepath.Base(specPath), ".yaml")
	}
	if err := truncateDB(dbPath, o.amend); err != nil {
		return err
	}
	return compileCmd(cmd, host, dbPath, o)
}

// CompileSpecReader compiles a spec from a reader (e.g. stdin) into a bbolt DB.
// If host is empty, it's derived from the spec's root command name.
func CompileSpecReader(r io.Reader, host, dbPath string, opts ...CompileOption) error {
	content, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	o := newCompileOptions(opts)
	var cmd command.Command
	if err := yaml.Unmarshal(content, &cmd); err != nil {
		return err
	}
	if host == "" {
		host, _, _ = strings.Cut(cmd.Name, " ")
	}
	if err := truncateDB(dbPath, o.amend); err != nil {
		return err
	}
	return compileCmd(cmd, host, dbPath, o)
}

// CompileHostDir compiles all *.yaml split spec files in a directory into a
// single host bucket in a bbolt DB. The host name is the directory name.
// This handles the man/cmd/<host>/ layout where each file is a partial spec
// for one subcommand (e.g. git.commit.yaml, git.log.yaml) plus the root
// (git.yaml). The path prefix for each file is derived from the filename
// (the part after <host>. with dots replaced by slashes), or from the spec's
// name field if present.
func CompileHostDir(hostDir, dbPath string, opts ...CompileOption) error {
	return compileHostDir(hostDir, dbPath, newCompileOptions(opts))
}

func compileHostDir(hostDir, dbPath string, o compileOptions) error {
	host := filepath.Base(hostDir)
	entries, err := os.ReadDir(hostDir)
	if err != nil {
		return err
	}

	db, err := bolt.Open(dbPath, 0o644, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		schemeBucket, err := tx.CreateBucketIfNotExists([]byte("cmd"))
		if err != nil {
			return err
		}
		hostBucket, err := createHostBucket(schemeBucket, host, o)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}
			specPath := filepath.Join(hostDir, entry.Name())
			content, err := os.ReadFile(specPath)
			if err != nil {
				return err
			}
			var cmd command.Command
			if err := yaml.Unmarshal(content, &cmd); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", entry.Name(), err)
				continue
			}
			// Derive the path prefix from the filename: <host>.<sub.path>.yaml → <sub/path>
			// e.g. git.commit.diff.yaml → commit/diff, git.yaml → "" (root)
			filename := UnmangleFilename(strings.TrimSuffix(entry.Name(), ".yaml"))
			var prefix string
			if filename != host {
				remaining := strings.TrimPrefix(filename, host+".")
				prefix = strings.ReplaceAll(remaining, ".", "/")
			}
			// If the spec has a name, use it to derive the path prefix instead
			if cmd.Name != "" {
				nameParts := strings.Fields(cmd.Name)
				if len(nameParts) > 0 && nameParts[0] == host {
					if len(nameParts) > 1 {
						prefix = strings.Join(nameParts[1:], "/")
					} else {
						prefix = ""
					}
				}
			}
			if err := writeCmdDocs(hostBucket, cmd, prefix); err != nil {
				return fmt.Errorf("%s: %w", entry.Name(), err)
			}
		}
		return nil
	})
}

// CompileStatic compiles a static scheme doc (map[string]string YAML) into a bbolt DB.
// If a value starts with "./", it's resolved as a file path relative to the YAML file's directory.
func CompileStatic(yamlPath, scheme, host, dbPath string, opts ...CompileOption) error {
	return compileStaticFile(yamlPath, scheme, host, dbPath, newCompileOptions(opts))
}

func compileStaticFile(yamlPath, scheme, host, dbPath string, o compileOptions) error {
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		return err
	}
	var m map[string]string
	if err := yaml.Unmarshal(content, &m); err != nil {
		return err
	}
	baseDir := filepath.Dir(yamlPath)
	return compileStaticMap(m, scheme, host, dbPath, baseDir, o)
}

// CompileManDir compiles an entire man/ directory tree into a single bbolt DB.
// man/cmd/<host>/*.yaml → cmd scheme, each subdirectory = one host
// man/<scheme>/<host>/<host>.yaml → static scheme, each file = one host
func CompileManDir(manDir, dbPath string, opts ...CompileOption) error {
	o := newCompileOptions(opts)
	if err := truncateDB(dbPath, o.amend); err != nil {
		return err
	}
	entries, err := os.ReadDir(manDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scheme := entry.Name()
		schemeDir := filepath.Join(manDir, scheme)

		if scheme == "cmd" {
			hosts, err := os.ReadDir(schemeDir)
			if err != nil {
				return fmt.Errorf("cmd: %w", err)
			}
			for _, hostDir := range hosts {
				if !hostDir.IsDir() {
					continue
				}
				hostPath := filepath.Join(schemeDir, hostDir.Name())
				if err := compileHostDir(hostPath, dbPath, o); err != nil {
					return fmt.Errorf("cmd/%s: %w", hostDir.Name(), err)
				}
			}
		} else {
			hosts, err := os.ReadDir(schemeDir)
			if err != nil {
				return fmt.Errorf("%s: %w", scheme, err)
			}
			for _, hostDir := range hosts {
				if !hostDir.IsDir() {
					continue
				}
				host := hostDir.Name()
				yamlFile := filepath.Join(schemeDir, host, host+".yaml")
				if _, err := os.Stat(yamlFile); err != nil {
					continue
				}
				if err := compileStaticFile(yamlFile, scheme, host, dbPath, o); err != nil {
					return fmt.Errorf("%s/%s: %w", scheme, host, err)
				}
			}
		}
	}
	return nil
}

func compileCmd(cmd command.Command, host, dbPath string, o compileOptions) error {
	db, err := bolt.Open(dbPath, 0o644, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		schemeBucket, err := tx.CreateBucketIfNotExists([]byte("cmd"))
		if err != nil {
			return err
		}
		hostBucket, err := createHostBucket(schemeBucket, host, o)
		if err != nil {
			return err
		}
		return writeCmdDocs(hostBucket, cmd, "")
	})
}

// createHostBucket replaces the given host bucket so that a recompiled host
// starts without stale entries; other hosts are left untouched.
func createHostBucket(schemeBucket *bolt.Bucket, host string, o compileOptions) (*bolt.Bucket, error) {
	if o.amend {
		if previous := schemeBucket.Bucket([]byte(host)); previous != nil {
			if err := schemeBucket.DeleteBucket([]byte(host)); err != nil {
				return nil, err
			}
		}
	}
	return schemeBucket.CreateBucketIfNotExists([]byte(host))
}

func compressedValue(value []byte) []byte {
	if len(value) == 0 {
		return value
	}
	compressed := dbModel.Compress(value)
	if len(compressed) < len(value) {
		return compressed
	}
	return value
}

func putCompressed(bucket *bolt.Bucket, key string, value string) error {
	return bucket.Put([]byte(key), compressedValue([]byte(value)))
}

func writeCmdDocs(bucket *bolt.Bucket, cmd command.Command, prefix string) error {
	for _, sub := range cmd.Commands {
		subPrefix := prefix
		if subPrefix != "" {
			subPrefix += "/"
		}
		subPrefix += strings.SplitN(sub.Name, " ", 2)[0]
		if err := writeCmdDocs(bucket, sub, subPrefix); err != nil {
			return err
		}
	}

	keyPrefix := prefix
	if keyPrefix != "" {
		keyPrefix += "/"
	}

	if cmd.Documentation.Command != "" {
		if err := putCompressed(bucket, keyPrefix+"command", cmd.Documentation.Command); err != nil {
			return err
		}
	}
	for flag, doc := range cmd.Documentation.Flag {
		if doc != "" {
			if err := putCompressed(bucket, keyPrefix+"flag/"+flag, doc); err != nil {
				return err
			}
		}
	}
	for i, doc := range cmd.Documentation.Positional {
		if doc != "" {
			if err := putCompressed(bucket, keyPrefix+fmt.Sprintf("positional/%d", i), doc); err != nil {
				return err
			}
		}
	}
	if cmd.Documentation.PositionalAny != "" {
		if err := putCompressed(bucket, keyPrefix+"positionalany", cmd.Documentation.PositionalAny); err != nil {
			return err
		}
	}
	for i, doc := range cmd.Documentation.Dash {
		if doc != "" {
			if err := putCompressed(bucket, keyPrefix+fmt.Sprintf("dash/%d", i), doc); err != nil {
				return err
			}
		}
	}
	if cmd.Documentation.DashAny != "" {
		if err := putCompressed(bucket, keyPrefix+"dashany", cmd.Documentation.DashAny); err != nil {
			return err
		}
	}
	for name, doc := range cmd.Examples {
		if doc != "" {
			if err := putCompressed(bucket, keyPrefix+"example/"+name, doc); err != nil {
				return err
			}
		}
	}
	return nil
}

func compileStaticMap(m map[string]string, scheme, host, dbPath, baseDir string, o compileOptions) error {
	db, err := bolt.Open(dbPath, 0o644, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		schemeBucket, err := tx.CreateBucketIfNotExists([]byte(scheme))
		if err != nil {
			return err
		}
		hostBucket, err := createHostBucket(schemeBucket, host, o)
		if err != nil {
			return err
		}
		for key, value := range m {
			if value == "" {
				continue
			}
			if suffix, ok := strings.CutPrefix(value, "./"); ok {
				resolved := filepath.Join(baseDir, suffix)
				content, err := os.ReadFile(resolved)
				if err != nil {
					return fmt.Errorf("resolve %s: %w", value, err)
				}
				if err := hostBucket.Put([]byte(key), compressedValue(content)); err != nil {
					return err
				}
			} else {
				if err := hostBucket.Put([]byte(key), compressedValue([]byte(value))); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
