package manpages

import (
	"strconv"
	"strings"
)

// interpolate expands macro arguments (when bound), user strings, number
// registers, and register assignments. All other escapes pass through for
// the render stage.
func (e *Engine) interpolate(s string) string {
	return e.interpolateDepth(s, 0)
}

func (e *Engine) interpolateDepth(s string, depth int) string {
	if depth > maxInterpDepth {
		return s
	}
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			i++
			continue
		}
		rest := s[i+1:]
		switch rest[0] {
		case '$':
			key := argKey(rest[1:])
			if key == "" {
				b.WriteString(s[i : i+2])
				i += 2
				continue
			}
			b.WriteString(e.argOf(key))
			i += 2 + len(key)
		case '*':
			name, size := stringRef(rest[1:])
			if size == 0 {
				b.WriteString(s[i : i+2])
				i += 2
				continue
			}
			value := e.strs[name]
			b.WriteString(e.interpolateDepth(value, depth+1))
			i += 2 + size
		case 'n':
			name, size := stringRef(rest[1:])
			if size == 0 {
				b.WriteString(s[i : i+2])
				i += 2
				continue
			}
			if strings.HasPrefix(name, "+") || strings.HasPrefix(name, "-") {
				name = name[1:]
			}
			b.WriteString(e.formatRegister(name))
			i += 2 + size
		case 'R':
			name, value, consumed := registerAssign(rest[1:])
			if consumed == 0 {
				b.WriteString(s[i : i+2])
				i += 2
				continue
			}
			e.regs[name] = int(evalSignedNumeric(value))
			i += 2 + consumed
		default:
			b.WriteString(s[i : i+2])
			i += 2
		}
	}
	return b.String()
}

// stringRef parses a name following `\*` or `\n`: either a single character,
// a two-character `(...`-style name, or a bracketed `[...]-style name (which
// may carry parameters). It returns the name and the consumed byte count.
func stringRef(rest string) (string, int) {
	if rest == "" {
		return "", 0
	}
	switch rest[0] {
	case '(':
		if len(rest) < 3 {
			return "", 0
		}
		return rest[1:3], 3
	case '[':
		end := strings.IndexByte(rest, ']')
		if end < 0 {
			return "", 0
		}
		inner := rest[1:end]
		spacey := strings.IndexAny(inner, " \t")
		if spacey >= 0 {
			return inner[:spacey], end + 1
		}
		return inner, end + 1
	default:
		return rest[:1], 1
	}
}

// argKey parses the argument reference following `\$` and returns its key
// form: a digit, an `$(nn` index, an `$[nnn]` index, or one of `*`, `@`, `^`.
func argKey(rest string) string {
	if rest == "" {
		return ""
	}
	if rest[0] == '(' {
		i := 1
		digits := 0
		for i < len(rest) && digits < 2 {
			if rest[i] < '0' || rest[i] > '9' {
				break
			}
			i++
			digits++
		}
		if !isNameSeparator(safeByte(rest, i)) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		return rest[:i]
	}
	if rest[0] == '[' {
		end := strings.IndexByte(rest, ']')
		if end < 0 {
			return ""
		}
		return `[]`
	}
	if rest[0] == '*' || rest[0] == '@' || rest[0] == '^' {
		return string(rest[0])
	}
	if rest[0] >= '0' && rest[0] <= '9' {
		return rest[:1]
	}
	return ""
}

// argOf resolves an argument key against the bound invocation arguments.
func (e *Engine) argOf(key string) string {
	switch key {
	case "*":
		return strings.Join(e.args, " ")
	case "@":
		var quoted []string
		for _, arg := range e.args {
			quoted = append(quoted, `"`+arg+`"`)
		}
		return strings.Join(quoted, " ")
	case "^":
		return strings.Join(e.args, " ")
	}
	if strings.HasPrefix(key, "(") && len(key) > 1 {
		key = key[1:]
	}
	if key == "0" {
		return e.argName
	}
	if key == "" {
		return ""
	}
	if key[0] >= '0' && key[0] <= '9' {
		index, err := strconv.Atoi(key)
		if err != nil || index == 0 {
			return ""
		}
		if index <= len(e.args) {
			return e.args[index-1]
		}
		return ""
	}
	return ""
}

func safeByte(s string, i int) byte {
	if i >= 0 && i < len(s) {
		return s[i]
	}
	return 0
}

// registerAssign parses `\R'name ±expr'` and returns the register name, the
// expression, and the consumed byte count.
func registerAssign(rest string) (string, string, int) {
	if rest == "" || rest[0] != '\'' {
		return "", "", 0
	}
	end := strings.IndexByte(rest[1:], '\'')
	if end < 0 {
		return "", "", 0
	}
	content := rest[1 : 1+end]
	if strings.HasPrefix(content, "\\") {
		content = content[1:]
	}
	name, expr := firstArgToken(content)
	if name == "" || (strings.HasPrefix(expr, "+") || strings.HasPrefix(expr, "-")) {
		return name, expr, end + 2
	}
	return name, expr, end + 2
}

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
	return e.applyCharTranslations(out)
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
	case 'h', 'v', 'w', 'x', 'l', 'L', 'o', 'Z', 'k', 'D':
		return delimitedSkip(rest)
	case 'a', 'j', 'g', 'A', 'B', 'b', 'O':
		return delimitedSkip(rest)
	case 'c', 'p':
		return 0
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
