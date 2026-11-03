package man

import (
	"strings"
	"testing"
)

func TestManToMdNative(t *testing.T) {
	out, err := ManToMd("ls", "1")
	if err != nil {
		t.Skipf("man page not available: %v", err)
	}
	if !strings.Contains(out, "ls - list directory contents") {
		t.Errorf("title missing: %q", out[:min(200, len(out))])
	}
	for _, directive := range []string{".TH", ".SH", ".IP", ".PP", ".TP", ".nf"} {
		if strings.Contains(out, directive) {
			t.Errorf("raw directive %q leaked into output", directive)
		}
	}
}

func TestManToMdLegacy(t *testing.T) {
	if _, err := ManToMd("ls", "1"); err != nil {
		t.Skipf("man page not available: %v", err)
	}
	out, err := ManToMdLegacy("ls", "1")
	if err != nil {
		t.Skipf("legacy converter needs perl: %v", err)
	}
	if out == "" {
		t.Error("legacy output empty")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
