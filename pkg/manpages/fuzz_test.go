package manpages

import (
	"strings"
	"testing"
)

// FuzzParse drives the full pipeline end-to-end on random roff input; the
// property is that conversion never panics and, on success, leaves no raw
// directive lines outside code fences.
func FuzzParse(f *testing.F) {
	seeds := []string{
		".TH FOO 1\n.SH NAME\nfoo \\- desc\n",
		".TH FOO 1\n.SH OPTIONS\n.TP\n.B \\-v\nverbose\n.IP \"\\(bu\"\nbullet\n",
		".de Vb\n.ft CW\n.nf\n..\n.TH FOO 1\n.Vb 2\ncode\n.Ve\n",
		".if n \\{\n.ds x yes\n'br\\}\n\\*[x]\n",
		".Dd $Mdocdate$\n.Dt FOO 1\n.Os\n.Sh NAME\n.Nm foo\n.Nd desc\n.Sh DESCRIPTION\n.It\nlist\n",
		".TS\ntab(;);\nCb Cb .\na;b\n.TE\n",
		"\n.\n\n\\\n\n.nr x\nxx\\.c y\n",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		out, err := Parse(strings.NewReader(input))
		if err != nil {
			return
		}
		if leaked := directiveLeaksOutsideFences(out); len(leaked) > 0 {
			t.Fatalf("directive leak %q for input %q", leaked[0], input)
		}
	})
}

// FuzzSplitArgs checks the tokenizer on random argument lines.
func FuzzSplitArgs(f *testing.F) {
	for _, seed := range []string{"", "a b c", `"quoted str"`, `'s "q""`, `""`, `a\ b\ c "x"`, `\ \"`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		splitArgs(input)
	})
}
