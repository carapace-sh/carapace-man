package manpages

import (
	"strings"
	"testing"
)

func TestLinkifySeeAlso(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".TH FOO 1",
		".SH SEE ALSO",
		".BR bar (2),",
		"and",
		".BR baz (3p)",
		"plus",
		".BR qux (x)",
	}, "\n"))
	for _, want := range []string{
		"[**bar**(2)](man://bar/2)",
		"[**baz**(3p)](man://baz/3p)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "**qux**(x)") {
		t.Errorf("invalid section was linkified:\n%s", out)
	}
}

func TestLinkifyInNameRegion(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".TH FOO 1",
		`.BR foo (1)`,
		"detail goes here",
		".SH DESCRIPTION",
		".BR foo (1)",
		"again",
	}, "\n"))
	if !strings.Contains(out, "[**foo**(1)](man://foo/1)") {
		t.Errorf("pre-section content not linkified:\n%s", out)
	}
	if strings.Count(out, "(man://foo/1)") != 1 {
		t.Errorf("linked outside the pre-section zone:\n%s", out)
	}
}

func TestLinkifyPlainProse(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".TH FOO 1",
		".SH DESCRIPTION",
		"See bar(2) and foo(3p) maybe.",
	}, "\n"))
	if strings.Contains(out, "man://") {
		t.Errorf("linkified outside the xref zone:\n%s", out)
	}
}

func TestEscapedProseIsMarkdownSafe(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".TH FOO 1",
		".SH DESCRIPTION",
		"angle <brackets>, backslash \\e and literal #tick.",
	}, "\n"))
	for _, want := range []string{"&lt;brackets&gt;", "\\\\", "#tick."} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestEscapeRunsJoinWithoutSpace(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".TH FOO 1",
		".SH DESCRIPTION",
		`.B "a\c"`,
		"b",
	}, "\n"))
	if !strings.Contains(out, "**a**b") {
		t.Errorf("join missing: %q", out)
	}
}

func TestLineStartHeads(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".TH FOO 1",
		".SH DESCRIPTION",
		".P",
		"- a dash option",
	}, "\n"))
	if !strings.Contains(out, "\\- a dash") {
		t.Errorf("dash not escaped: %q", out)
	}
}
