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

func isArgDigit(c byte) bool {
	return c >= '0' && c <= '9'
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
		if len(rest) >= 3 && isArgDigit(rest[1]) && isArgDigit(rest[2]) {
			return rest[:3]
		}
		return ""
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
