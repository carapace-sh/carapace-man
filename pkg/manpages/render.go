package manpages

import (
	"regexp"
	"strconv"
	"strings"
)

// renderEscapes translates the remaining escape sequences to plain text.
// Font, size, and presentation escapes fall back consistent with the mapping
// decisions in the manpages skill (they carry no value for markdown output).
func (e *Engine) renderEscapes(s string) string {
	for key, value := range e.transl {
		if strings.HasPrefix(key, `\`) {
			s = strings.ReplaceAll(s, key, value)
		}
	}
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			i++
			continue
		}
		c := s[i+1]
		switch c {
		case '\\':
			b.WriteByte('\\')
			i += 2
		case 'e':
			b.WriteByte('\\')
			i += 2
		case '-':
			b.WriteByte('-')
			i += 2
		case '&', '^', '|', '%', ':', '{', '}':
			i += 2
		case ' ', '~', '0':
			b.WriteByte(' ')
			i += 2
		case 't':
			b.WriteByte('\t')
			i += 2
		case 'c':
			b.WriteString(joinMarker)
			i += 2
		case 'p', 'F', 'P', 'Z', 'z', 'k', 'a', 'j', 'g', 'O', 'o', 'b', 'x', 'w', 'h', 'v', 'D':
			i += 2 + escapeSkip(s[i+2:], c)
		case 'f':
			i += 2 + fontSkip(s[i+2:])
		case 's':
			i += 2 + sizeSkip(s[i+2:])
		case 'm', 'M':
			i += 2 + delimitedSkip(s[i+2:])
		case 'n', '*', 'R', '$', 'N', 'V', 'A', 'B':
			// leftover interpolation forms: drop escape and remainder marker
			i += 2 + delimitedSkip(s[i+2:])
		case 'C':
			i += 2 + delimitedSkip(s[i+2:])
		case 'X':
			b.WriteString("X")
			i += 2
		case '`':
			b.WriteByte('`')
			i += 2
		case '!', '?':
			b.WriteByte(c)
			i += 2
		case '(':
			key := s[i+2:]
			if len(key) >= 2 {
				if glyph, ok := glyphs[key[:2]]; ok {
					b.WriteString(glyph)
					i += 4
					continue
				}
			}
			i += 2
		case '[':
			if end := strings.IndexByte(s[i+2:], ']'); end >= 0 {
				key := s[i+2 : i+2+end]
				if glyph, ok := glyphs[key]; ok {
					b.WriteString(glyph)
				} else {
					b.WriteString(numericGlyphReturn(key))
				}
				i += 3 + end
				continue
			}
			i += 2
		default:
			i += 2
		}
	}
	out := b.String()
	out = ttyLinks(out)
	return e.applyCharTranslations(out)
}

// ttyLinks converts groff's OSC-8 device extensions
var ttyLinkPattern = regexp.MustCompile(`X'tty: link ([^']*)'(?s)(.*?)X'tty: link'`)

func ttyLinks(text string) string {
	text = ttyLinkPattern.ReplaceAllString(text, "[$2]($1)")
	return strings.ReplaceAll(text, `X'tty: link'`, "")
}

func numericGlyphReturn(key string) string {
	if glyph, ok := numericGlyph(key); ok {
		return glyph
	}
	return ""
}

// escapeSkip skips escape contents: `'...`-quoted arguments, `...`-bracket
// arguments, or plain continuation for word-forming escapes.
func escapeSkip(rest string, typ byte) int {
	switch typ {
	case 'h', 'v', 'w', 'x', 'l', 'L', 'o', 'Z', 'k', 'D', 'a', 'j', 'g', 'A', 'B', 'b', 'O':
		return delimitedSkip(rest)
	}
	return 0
}

// delimitedSkip skips a `'`-quoted or `[`-`...]`-bracketed escape argument.
func delimitedSkip(rest string) int {
	if rest == "" {
		return 0
	}
	if rest[0] == '\'' {
		if end := strings.IndexByte(rest[1:], '\''); end >= 0 {
			return end + 2
		}
		return len(rest)
	}
	if rest[0] == '[' {
		if end := strings.IndexByte(rest[1:], ']'); end >= 0 {
			return end + 2
		}
		return len(rest)
	}
	if rest[0] == '(' {
		if len(rest) > 2 {
			return 3
		}
		return len(rest)
	}
	return 1
}

