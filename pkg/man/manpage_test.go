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
