package manpages

import (
	"fmt"
	"strconv"
	"strings"
)

type mode int

const (
	modeUnknown mode = iota
	modeMan
	modeMdoc
)

// Sink receives processed output from the expansion engine: text lines,
// line/paragraph breaks, preprocessor regions, and unknown macro calls that
// the dialect layer dispatches as man(7)/mdoc(7) macros.
type Sink interface {
	Text(line string)
	UnknownRequest(name string, args []string, noBreak bool)
	Break()
	Space(n int)
	PreprocessorBlock(name string, body []string)
}

type macroDef struct {
	body []string
	end  string
}

const (
	maxMacroDepth  = 64
	maxInterpDepth = 64
	maxBlockLines  = 4096
)

// preprocessorEnds maps preprocessor region openers to their terminators:
// the region body is captured raw and emitted as a neutral code block.
var preprocessorEnds = map[string]string{
	"TS":      "TE",
	"EQ":      "EN",
	"PS":      "PE",
	"PF":      "PE",
	"PY":      "PE",
	"G1":      "G2",
	"R1":      "R2",
	"cstart":  "cend",
	".cstart": ".cend",
}

// Engine runs the roff expansion layer: strings, number registers, user
// macros, conditionals, ignored blocks, and character translation. Text and
// unrecognized control lines are handed to a Sink after interpolation.
type Engine struct {
	src       *source
	sink      Sink
	mode      mode
	nroffMode bool

	strs   map[string]string
	regs   map[string]int
	regFmt map[string]string
	macros map[string]macroDef
	transl map[string]string

	args    []string
	argName string

	pendingElse *bool

	depth int
}

func newEngine(src *source, sink Sink) *Engine {
	return &Engine{
		src:       src,
		sink:      sink,
		nroffMode: true,
		strs: map[string]string{
			".T": "utf8",
			"lq": "“",
			"rq": "”",
			"Tm": "™",
			"R":  "®",
			"S":  "",
		},
		regs: map[string]int{
			".g":    1,
			".H":    0,
			".V":    40,
			".T":    1,
			".l":    6720,
			".n":    6720,
			".p":    120000,
			".v":    40,
			".y":    0,
			".d":    0,
			".c":    0,
			".%":    0,
			".tabs": 0,
		},
		regFmt: map[string]string{},
		macros: map[string]macroDef{},
		transl: map[string]string{},
	}
}

func (e *Engine) lineBreak() {
	e.sink.Break()
}

func (e *Engine) spaceBreak(args []string) {
	value := 1
	if len(args) > 0 {
		if parsed := parseSpaceValue(args[0]); parsed > 0 {
			value = parsed
		}
	}
	e.sink.Space(value)
}

func parseSpaceValue(arg string) int {
	digits := ""
	for _, c := range arg {
		if c < '0' || c > '9' {
			break
		}
		digits += string(c)
	}
	if digits == "" {
		return 0
	}
	value := 0
	for _, c := range digits {
		value = value*10 + int(c-'0')
	}
	return value
}

// preprocessorBlock captures a tbl/eqn/pic/... region and hands it to a
// PreprocessorBlock sink; regions are skipped when no sink supports them.
func (e *Engine) preprocessorBlock(name, end string) {
	var body []string
	for {
		line, ok := e.src.next()
		if !ok {
			break
		}
		stripped, _ := stripComment(strings.TrimRight(line, " \t"))
		body = append(body, line)
		if isEndedBy(strings.TrimSpace(stripped), end) {
			break
		}
		if len(body) > maxBlockLines {
			break
		}
	}
	e.sink.PreprocessorBlock(name, body)
}

func (e *Engine) run() error {
	for {
		line, ok := e.src.next()
		if !ok {
			return nil
		}
		if err := e.processLine(line); err != nil {
			return err
		}
	}
}

func (e *Engine) processLine(line string) error {
	stripped, hadComment := stripComment(strings.TrimRight(line, " \t"))
	if strings.TrimSpace(stripped) == "" {
		if hadComment || strings.TrimSpace(line) != "" {
			return nil
		}
		e.sink.Text("")
		return nil
	}
	if hadComment && (stripped == "." || stripped == "'") {
		return nil
	}
	name, rest, args, noBreak, ok := parseControl(stripped)
	if !ok {
		return e.emitText(stripped)
	}
	return e.dispatchControl(name, rest, args, noBreak)
}

