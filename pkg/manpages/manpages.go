package manpages

import (
	"fmt"
	"io"
	"strings"
)

// Parse converts a man(7) or mdoc(7) roff source to markdown. It returns an
// error for mdoc(7) input until the mdoc macro layer is implemented.
func Parse(input io.Reader) (string, error) {
	src, err := newSource(input)
	if err != nil {
		return "", err
	}
	mode := sniffDialect(src.lines)
	if mode == modeMdoc {
		return "", fmt.Errorf("manpages: mdoc(7) input is not supported yet")
	}
	engine := newEngine(src, nil)
	w := newWriter(func(text string) string {
		return engine.renderEscapes(engine.interpolate(text))
	})
	engine.sink = w
	if err := engine.run(); err != nil {
		return "", err
	}
	w.flushBlock()
	return strings.TrimRight(w.out.String(), "\n") + "\n", nil
}
