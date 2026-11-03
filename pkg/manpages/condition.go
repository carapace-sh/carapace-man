package manpages

import (
	"strconv"
	"strings"
)

// doConditional handles .if/.ie/.el. Bodies are either the remainder of the
// line, a `\{ ... \}` group, or the next input line. Group bodies are always
// consumed, then executed only when the condition was true.
func (e *Engine) doConditional(name, rest string, args []string) error {
	switch name {
	case "el":
		if e.pendingElse == nil {
			return nil
		}
		truth := *e.pendingElse
		e.pendingElse = nil
		if truth {
			return nil
		}
		return e.runConditionalBody(rest)
	}
	cond, body := e.splitConditional(rest)
	truth := e.evaluateCondition(cond)
	if name == "ie" {
		pending := truth
		e.pendingElse = &pending
	}
	lines, err := e.captureConditionalBody(body)
	if err != nil {
		return err
	}
	if truth {
		return e.runBody(lines)
	}
	return nil
}

func (e *Engine) runConditionalBody(rest string) error {
	lines, err := e.captureConditionalBody(rest)
	if err != nil {
		return err
	}
	return e.runBody(lines)
}

// captureConditionalBody collects the body lines of a conditional: the rest
// of the line, a `\{ ... \}` group, or (when empty) the next input line.
func (e *Engine) captureConditionalBody(rest string) ([]string, error) {
	rest = strings.TrimPrefix(rest, " ")
	if rest == "" {
		line, ok := e.src.next()
		if !ok {
			return nil, nil
		}
		rest = line
	}
	rest = stripCommentTrailing(rest)
	if idx := strings.Index(rest, `\{`); idx >= 0 {
		var lines []string
		head := strings.TrimSpace(rest[:idx])
		tail := rest[idx+2:]
		if head != "" {
			lines = append(lines, head)
		}
		if strings.TrimSpace(tail) != "" {
			lines = append(lines, tail)
		}
		follow, err := e.captureGroup(nil)
		return append(lines, follow...), err
	}
	if strings.TrimSpace(rest) == "" {
		return nil, nil
	}
	return []string{rest}, nil
}

func (e *Engine) captureGroup(prefix []string) ([]string, error) {
	var lines []string
	if len(prefix) > 0 {
		lines = append(lines, prefix...)
	}
	for {
		line, ok := e.src.next()
		if !ok {
			break
		}
		trimmed := strings.TrimRight(stripCommentTrailing(line), " \t")
		if strings.HasSuffix(trimmed, `\}`) {
			head := strings.TrimSpace(strings.TrimSuffix(trimmed, `\}`))
			if head != "" {
				lines = append(lines, head)
			}
			break
		}
		if strings.HasSuffix(trimmed, `\{`) {
			head := strings.TrimSpace(strings.TrimSuffix(trimmed, `\{`))
			if head != "" {
				lines = append(lines, head)
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) > maxBlockLines {
			return lines, &macroDepthError{}
		}
	}
	return lines, nil
}

func stripCommentTrailing(line string) string {
	stripped, _ := stripComment(line)
	return stripped
}

func trimRightSpaces(s string) string {
	return strings.TrimRight(s, " \t")
}

// splitConditional separates the condition expression and the rest-of-line
// body. Numeric expressions may span several tokens (`.if 3 >= 2 body`), so
// the longest prefix that fully evaluates as a number becomes the condition;
// conditions framed with `'` are compared verbatim.
func (e *Engine) splitConditional(rest string) (cond string, body string) {
	rest = strings.TrimLeft(rest, " 	")
	if strings.HasPrefix(rest, "'") {
		end := strings.Index(rest[1:], "'")
		if end >= 0 {
			return rest[:1+end+1], rest[1+end+1:]
		}
	}
	tokens := splitArgs(rest)
	if len(tokens) > 1 {
		for end := len(tokens) - 1; end >= 1; end-- {
			if _, ok := evalNumericExpr(e.interpolate(strings.Join(tokens[:end], " "))); ok {
				return strings.Join(tokens[:end], " "), strings.Join(tokens[end:], " ")
			}
		}
	}
	cond, body = firstArgToken(rest)
	return cond, body
}

// evaluateCondition evaluates a roff conditional expression against the
// engine's fixed environment: nroff mode (terminal), even page, no graphics.
func (e *Engine) evaluateCondition(condition string) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return false
	}
	if strings.HasPrefix(condition, "!") {
		negated := strings.TrimSpace(condition[1:])
		if strings.HasPrefix(negated, "'") {
			return !e.stringComparison(negated)
		}
		return !e.evaluateCondition(negated)
	}
	if strings.HasPrefix(condition, "'") {
		return e.stringComparison(condition)
	}
	condition = strings.TrimSpace(e.interpolate(condition))
	switch {
	case strings.HasPrefix(condition, "r "):
		key := strings.TrimSpace(condition[2:])
		_, ok := e.regs[key]
		return ok
	case strings.HasPrefix(condition, "d "):
		key := strings.TrimSpace(condition[2:])
		if _, ok := e.strs[key]; ok {
			return true
		}
		_, ok := e.macros[key]
		return ok
	case strings.HasPrefix(condition, "c "):
		return false
	case strings.HasPrefix(condition, "F "):
		return false
	case strings.HasPrefix(condition, "m "):
		return false
	case strings.HasPrefix(condition, "S "):
		return false
	}
	switch condition {
	case "n":
		return e.nroffMode
	case "t":
		return !e.nroffMode
	case "o", "e":
		return false
	}
	value, ok := evalNumericExpr(condition)
	return ok && value != 0
}