func (e *Engine) dispatchControl(name, rest string, args []string, noBreak bool) error {
	if end, ok := preprocessorEnds[name]; ok {
		e.preprocessorBlock(name, end)
		return nil
	}
	switch name {
	case "ds", "as", "ds1", "as1":
		e.doString(name, rest)
		return nil
	case "rm":
		for _, arg := range args {
			delete(e.strs, arg)
		}
		return nil
	case "nr", "nr1":
		e.doRegister(args)
		return nil
	case "af":
		if len(args) >= 2 {
			e.regFmt[args[0]] = args[1]
		}
		return nil
	case "rr":
		for _, arg := range args {
			delete(e.regs, arg)
		}
		return nil
	case "de", "de1", "am", "am1":
		return e.doDefine(name, args)
	case "ig":
		e.skipIgnored(args)
		return nil
	case "if", "ie", "el":
		return e.doConditional(name, rest, args)
	case "tr":
		e.doTranslation(args)
		return nil
	case "shift":
		if len(e.args) > 0 {
			e.args = e.args[1:]
		}
		return nil
	case "char", "fchar", "trin", "trai", "rn":
		return nil
	case "sy", "pi", "open", "write", "writec", "close", "cf", "lf", "nx":
		return nil
	case "sp":
		e.spaceBreak(args)
		return nil
	case "br", "bp":
		e.lineBreak()
		return nil
	case "ne", "fl":
		return nil
	}
	if def, ok := e.macros[name]; ok {
		return e.expandMacro(name, def, args)
	}

	e.sink.UnknownRequest(name, e.interpolatedArgs(args), noBreak)
	return nil
}

// interpolatedArgs expands string and register interpolations in each
// argument, matching mandoc's behavior of expanding macro arguments.
func (e *Engine) interpolatedArgs(args []string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = e.interpolate(arg)
	}
	return out
}

func (e *Engine) emitText(line string) error {
	e.sink.Text(strings.TrimRight(e.renderEscapes(e.interpolate(line)), " \t"))
	return nil
}

func (e *Engine) doString(name, rest string) {
	key, value := firstArgToken(rest)
	if key == "" {
		return
	}
	if strings.HasPrefix(value, `"`) {
		value = strings.TrimPrefix(value, `"`)
	} else {
		value = strings.TrimLeft(value, " ")
	}
	if name == "as" || name == "as1" {
		if old, ok := e.strs[key]; ok {
			e.strs[key] = old + value
			return
		}
	}
	e.strs[key] = value
}

func (e *Engine) doRegister(args []string) {
	if len(args) == 0 {
		return
	}
	key := args[0]
	old := e.regs[key]
	if len(args) == 1 {
		return
	}
	expr := args[1]
	switch {
	case strings.HasPrefix(expr, "+"):
		step := 1
		if len(args) > 2 {
			if autoStep, err := strconv.Atoi(e.interpolate(args[2])); err == nil && autoStep != 0 {
				step = autoStep
			}
		}
		delta := int(evalSignedNumeric(strings.TrimPrefix(expr, "+")))
		if delta == 0 && step != 0 {
			delta = step
		}
		e.regs[key] = old + delta
	case strings.HasPrefix(expr, "-"):
		step := 1
		if len(args) > 2 {
			if autoStep, err := strconv.Atoi(e.interpolate(args[2])); err == nil && autoStep != 0 {
				step = autoStep
			}
		}
		delta := int(evalSignedNumeric(strings.TrimPrefix(expr, "-")))
		if delta == 0 && step != 0 {
			delta = step
		}
		e.regs[key] = old - delta
	default:
		e.regs[key] = int(evalSignedNumeric(expr))
	}
}

func (e *Engine) doDefine(name string, args []string) error {
	if len(args) == 0 {
		return nil
	}
	key := args[0]
	end := ""
	if len(args) >= 2 {
		candidate := strings.TrimSpace(args[1])
		// macro and end names are single tokens; anything else (e.g. a
		// comment that lost its escaping) is not a name.
		if candidate != "" && !strings.ContainsAny(candidate, " 	") {
			end = candidate
		}
	}
	var body []string
	for {
		line, ok := e.src.next()
		if !ok {
			break
		}
		stripped, _ := stripComment(line)
		trimmed := strings.TrimSpace(stripped)
		if end == "" {
			if trimmed == ".." || trimmed == "\\." {
				break
			}
		} else if isEndedBy(trimmed, end) {
			break
		}
		body = append(body, strings.TrimLeft(stripped, " "))
	}
	if (name == "am" || name == "am1") && len(body) > 0 {
		if old, ok := e.macros[key]; ok {
			e.macros[key] = macroDef{body: append(old.body, body...), end: defEnd(end, old.end)}
			return nil
		}
	}
	if key != "" {
		e.macros[key] = macroDef{body: body, end: end}
	}
	return nil
}

