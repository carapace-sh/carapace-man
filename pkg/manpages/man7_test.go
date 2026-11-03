package manpages

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, src string) string {
	t.Helper()
	out, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return out
}

func contains(t *testing.T, src string, want ...string) {
	t.Helper()
	out := mustParse(t, src)
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in:\n%s", w, out)
		}
	}
}

func page(lines ...string) string {
	return strings.Join(lines, "\n")
}

func TestTitleHeading(t *testing.T) {
	contains(t, page(
		`.TH FOO 1 "2026-03-14" "foo 2.1" "User Commands"`,
		".SH NAME",
		`foo \- do something useful`,
	), "# FOO(1)", "<!-- date: 2026-03-14; source: foo 2.1 -->", "## NAME", "foo - do something useful")
}

func TestParagraphsAndSubHeading(t *testing.T) {
	contains(t, page(
		".TH FOO 1",
		".SH DESCRIPTION",
		"First paragraph.",
		".PP",
		"Second paragraph.",
		".SS Details",
		"Detail text.",
	),
		"## DESCRIPTION",
		"First paragraph.",
		"Second paragraph.",
		"### Details",
		"Detail text.")
}

func TestBulletsAndOrderedItems(t *testing.T) {
	contains(t, page(
		".TH FOO 1",
		".SH DESCRIPTION",
		".TP",
		".B first tag",
		"description body",
		`.IP "\(bu"`,
		"bullet item",
		`.IP "\(bu"`,
		"second bullet",
		`.IP "1."`,
		"ordered item",
	),
		"**first tag**",
		"description body",
		"- bullet item",
		"- second bullet",
		"1. ordered item")
}

func TestFontMacros(t *testing.T) {
	contains(t, page(
		".TH FOO 1",
		".SH DESCRIPTION",
		".B bold text",
		"plain followup",
		".BR foo (1),",
		"ref trailing",
		".IR file ...",
	),
		"**bold text** plain followup",
		"**foo**(1), ref trailing",
		"*file*...")
}

func TestCodeBlocks(t *testing.T) {
	contains(t, page(
		".TH FOO 1",
		".SH DESCRIPTION",
		".EX",
		"command line 1",
		"command line 2",
		".EE",
		".PP",
		"after code",
	),
		"command line 1\ncommand line 2\n```",
		"after code")
}

func TestSynopsisBlocks(t *testing.T) {
	contains(t, page(
		".TH FOO 1",
		".SH SYNOPSIS",
		".SY foo",
		".OP \\-v",
		".OP \\-o file",
		".Ys",
	),
		"**foo**",
		"[**-v**]",
		"[**-o** *file*]")
}

func TestUserMacroVbVe(t *testing.T) {
	contains(t, page(
		".de Vb",
		".ft CW",
		".nf",
		"..",
		".de Ve",
		".ft R",
		".fi",
		"..",
		".TH FOO 1",
		".SH EXAMPLES",
		".Vb 2",
		"raw code line",
		".Ve",
		"after",
	),
		"raw code line\n```",
		"after")
}

func TestLinkMacros(t *testing.T) {
	contains(t, page(
		".TH FOO 1",
		".SH BUGS",
		".UR https://example.org",
		"the project page",
		".UE .",
		".MT mail@example.org",
		"mail me",
		".ME",
	),
		"[the project page](https://example.org)",
		"[mail me](mailto:mail@example.org)")
}

func TestHyperlinkMR(t *testing.T) {
	contains(t, page(
		".TH FOO 1",
		".SH SEE ALSO",
		".MR bar 1",
	), "bar(1)")
}

func TestPod2manStylePage(t *testing.T) {
	contains(t, page(
		".de Sp \" Vertical space",
		".if t .sp .5v",
		".if n .sp",
		"..",
		".de IX",
		"..",
		`.TH PERLPOD 1 "2025-04-12" "perl 5.0"`,
		".SH NAME",
		`perlpod \- the Plain Old Documentation format`,
		".SH DESCRIPTION",
		`.IX Header "DESCRIPTION"`,
		".PP",
		"Text about the format.",
		`.IP "\(bu 4"`,
		"=pod item",
		".RS",
		`.IP "\(bu 4"`,
		"nested item",
		".RE",
	),
		"# PERLPOD(1)",
		"perlpod - the Plain Old Documentation format",
		"Text about the format.",
		"- =pod item",
		"- nested item")
}