func fontSkip(rest string) int {
	if rest == "" {
		return 0
	}
	switch rest[0] {
	case '(':
		if len(rest) >= 3 {
			return 3
		}
		return len(rest)
	case '[':
		return delimitedSkip(rest)
	default:
		return 1
	}
}

func sizeSkip(rest string) int {
	if rest == "" {
		return 0
	}
	switch rest[0] {
	case '(':
		if len(rest) >= 3 {
			return 3
		}
		return len(rest)
	case '[':
		return delimitedSkip(rest)
	case '\'':
		return delimitedSkip(rest)
	case '+', '-':
		i := 1
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' && i < 4 {
			i++
		}
		return i
	case '0':
		return 1
	default:
		i := 0
		for i < len(rest) && i < 3 && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		if i == 0 {
			return 1
		}
		return i
	}
}

// applyCharTranslations applies `.tr` character translation pairs to plain
// text positions.
func (e *Engine) applyCharTranslations(s string) string {
	if len(e.transl) == 0 {
		return s
	}
	var b strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if i+1 < len(runes) && runes[i] == '\\' && runes[i+1] != '\\' && runes[i+1] != '(' && runes[i+1] != '[' {
			key := `\` + string(runes[i+1])
			if value, ok := e.transl[key]; ok {
				b.WriteString(value)
				i++
				continue
			}
		}
		if value, ok := e.transl[string(runes[i])]; ok && len([]rune(value)) == 1 {
			b.WriteString(value)
			continue
		}
		b.WriteRune(runes[i])
	}
	return b.String()
}

func numericGlyph(key string) (string, bool) {
	if strings.HasPrefix(key, "u") {
		code, err := strconv.ParseUint(key[1:], 16, 32)
		if err != nil || code == 0 {
			return "", false
		}
		return string(rune(code)), true
	}
	return "", false
}

var glyphs = map[string]string{
	"em": "—",
	"aq": "'",
	"ga": "`",
	"bu": "•",
	"co": "©",
	"rg": "®",
	"ti": "~",
	"ha": "^",
	"mi": "-",
	"hy": "-",
	"lq": "“",
	"rq": "”",
	"dq": `"`,
	"rs": `\`,
	"bs": `\`,
	"lb": "[",
	"rb": "]",
	"lc": "|",
	"rc": "|",
	"ba": "|",
	"lt": "<",
	"gt": ">",
	"ul": "_",
	"ru": "_",
	"en": "–",
	"tm": "™",
	"sc": "§",
	"ff": "ff",
	"fi": "fi",
	"fl": "fl",
	"SP": " ",
	"sp": " ",
}

// parseTranslationPairs parses `.tr` pairs. From and to tokens may be plain
// characters or escape-sequence texts.
func parseTranslationPairs(s string) [][2]string {
	var pairs [][2]string
	runes := []rune(s)
	i := 0
	readOne := func() (string, bool) {
		if i >= len(runes) {
			return "", false
		}
		if runes[i] != '\\' {
			token := string(runes[i])
			i++
			return token, true
		}
		rest := string(runes[i:])
		if strings.HasPrefix(rest, `\\`) || strings.HasPrefix(rest, `\e`) {
			token := string(runes[i : i+2])
			i += 2
			return token, true
		}
		if strings.HasPrefix(rest, `\[`) {
			end := strings.Index(rest, `]`)
			if end > 1 {
				token := string(runes[i : i+end+1])
				i += end + 1
				return token, true
			}
		}
		if strings.HasPrefix(rest, `\*(`) || strings.HasPrefix(rest, `\n(`) {
			if len(runes) >= i+4 {
				token := string(runes[i : i+4])
				i += 4
				return token, true
			}
		}
		if strings.HasPrefix(rest, `\*`) || strings.HasPrefix(rest, `\n`) {
			token := string(runes[i : i+2])
			i += 2
			return token, true
		}
		if strings.HasPrefix(rest, `\(`) && len(runes) >= i+4 {
			token := string(runes[i : i+4])
			i += 4
			return token, true
		}
		token := string(runes[i : i+2])
		i += 2
		return token, true
	}
	for {
		from, ok1 := readOne()
		if !ok1 {
			break
		}
		to, ok2 := readOne()
		if !ok2 {
			break
		}
		if from == "" {
			break
		}
		pairs = append(pairs, [2]string{from, to})
	}
	return pairs
}