func defEnd(current, previous string) string {
	if current != "" {
		return current
	}
	return previous
}

func isEndedBy(line, end string) bool {
	if len(line) < 2 {
		return false
	}
	if line[0] != '.' && line[0] != '\'' {
		return false
	}
	return strings.TrimSpace(line[1:]) == end
}

func (e *Engine) skipIgnored(args []string) {
	end := ""
	if len(args) >= 1 && args[0] != "" {
		end = args[0]
	}
	seen := 0
	for {
		line, ok := e.src.next()
		if !ok {
			return
		}
		seen++
		if seen > maxBlockLines {
			return
		}
		stripped, _ := stripComment(line)
		trimmed := strings.TrimSpace(stripped)
		if end == "" {
			if trimmed == ".." || trimmed == "\\." {
				return
			}
		} else if isEndedBy(trimmed, end) {
			return
		}
	}
}

func (e *Engine) doTranslation(args []string) {
	if len(args) == 0 {
		return
	}
	pairs := parseTranslationPairs(strings.Join(args, " "))
	for _, pair := range pairs {
		if pair[0] != "" {
			e.transl[pair[0]] = pair[1]
		}
	}
}

func (e *Engine) expandMacro(name string, def macroDef, args []string) error {
	if e.depth >= maxMacroDepth {
		return errTooDeep()
	}
	savedArgs, savedName, savedCount := e.args, e.argName, e.regs[".$"]
	e.args, e.argName = args, name
	e.regs[".$"] = len(args)
	body := deEscapeArguments(def.body)
	e.depth++
	err := e.runBody(body)
	e.args, e.argName = savedArgs, savedName
	e.regs[".$"] = savedCount
	e.depth--
	return err
}

// deEscapeArguments performs the single invocation-time de-escape of macro
// bodies so that copy-mode `\\$n` references resolve through the ordinary
// argument interpolation pass.
func deEscapeArguments(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = strings.ReplaceAll(line, `\\$`, `\$`)
	}
	return out
}

func (e *Engine) runBody(lines []string) error {
	if e.depth > maxMacroDepth || len(lines) > maxBlockLines {
		return errTooDeep()
	}
	saved := e.src
	body := &source{lines: lines}
	e.src = body
	var result error
	for {
		line, ok := e.src.next()
		if !ok {
			break
		}
		if err := e.processLine(line); err != nil {
			result = err
			break
		}
	}
	e.src = saved
	return result
}

func errTooDeep() error {
	return &macroDepthError{}
}

type macroDepthError struct{}

func (e *macroDepthError) Error() string {
	return "manpages: macro recursion limit exceeded"
}

func (e *Engine) formatRegister(name string) string {
	value := e.regs[name]
	return formatRegisterValue(value, e.regFmt[name])
}

func formatRegisterValue(value int, format string) string {
	if format == "" {
		return strconv.Itoa(value)
	}
	switch {
	case strings.HasPrefix(format, "i"):
		return roman(value)
	case strings.HasPrefix(format, "I"):
		return roman(value)
	case strings.HasPrefix(format, "a"):
		return strings.ToLower(alpha(value))
	case strings.HasPrefix(format, "A"):
		return alpha(value)
	default:
		if pad := strings.Count(format, "0"); pad > 0 {
			return fmt.Sprintf("%0*d", pad, value)
		}
		return strconv.Itoa(value)
	}
}

func roman(value int) string {
	if value <= 0 {
		return ""
	}
	values := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	symbols := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	var builder strings.Builder
	for i, v := range values {
		for value >= v {
			builder.WriteString(symbols[i])
			value -= v
		}
	}
	return builder.String()
}

func alpha(value int) string {
	if value <= 0 {
		return ""
	}
	var out []rune
	for value > 0 {
		value--
		out = append([]rune{rune('A' + value%26)}, out...)
		value /= 26
	}
	return string(out)
}
