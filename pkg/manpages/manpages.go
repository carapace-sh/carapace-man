package manpages

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"
)

// Parse converts a man(7) or mdoc(7) roff source to markdown.
func Parse(inputStream io.Reader) (string, error) {
	raw, err := io.ReadAll(inputStream)
	if err != nil {
		return "", err
	}
	input := bytes.NewReader(decodeRoffBytes(raw))
	src, err := newSource(input)
	if err != nil {
		return "", err
	}
	mode := sniffDialect(src.lines)
	engine := newEngine(src, nil)
	w := newWriter(
		func(text string) string {
			return markdownEscape(engine.renderEscapes(engine.interpolate(text)))
		},
		func(text string) string { return markdownEscape(text) },
	)
	if mode == modeMdoc {
		m := newMdoc(w)
		engine.sink = mdocSink{m: m, w: w}
		if err := engine.run(); err != nil {
			return "", err
		}
		m.finish()
	} else {
		engine.sink = w
		if err := engine.run(); err != nil {
			return "", err
		}
		w.flushBlock()
	}
	return collapseBlankRuns(strings.TrimRight(w.out.String(), "\n") + "\n"), nil
}

// collapseBlankRuns normalizes generated markdown to single blank lines
// between blocks, absorbing separator doubling from titles and fences.
func collapseBlankRuns(md string) string {
	for {
		next := strings.ReplaceAll(md, "\n\n\n", "\n\n")
		if next == md {
			return md
		}
		md = next
	}
}

// decodeRoffBytes interprets the page source: valid UTF-8 passes through,
// otherwise the file decodes as legacy ISO-8859-1 (a 1:1 byte-to-rune map),
// which is the historical man page encoding.
func decodeRoffBytes(raw []byte) []byte {
	if utf8.Valid(raw) {
		return stripControlBytes(raw)
	}
	runes := make([]rune, len(raw))
	for i, b := range raw {
		runes[i] = rune(b)
	}
	return stripControlBytes([]byte(string(runes)))
}

// stripControlBytes removes NUL, DEL and C0 control characters except tab,
// newline, and carriage return; the markdown contract is UTF-8 text.
func stripControlBytes(raw []byte) []byte {
	out := make([]byte, 0, len(raw))
	for _, c := range raw {
		switch {
		case c == 0x0b || c == 0x0c:
			out = append(out, '\n')
		case c == 0x09 || c == 0x0d || c == 0x0a:
			out = append(out, c)
		case c < 0x20 || c == 0x7f:
		default:
			out = append(out, c)
		}
	}
	return out
}

// mdocSink routes engine output: text into the writer, unknown macros into
// the mdoc layer.
type mdocSink struct {
	m *mdoc
	w *writer
}

func (s mdocSink) Text(line string) {
	s.w.Text(line)
}

func (s mdocSink) UnknownRequest(name string, args []string, noBreak bool) {
	s.m.macro(name, args, noBreak)
}

func (s mdocSink) Break()      { s.w.Break() }
func (s mdocSink) Space(n int) { s.w.Space(n) }

func (s mdocSink) PreprocessorBlock(name string, body []string) {
	s.w.PreprocessorBlock(name, body)
}
