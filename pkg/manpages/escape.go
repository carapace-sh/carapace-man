package manpages

import (
	"strings"
)

// markdownEscape hardens text for markdown output: backslashes, angle
// brackets, and the metacharacters that would otherwise reformat the line.
func markdownEscape(text string) string {
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		switch c := text[i]; c {
		case '\\':
			b.WriteString("\\\\")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// escapeLineStart escapes a leading markdown block marker (`#`, `+`, `-`,
// `*`, and integer ordered-list prefixes) on the first line of a paragraph
// or item body.
func escapeLineStart(line string) bool {
	if line == "" {
		return false
	}
	switch line[0] {
	case '#', '+', '-', '*', '=':
		return true
	}
	if line[0] >= '0' && line[0] <= '9' {
		for i := 0; i < len(line) && i < 3; i++ {
			c := line[i]
			if c == '.' || c == ')' {
				return true
			}
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return false
}
