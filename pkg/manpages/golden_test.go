package manpages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGolden compares Parse output against checked-in markdown goldens in
// testdata/golden. Set MANPAGES_GOLDEN_WRITE=1 to regenerate them.
func TestGolden(t *testing.T) {
	roffs, err := filepath.Glob(filepath.Join("testdata", "golden", "*.roff"))
	if err != nil {
		t.Fatal(err)
	}
	if len(roffs) == 0 {
		t.Fatal("no golden fixtures")
	}
	write := os.Getenv("MANPAGES_GOLDEN_WRITE") == "1"
	for _, roff := range roffs {
		roff := roff
		t.Run(strings.TrimSuffix(filepath.Base(roff), ".roff"), func(t *testing.T) {
			source, err := os.ReadFile(roff)
			if err != nil {
				t.Fatal(err)
			}
			out, err := Parse(strings.NewReader(string(source)))
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			golden := strings.TrimSuffix(roff, ".roff") + ".md"
			if write {
				if err := os.WriteFile(golden, []byte(out), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("missing golden %s (set MANPAGES_GOLDEN_WRITE=1): %v", golden, err)
			}
			if got, want := out, string(want); got != want {
				t.Errorf("golden mismatch\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