func (e *Engine) stringComparison(condition string) bool {
	inner := strings.TrimSpace(strings.TrimPrefix(condition, "'"))
	parts := splitQuoted(inner)
	if len(parts) < 2 {
		return false
	}
	left := e.interpolate(parts[0])
	right := e.interpolate(parts[1])
	return left == right
}

// splitQuoted splits strings framed by `'`.
func splitQuoted(s string) []string {
	var parts []string
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			parts = append(parts, s[last:i])
			last = i + 1
		}
	}
	if last <= len(s) {
		parts = append(parts, s[last:])
	}
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func evalSignedNumeric(expr string) float64 {
	value, ok := evalNumericExpr(expr)
	if !ok {
		return 0
	}
	return value
}

// evalNumericExpr evaluates a small numeric expression subset: integers and
// decimals, unit suffixes, + - * / %, comparisons, & (and), | (or),
// parentheses, and unary +/-.
func evalNumericExpr(expr string) (float64, bool) {
	p := &numParser{s: expr}
	value, ok := p.parseExpr()
	if !ok {
		return 0, false
	}
	p.skipSpaces()
	if p.pos != len(p.s) {
		return 0, false
	}
	return value, true
}

type numParser struct {
	s   string
	pos int
}

func (p *numParser) skipSpaces() {
	for p.pos < len(p.s) && (p.s[p.pos] == ' ' || p.s[p.pos] == '\t') {
		p.pos++
	}
}

func (p *numParser) peek() byte {
	if p.pos < len(p.s) {
		return p.s[p.pos]
	}
	return 0
}

func (p *numParser) parseExpr() (float64, bool) {
	p.skipSpaces()
	value, ok := p.parseOr()
	if !ok {
		return 0, false
	}
	return value, true
}

func (p *numParser) parseOr() (float64, bool) {
	p.skipSpaces()
	left, ok := p.parseAnd()
	if !ok {
		return 0, false
	}
	for {
		p.skipSpaces()
		c := p.peek()
		if c != '|' && c != ':' {
			return left, true
		}
		if c == ':' && p.pos+1 < len(p.s) && p.s[p.pos+1] == '=' {
			return left, true
		}
		p.pos++
		right, ok := p.parseAnd()
		if !ok {
			return 0, false
		}
		if left != 0 || right != 0 {
			left = 1
		} else {
			left = 0
		}
	}
}

func (p *numParser) parseAnd() (float64, bool) {
	p.skipSpaces()
	left, ok := p.parseComparison()
	if !ok {
		return 0, false
	}
	for {
		p.skipSpaces()
		if p.peek() != '&' {
			return left, true
		}
		p.pos++
		right, ok := p.parseComparison()
		if !ok {
			return 0, false
		}
		if left != 0 && right != 0 {
			left = 1
		} else {
			left = 0
		}
	}
}

