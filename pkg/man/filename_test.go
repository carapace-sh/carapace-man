package man

import "testing"

func TestRoundTrip(t *testing.T) {
	names := []string{
		"",
		"hg",
		"hg.debug::clonebundle-manifest",
		"hg.debug::stable-tail-sort-leaps",
		"cmd.exe",
		"<>*?\"\\|colon:test%",
		"%253C%253E",
	}
	for _, name := range names {
		if got := UnmangleFilename(MangleFilename(name)); got != name {
			t.Errorf("UnmangleFilename(MangleFilename(%q)) = %q", name, got)
		}
	}
}

func TestMangleFilename(t *testing.T) {
	if got := MangleFilename("hg.debug::clonebundle-manifest"); got != "hg.debug%3A%3Aclonebundle-manifest" {
		t.Errorf("unexpected mangled name: %q", got)
	}
}
