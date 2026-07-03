package man

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carapace-sh/carapace/pkg/xdg"
	bolt "go.etcd.io/bbolt"
)

// dbLocation returns the directory where compiled .db files are searched.
// Uses $XDG_CONFIG_HOME/carapace/man/db/ (alongside the YAML man/ tree).
// Can be overridden with CARAPACE_MAN_DB_PATH (colon-separated paths).
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

// hostIndex maps (scheme, host) → db file path.
type hostIndex map[string]string // key = scheme + "\x00" + host

func indexKey(scheme, host string) string {
	return scheme + "\x00" + host
}

// buildHostIndex scans all .db files in the search paths and builds
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
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".db") {
				dbPaths = append(dbPaths, filepath.Join(sp, entry.Name()))
			}
		}
	}
	sort.Strings(dbPaths)

	index := hostIndex{}
	for _, dbPath := range dbPaths {
		db, err := bolt.Open(dbPath, 0o400, &bolt.Options{ReadOnly: true})
		if err != nil {
			continue
		}
		err = db.View(func(tx *bolt.Tx) error {
			return tx.ForEach(func(schemeName []byte, schemeBucket *bolt.Bucket) error {
				return schemeBucket.ForEachBucket(func(hostName []byte) error {
					k := indexKey(string(schemeName), string(hostName))
					if _, exists := index[k]; !exists {
						index[k] = dbPath
					}
					return nil
				})
			})
		})
		db.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", dbPath, err)
		}
	}
	return index, nil
}

// lookupDB opens the DB that contains (scheme, host) read-only and runs fn
// against the host bucket. Returns false if no DB contains the (scheme, host).
func lookupDB(scheme, host string, fn func(*bolt.Bucket) error) (bool, error) {
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

// lookupDBFile opens a specific DB file read-only and runs fn against the
// (scheme, host) host bucket. Returns false if the scheme or host bucket
// doesn't exist in the DB.
func lookupDBFile(dbPath, scheme, host string, fn func(*bolt.Bucket) error) (bool, error) {
	db, err := bolt.Open(dbPath, 0o400, &bolt.Options{ReadOnly: true})
	if err != nil {
		return false, err
	}
	defer db.Close()

	var found bool
	err = db.View(func(tx *bolt.Tx) error {
		schemeBucket := tx.Bucket([]byte(scheme))
		if schemeBucket == nil {
			return nil
		}
		hostBucket := schemeBucket.Bucket([]byte(host))
		if hostBucket == nil {
			return nil
		}
		found = true
		return fn(hostBucket)
	})
	return found, err
}

// cmdDB looks up a cmd:// UID from compiled bbolt DBs.
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
	found, err := lookupDB("cmd", uid.Host, func(bucket *bolt.Bucket) error {
		val := bucket.Get([]byte(key))
		if val != nil {
			result = string(val)
		}
		return nil
	})
	if err != nil {
		return "", false, err
	}
	if !found || result == "" {
		return "", false, nil
	}
	return result, true, nil
}

// describeDB looks up a static scheme UID from compiled bbolt DBs.
// Returns ("", false, nil) if not found in any DB (fall through to YAML).
func describeDB(uid *url.URL) (string, bool, error) {
	key := strings.TrimPrefix(uid.Path, "/")

	var result string
	found, err := lookupDB(uid.Scheme, uid.Host, func(bucket *bolt.Bucket) error {
		val := bucket.Get([]byte(key))
		if val != nil {
			result = string(val)
		}
		return nil
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
// For cmd scheme: keys ending in "/command" (or "command" for root) yield
// command UIDs; keys containing "/flag/" yield ?flag= UIDs.
// For static schemes: all keys yield UIDs with the key as the path.
func UidsFromDBs(scheme, host string) ([]*url.URL, error) {
	var uids []*url.URL
	_, err := lookupDB(scheme, host, func(bucket *bolt.Bucket) error {
		var innerErr error
		uids, innerErr = enumerateUids(bucket, scheme, host)
		return innerErr
	})
	if err != nil {
		return nil, err
	}
	return uids, nil
}

// enumerateUids scans a host bucket and reconstructs UIDs from its keys.
func enumerateUids(bucket *bolt.Bucket, scheme, host string) ([]*url.URL, error) {
	var uids []*url.URL
	c := bucket.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		if v == nil {
			continue
		}
		key := string(k)
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
			} else if field == "command" {
				// no query
			} else {
				continue
			}
			uid, err := url.Parse(uidStr)
			if err != nil {
				return nil, err
			}
			uids = append(uids, uid)
		} else {
			uidStr := fmt.Sprintf("%s://%s/%s", scheme, host, key)
			uid, err := url.Parse(uidStr)
			if err != nil {
				return nil, err
			}
			uids = append(uids, uid)
		}
	}
	return uids, nil
}

// splitCmdKey splits a cmd bucket key into (path, field).
// e.g. "commit/flag/message" → ("commit", "flag/message")
//
//	"command"             → ("", "command")
//	"commit/diff/command" → ("commit/diff", "command")
func splitCmdKey(key string) (path, field string, ok bool) {
	// The field is the last segment if it's "command", or the last two
	// segments if they start with "flag/", "positional/", "dash/", "example/".
	// Otherwise we can't determine the boundary.
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

// InspectSchemes returns all scheme names in a specific DB file.
func InspectSchemes(dbPath string) ([]string, error) {
	db, err := bolt.Open(dbPath, 0o400, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var schemes []string
	err = db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			if string(name) == "_meta" {
				return nil
			}
			schemes = append(schemes, string(name))
			return nil
		})
	})
	sort.Strings(schemes)
	return schemes, err
}

// InspectHosts returns all host names for a scheme in a specific DB file.
func InspectHosts(dbPath, scheme string) ([]string, error) {
	db, err := bolt.Open(dbPath, 0o400, &bolt.Options{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var hosts []string
	err = db.View(func(tx *bolt.Tx) error {
		schemeBucket := tx.Bucket([]byte(scheme))
		if schemeBucket == nil {
			return nil
		}
		return schemeBucket.ForEachBucket(func(hostName []byte) error {
			hosts = append(hosts, string(hostName))
			return nil
		})
	})
	sort.Strings(hosts)
	return hosts, err
}

// InspectUids returns all UIDs for a (scheme, host) in a specific DB file.
func InspectUids(dbPath, scheme, host string) ([]*url.URL, error) {
	var uids []*url.URL
	found, err := lookupDBFile(dbPath, scheme, host, func(bucket *bolt.Bucket) error {
		var innerErr error
		uids, innerErr = enumerateUids(bucket, scheme, host)
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

// InspectDescribe looks up a UID in a specific DB file and returns the doc.
// Returns ("", false, nil) if the key is not present.
func InspectDescribe(dbPath string, uid *url.URL) (string, bool, error) {
	scheme := uid.Scheme
	host := uid.Host

	var result string
	found, err := lookupDBFile(dbPath, scheme, host, func(bucket *bolt.Bucket) error {
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
		val := bucket.Get([]byte(key))
		if val != nil {
			result = string(val)
		}
		return nil
	})
	if err != nil {
		return "", false, err
	}
	return result, found && result != "", nil
}
