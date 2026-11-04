package manpages

import (
	"strings"
	"testing"
)

func TestPreprocessorBlocksPreserved(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".TH FOO 1",
		".SH DESCRIPTION",
		"Before the table.",
		".TS",
		"tab(;);",
		"Cb Cb .",
		"key;value",
		".TE",
		"after the table.",
		".EQ",
		"x over 2a",
		".EN",
		"math done",
	}, "\n"))
	for _, want := range []string{
		"```",
		"key;value",
		"x over 2a",
		"after the table.",
		"math done",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, ".TS\n") || strings.Contains(out, ".EQ\n") {
		t.Errorf("directives leaked: %s", out)
	}
}

func TestPreprocessorEndsByTerminator(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".TH FOO 1",
		".SH DESCRIPTION",
		".TS",
		"row one",
		".TE",
		"plain body",
	}, "\n"))
	if !strings.Contains(out, "row one") || !strings.Contains(out, "plain body") {
		t.Errorf("content lost: %s", out)
	}
}
