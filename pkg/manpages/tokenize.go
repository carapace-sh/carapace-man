package manpages

import "strings"

// parseControl splits a control line into name, raw remainder, and arguments.
// It reports false if the line is not a control line. Lines beginning with `'`
// are no-break control lines.
func parseControl(line string) (name string, rest string, args []string, noBreak bool, ok bool) {
	if line == "" {
		return "", "", nil, false, false
	}
	switch line[0] {
	case '.':
	case '\'':
		noBreak = true
	default:
		return "", "", nil, false, false
	}
	if line == "." || line == "'" {
		return "", "", nil, noBreak, false
	}
	body := strings.TrimLeft(line[1:], " \t")
	if body == "" {
		return "", "", nil, noBreak, false
	}
	i := 0
	for i < len(body) && !isNameSeparator(body[i]) {
		i++
	}
	name = body[:i]
	rest = body[i:]
	return name, rest, splitArgs(rest), noBreak, true
}

// firstArgToken splits the first whitespace-delimited token off s.
func firstArgToken(s string) (string, string) {
	s = strings.TrimLeft(s, " \t")
	i := 0
	for i < len(s) && !isNameSeparator(s[i]) {
		i++
	}
	return s[:i], strings.TrimLeft(s[i:], " \t")
}

func isNameSeparator(c byte) bool {
	return c == ' ' || c == '\t'
}

// splitArgs splits request arguments on whitespace, honoring quoting with
// double and single quotes.
func splitArgs(s string) []string {
	var args []string
	i := 0
	for i < len(s) {
		for i < len(s) && isNameSeparator(s[i]) {
			i++
		}
		if i >= len(s) {
			break
		}
		var quoted bool
		var quote byte
		if s[i] == '"' || s[i] == '\'' {
			quoted = true
			quote = s[i]
			i++
		}
		var arg strings.Builder
		for i < len(s) {
			if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == ' ') {
				arg.WriteByte(s[i+1])
				i += 2
				continue
			}
			if quoted && s[i] == '\\' && i+1 < len(s) && s[i+1] == quote {
				arg.WriteByte(quote)
				i += 2
				continue
			}
			if !quoted && isNameSeparator(s[i]) {
				break
			}
			if quoted && s[i] == quote {
				i++
				break
			}
			arg.WriteByte(s[i])
			i++
		}
		args = append(args, arg.String())
	}
	return args
}

// stripComment removes comment escapes and their tail from a line, reporting
// whether a comment was found.
func stripComment(line string) (string, bool) {
	for i := 0; i < len(line); i++ {
		if line[i] != '\\' || i+1 >= len(line) {
			continue
		}
		c := line[i+1]
		if c == '"' || c == '#' {
			return line[:i], true
		}
		if c == '\\' {
			i++
		}
	}
	return line, false
}
