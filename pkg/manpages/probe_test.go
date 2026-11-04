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

func parseOne2(t *testing.T, file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var reader io.Reader = f
	magic := make([]byte, 2)
	if n, _ := io.ReadFull(f, magic); n == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		gz, err := gzip.NewReader(f)
		if err != nil {
			return "", err
		}
		defer gz.Close()
		reader = gz
	}
	return Parse(reader)
}

func directiveLeaksOutsideFences(md string) []string {
	var leaks []string
	inFence := false
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(lines(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, directive := range []string{"\\n.TP ", "??SH", "\\.SH ", "\\.IP ", "\\.PP", "\\.TH ", "\\.nf", "\\.fi", "\\.B ", "\\.I ", "\\.RS", "\\.RE"} {
			if strings.HasPrefix(line, directive) {
				leaks = append(leaks, line)
				break
			}
		}
	}
	return leaks
}

func lines(line string) string { return line }

func TestProbeFullCorpus(t *testing.T) {
	if os.Getenv("MANPAGES_PROBE") != "1" {
		t.Skip("full corpus probe; set MANPAGES_PROBE=1")
	}
	if testing.Short() {
		t.Skip("short")
	}
	root := "/usr/share/man"
	var files []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	fmt.Printf("TOTAL %d pages\n", len(files))
	failures := 0
	errExamples := map[string][]string{}
	leakPages := 0
	leakLineExamples := map[string][]string{}
	for _, file := range files {
		out, err := parseOne2(t, file)
		base := filepath.Base(file)
		if err != nil {
			failures++
			key := strings.SplitN(err.Error(), ":", 2)[0]
			if len(errExamples[key]) < 4 {
				errExamples[key] = append(errExamples[key], base)
			}
			continue
		}
		leaked := directiveLeaksOutsideFences(out)
		if len(leaked) > 0 {
			leakPages++
			if len(leakLineExamples[base]) < 2 {
				leakLineExamples[base] = leaked[:1]
			}
		}
	}
	fmt.Printf("failures: %d\n", failures)
	for k, v := range errExamples {
		fmt.Printf("  %s: %v\n", k, v)
	}
	fmt.Printf("leak pages: %d\n", leakPages)
	for k, v := range leakLineExamples {
		fmt.Printf("  %s: %v\n", k, v)
	}
}
