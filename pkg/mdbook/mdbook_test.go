package mdbook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeArchive(t *testing.T, archive string, files map[string]string) {
	for name, content := range files {
		path := filepath.Join(archive, "man1", name)
		if strings.Contains(content, "\x00") {
			// not used
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGenerateLayout(t *testing.T) {
	archive := t.TempDir()
	book := t.TempDir()
	writeArchive(t, archive, map[string]string{
		"ls.1": strings.Join([]string{".TH LS 1", ".SH SEE ALSO", ".BR du (1),", ".BR missing (8)", ""}, "\n"),
		"du.1": ".TH DU 1\n.SH NAME\ndu \\- disk usage\n",
	})
	if err := Generate(archive, book); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(book, "src", "ls(1).md"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(page)
	if !strings.Contains(out, "[**du**(1)](<du(1).md>)") {
		t.Errorf("rewritten link missing:\n%s", out)
	}
	if !strings.Contains(out, "man://missing/8") {
		t.Errorf("unresolved target not preserved:\n%s", out)
	}
	for _, expected := range []string{"book.toml", filepath.Join("src", "SUMMARY.md"), filepath.Join("src", "ls(1).md"), filepath.Join("src", "du(1).md"), "report.txt"} {
		if _, err := os.Stat(filepath.Join(book, expected)); err != nil {
			t.Errorf("missing %v: %v", expected, err)
		}
	}
	summary, err := os.ReadFile(filepath.Join(book, "src", "SUMMARY.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(summary), "- [du(1)](<du(1).md>)") || !strings.Contains(string(summary), "- [ls(1)](<ls(1).md>)") {
		t.Errorf("summary pages wrong:\n%s", summary)
	}
	report, err := os.ReadFile(filepath.Join(book, "report.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(report), "missing link missing(8) in ls(1)") {
		t.Errorf("missing link not reported:\n%s", report)
	}
}

func TestGenerateCleanReport(t *testing.T) {
	archive := t.TempDir()
	book := t.TempDir()
	writeArchive(t, archive, map[string]string{"du.1": ".TH DU 1\n"})
	if err := Generate(archive, book); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(book, "report.txt")); err == nil {
		t.Error("report.txt should be absent when nothing failed")
	}
}
