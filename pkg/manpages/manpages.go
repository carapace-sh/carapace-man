package manpages

import (
	"io"
	"strings"
)

// Parse converts a man(7) or mdoc(7) roff source to markdown.
func Parse(input io.Reader) (string, error) {
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
	return strings.TrimRight(w.out.String(), "\n") + "\n", nil
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

func (s mdocSink) PreprocessorBlock(name string, body []string) {
	s.w.PreprocessorBlock(name, body)
}
