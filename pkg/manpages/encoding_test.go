package manpages

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParseLatin1Fallback(t *testing.T) {
	in := []byte{'.', 'T', 'H', ' ', 'X', ' ', '1', '\n', '.', 'S', 'H', ' ', 'N', 'A', 'M', 'E', '\n', 'A', 'l', 'f', 'o', 'n', 's', 'o', ' ', 'G', 'a', 'r', 'c', 0xed, 'a', ' ', 'P', 'a', 't', 'i', 0xf1, 'o', '\n'}
	out, err := Parse(strings.NewReader(string(in)))
	if err != nil {
		t.Fatal(err)
	}
	if !utf8.ValidString(out) {
		t.Errorf("output invalid UTF-8")
	}
	if !strings.Contains(out, "García") {
		t.Errorf("latin-1 not decoded: %q", out)
	}
}

func TestParseStripsControlBytes(t *testing.T) {
	in := []byte{'.', 'T', 'H', ' ', 'X', ' ', '1', '\n', 't', 'e', 'x', 't', 0x00, 0x07, 'a', 'f', 't'}
	out, err := Parse(strings.NewReader(string(in)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(out, "\x00\x07") {
		t.Errorf("control bytes survive: %q", out)
	}
	if !utf8.ValidString(out) {
		t.Errorf("output invalid UTF-8")
	}
}

func TestParseKeepsTabs(t *testing.T) {
	in := []byte{'.', 'T', 'H', ' ', 'X', ' ', '1', '\n', 'a', '\t', 'b', '\n'}
	out, err := Parse(strings.NewReader(string(in)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a\tb") {
		t.Errorf("tab lost: %q", out)
	}
}
