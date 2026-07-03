package man

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/carapace-sh/carapace-spec/pkg/command"
	bolt "go.etcd.io/bbolt"
	"gopkg.in/yaml.v3"
)

// CompileSpec compiles a carapace-spec command spec into a bbolt DB.
// The host is derived from the spec's root command name (first word).
// If the spec has no name, host is derived from the filename (without .yaml).
func CompileSpec(specPath, dbPath string) error {
	content, err := os.ReadFile(specPath)
	if err != nil {
		return err
	}
	var cmd command.Command
	if err := yaml.Unmarshal(content, &cmd); err != nil {
		return err
	}
	host, _, _ := strings.Cut(cmd.Name, " ")
	if host == "" {
		host = strings.TrimSuffix(filepath.Base(specPath), ".yaml")
	}
	return compileCmd(cmd, host, dbPath)
}

// CompileSpecReader compiles a spec from a reader (e.g. stdin) into a bbolt DB.
// If host is empty, it's derived from the spec's root command name.
func CompileSpecReader(r io.Reader, host, dbPath string) error {
	content, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	var cmd command.Command
	if err := yaml.Unmarshal(content, &cmd); err != nil {
		return err
	}
	if host == "" {
		host, _, _ = strings.Cut(cmd.Name, " ")
	}
	return compileCmd(cmd, host, dbPath)
}

// CompileSpecDir compiles all *.yaml specs in a directory into a single bbolt DB.
// Each file is a separate host (filename without .yaml = host name).
func CompileSpecDir(specDir, dbPath string) error {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		specPath := filepath.Join(specDir, entry.Name())
		if err := CompileSpec(specPath, dbPath); err != nil {
			return fmt.Errorf("%s: %w", entry.Name(), err)
		}
	}
	return nil
}

// CompileHostDir compiles all *.yaml split spec files in a directory into a
// single host bucket in a bbolt DB. The host name is the directory name.
// This handles the man/cmd/<host>/ layout where each file is a partial spec
// for one subcommand (e.g. git.commit.yaml, git.log.yaml) plus the root
// (git.yaml). The path prefix for each file is derived from the filename
// (the part after <host>. with dots replaced by slashes), or from the spec's
// name field if present.
func CompileHostDir(hostDir, dbPath string) error {
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
		hostBucket, err := schemeBucket.CreateBucketIfNotExists([]byte(host))
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
			filename := strings.TrimSuffix(entry.Name(), ".yaml")
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
func CompileStatic(yamlPath, scheme, host, dbPath string) error {
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		return err
	}
	var m map[string]string
	if err := yaml.Unmarshal(content, &m); err != nil {
		return err
	}
	baseDir := filepath.Dir(yamlPath)
	return compileStaticMap(m, scheme, host, dbPath, baseDir)
}

// CompileManDir compiles an entire man/ directory tree into a single bbolt DB.
// man/cmd/<host>/*.yaml → cmd scheme, each subdirectory = one host
// man/<scheme>/<host>/<host>.yaml → static scheme, each file = one host
func CompileManDir(manDir, dbPath string) error {
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
				if err := CompileHostDir(hostPath, dbPath); err != nil {
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
				if err := CompileStatic(yamlFile, scheme, host, dbPath); err != nil {
					return fmt.Errorf("%s/%s: %w", scheme, host, err)
				}
			}
		}
	}
	return nil
}

func compileCmd(cmd command.Command, host, dbPath string) error {
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
		hostBucket, err := schemeBucket.CreateBucketIfNotExists([]byte(host))
		if err != nil {
			return err
		}
		return writeCmdDocs(hostBucket, cmd, "")
	})
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
		if err := bucket.Put([]byte(keyPrefix+"command"), []byte(cmd.Documentation.Command)); err != nil {
			return err
		}
	}
	for flag, doc := range cmd.Documentation.Flag {
		if doc != "" {
			if err := bucket.Put([]byte(keyPrefix+"flag/"+flag), []byte(doc)); err != nil {
				return err
			}
		}
	}
	for i, doc := range cmd.Documentation.Positional {
		if doc != "" {
			if err := bucket.Put([]byte(keyPrefix+fmt.Sprintf("positional/%d", i)), []byte(doc)); err != nil {
				return err
			}
		}
	}
	if cmd.Documentation.PositionalAny != "" {
		if err := bucket.Put([]byte(keyPrefix+"positionalany"), []byte(cmd.Documentation.PositionalAny)); err != nil {
			return err
		}
	}
	for i, doc := range cmd.Documentation.Dash {
		if doc != "" {
			if err := bucket.Put([]byte(keyPrefix+fmt.Sprintf("dash/%d", i)), []byte(doc)); err != nil {
				return err
			}
		}
	}
	if cmd.Documentation.DashAny != "" {
		if err := bucket.Put([]byte(keyPrefix+"dashany"), []byte(cmd.Documentation.DashAny)); err != nil {
			return err
		}
	}
	for name, doc := range cmd.Examples {
		if doc != "" {
			if err := bucket.Put([]byte(keyPrefix+"example/"+name), []byte(doc)); err != nil {
				return err
			}
		}
	}
	return nil
}

func compileStaticMap(m map[string]string, scheme, host, dbPath, baseDir string) error {
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
		hostBucket, err := schemeBucket.CreateBucketIfNotExists([]byte(host))
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
				if err := hostBucket.Put([]byte(key), content); err != nil {
					return err
				}
			} else {
				if err := hostBucket.Put([]byte(key), []byte(value)); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
