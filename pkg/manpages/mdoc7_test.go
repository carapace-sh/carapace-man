package manpages

import (
	"strings"
	"testing"
)

func TestMdocBasicPage(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		`.Dd $Mdocdate$`,
		".Dt FOO 1",
		".Os",
		".Sh NAME",
		".Nm foo",
		".Nd do something useful",
		".Sh DESCRIPTION",
		"The",
		".Nm",
		"utility does it.",
		".Sh SEE ALSO",
		".Xr bar 1",
	}, "\n"))
	for _, want := range []string{
		"# FOO(1)",
		"## NAME",
		"**foo** - do something useful",
		"## DESCRIPTION",
		"**foo** utility does it.",
		"## SEE ALSO",
		"bar(1)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestMdocFlagsAndArguments(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".Dd FOO",
		".Dt FOO 8",
		".Os",
		".Sh SYNOPSIS",
		".Nm foo",
		".Op Fl v",
		".Op Fl o Ar file",
		".Sh DESCRIPTION",
		".Bl -tag -width Ds",
		".It Fl v",
		"Print verbose output.",
		".It Fl o Ar file",
		"Write output to",
		".Pa file.txt",
		"instead.",
		".El",
	}, "\n"))
	for _, want := range []string{
		"**-v**",
		"[**-v**]",
		"[**-o** *file*]",
		"## SYNOPSIS",
		"**-v**",
		"*file*",
		"*file.txt*",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestMdocLists(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".Dd FOO",
		".Dt FOO 1",
		".Os",
		".Sh DESCRIPTION",
		".Bl -bullet -compact",
		".It",
		"one bullet",
		".It",
		"second bullet",
		".El",
		".Bl -enum",
		".It",
		"first",
		".It",
		"second",
		".El",
	}, "\n"))
	for _, want := range []string{
		"- one bullet",
		"- second bullet",
		"1. first",
		"2. second",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestMdocDisplaysAndEnclosures(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".Dd FOO",
		".Dt FOO 1",
		".Os",
		".Sh DESCRIPTION",
		"Run the command",
		`.D1 Fl a`,
		"and see",
		`.Dl % mandoc foo.1 | less`,
		"for output.",
		".Bd -literal",
		"code line",
		"second line",
		".Ed",
		"Plain",
		".Ic italic",
		"and",
		".Dq quoted",
		"words.",
	}, "\n"))
	for _, want := range []string{
		"> **-a**",
		"",
		"```\ncode line\nsecond line\n```",
		"**italic**",
		"“quoted”",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestMdocSynopsisLiterals(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".Dd FOO",
		".Dt FOO 3",
		".Os",
		".Sh SYNOPSIS",
		".In stdio.h",
		".Ft int",
		".Fn foo int flag",
		".Sh DESCRIPTION",
		"Details.",
	}, "\n"))
	for _, want := range []string{
		"stdio.h",
		"*int*",
		"**foo**",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestMdocColumnFallback(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".Dd FOO",
		".Dt FOO 1",
		".Os",
		".Sh DESCRIPTION",
		".Bl -column key value",
		".It key Ta value",
		"It key Ta value",
		".El",
	}, "\n"))
	if !strings.Contains(out, "key   value") {
		t.Errorf("column rendering missing: %q", out)
	}
}

func TestMdocLink(t *testing.T) {
	out := mustParse(t, strings.Join([]string{
		".Dd FOO",
		".Dt FOO 1",
		".Os",
		".Sh DESCRIPTION",
		"See",
		".Lk https://example.org the site",
		"or mail",
		".Mt user@example.org",
		".",
	}, "\n"))
	for _, want := range []string{
		"[the site](https://example.org)",
		"[user@example.org](mailto:user@example.org)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
