package manpages

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCorpusNoPanic(t *testing.T) {
	root := os.Getenv("MANPAGES_CORPUS")
	if root == "" {
		root = "/usr/share/man"
	}
	if _, err := os.Stat(root); err != nil {
		t.Skipf("no manpage corpus at %v: %v", root, err)
	}
	var files []string
	limit := 300
	stop := false
	for !stop {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".gz") || strings.HasSuffix(strings.TrimSuffix(path, ".gz"), "1") {
				files = append(files, path)
			} else {
				files = append(files, path)
			}
			if len(files) >= limit {
				stop = true
				return filepath.SkipAll
			}
			return nil
		})
		stop = true
	}
	if len(files) == 0 {
		t.Skip("no manpage files found")
	}
	t.Logf("corpus files: %d", len(files))
	for i, file := range files {
		file := file
		t.Run(fmt.Sprintf("page%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on %v: %v", file, r)
				}
			}()
			f, err := os.Open(file)
			if err != nil {
				t.Skipf("open: %v", err)
			}
			defer f.Close()
			var reader io.Reader = f
			if strings.HasSuffix(file, ".gz") {
				gz, gerr := gzip.NewReader(f)
				if gerr != nil {
					t.Skipf("gzip: %v", gerr)
				}
				defer gz.Close()
				reader = gz
			}
			if _, err := Parse(reader); err != nil && !strings.Contains(err.Error(), "macro recursion limit") {
				t.Logf("parse error %v: %v", file, err)
			}
		})
	}
}