func (p *numParser) parseComparison() (float64, bool) {
	p.skipSpaces()
	left, ok := p.parseAdditive()
	if !ok {
		return 0, false
	}
	p.skipSpaces()
	op := ""
	switch {
	case strings.HasPrefix(p.s[p.pos:], "<="):
		op, p.pos = "<=", p.pos+2
	case strings.HasPrefix(p.s[p.pos:], ">="):
		op, p.pos = ">=", p.pos+2
	case strings.HasPrefix(p.s[p.pos:], "=="):
		op, p.pos = "==", p.pos+2
	case strings.HasPrefix(p.s[p.pos:], "!="):
		op, p.pos = "!=", p.pos+2
	case strings.HasPrefix(p.s[p.pos:], "<"):
		op, p.pos = "<", p.pos+1
	case strings.HasPrefix(p.s[p.pos:], ">"):
		op, p.pos = ">", p.pos+1
	case p.peek() == '=':
		op, p.pos = "=", p.pos+1
	default:
		return left, true
	}
	right, ok := p.parseAdditive()
	if !ok {
		return 0, false
	}
	switch op {
	case "<":
		return boolNum(left < right), true
	case "<=":
		return boolNum(left <= right), true
	case ">":
		return boolNum(left > right), true
	case ">=":
		return boolNum(left >= right), true
	default:
		return boolNum(left == right), true
	}
}

func (p *numParser) parseAdditive() (float64, bool) {
	p.skipSpaces()
	left, ok := p.parseMultiplicative()
	if !ok {
		return 0, false
	}
	for {
		p.skipSpaces()
		switch p.peek() {
		case '+':
			p.pos++
			right, ok := p.parseMultiplicative()
			if !ok {
				return 0, false
			}
			left += right
		case '-':
			if p.peekNextIsDigit() {
				continue
			}
			p.pos++
			right, ok := p.parseMultiplicative()
			if !ok {
				return 0, false
			}
			left -= right
		default:
			return left, true
		}
	}
}

func (p *numParser) peekNextIsDigit() bool {
	i := p.pos + 1
	return i < len(p.s) && (p.s[i] >= '0' && p.s[i] <= '9' || p.s[i] == '.')
}

func (p *numParser) parseMultiplicative() (float64, bool) {
	p.skipSpaces()
	left, ok := p.parseUnary()
	if !ok {
		return 0, false
	}
	for {
		p.skipSpaces()
		switch p.peek() {
		case '*':
			p.pos++
			right, ok := p.parseUnary()
			if !ok {
				return 0, false
			}
			left *= right
		case '/':
			p.pos++
			right, ok := p.parseUnary()
			if !ok {
				return 0, false
			}
			if right == 0 {
				return 0, false
			}
			left /= right
		case '%':
			p.pos++
			right, ok := p.parseUnary()
			if !ok {
				return 0, false
			}
			if int(right) == 0 {
				return 0, false
			}
			left = float64(int(left) % int(right))
		default:
			return left, true
		}
	}
}

func (p *numParser) parseUnary() (float64, bool) {
	p.skipSpaces()
	switch p.peek() {
	case '-':
		p.pos++
		right, ok := p.parseUnary()
		if !ok {
			return 0, false
		}
		return -right, true
	case '+':
		p.pos++
		return p.parseUnary()
	case '(':
		p.pos++
		value, ok := p.parseExpr()
		if !ok {
			return 0, false
		}
		p.skipSpaces()
		if p.peek() != ')' {
			return 0, false
		}
		p.pos++
		return value, true
	}
	return p.parseNumber()
}

func (p *numParser) parseNumber() (float64, bool) {
	p.skipSpaces()
	start := p.pos
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c >= '0' && c <= '9' || c == '.' {
			p.pos++
			continue
		}
		break
	}
	if p.pos == start || p.pos == -1 {
		return 0, false
	}
	text := p.s[start:p.pos]
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, false
	}
	if p.pos < len(p.s) {
		switch p.s[p.pos] {
		case 'u':
			value *= 1
			p.pos++
		case 'p':
			value *= 4
			p.pos++
		case 'n':
			value *= 17
			p.pos++
		case 'm':
			value *= 33
			p.pos++
		case 'c':
			value *= 95
			p.pos++
		case 'P':
			value *= 40
			p.pos++
		case 'i':
			value *= 240
			p.pos++
		case 'v':
			value *= 40
			p.pos++
		}
	}
	return value, true
}

func boolNum(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
