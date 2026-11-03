package manpages

import (
	"strconv"
	"strings"
)

// mdoc holds the mdoc(7) macro layer state: open lists, the section being
// rendered, prologue metadata, and the last .Nm name.
type mdoc struct {
	w        *writer
	lists    []*mdocList
	synopsis bool
	lastName string
	date     string
	title    string
	arch     string
	osDone   bool
}

type mdocList struct {
	kind    string // bullet, dash, enum, item, inset, diag, ohang, tag, hang, column
	width   string
	compact bool
	count   int
}

func newMdoc(w *writer) *mdoc {
	return &mdoc{w: w}
}

// mdocArity maps macros that consume exactly one inline argument.
var mdocArity = map[string]int{
	"Ar": 1, "Cm": 1, "Ic": 1, "Ms": 1, "Cd": 1, "Em": 1, "Sy": 1, "No": 1,
	"Li": 1, "Tn": 1, "Pa": 1, "Ev": 1, "Dv": 1, "Er": 1, "Ad": 1, "Va": 1,
	"Vt": 1, "Fa": 1, "Ft": 1, "St": 1, "An": 1, "Al": 1,
}

func (m *mdoc) isMacroName(word string) bool {
	switch word {
	case "Ar", "Cm", "Ic", "Ms", "Cd", "Em", "Sy", "No", "Li", "Tn", "Pa",
		"Ev", "Dv", "Er", "Ad", "Va", "Vt", "Fa", "Ft", "An", "St", "Al",
		"Op", "Oo", "Oc", "Bo", "Bc", "Do", "Dc", "Dq", "Qo", "Qc", "Qq",
		"So", "Sc", "Sq", "Po", "Pc", "Pq", "Ao", "Ac", "Aq", "Bro", "Brc",
		"Brq", "Eo", "Ec", "Fl", "Ns", "Ta", "Lk", "Mt", "Nm", "Nd", "Xr",
		"Lb", "Bx", "At", "Ux", "Ex", "Rv":
		return true
	}
	return false
}

func isPunctuation(word string) bool {
	switch word {
	case ",", ";", ".", ":", "!", "?", ")", "]", "}":
		return true
	}
	return false
}

// inline renders a token run through the mdoc inline macro set and returns
// the markdown text. It is the workhorse for It heads, D1/Dl lines, Fn, In,
// and standalone semantic macros.
func (m *mdoc) inline(tokens []string) string {
	var b strings.Builder
	args := tokens
	sep := ""
	for len(args) > 0 {
		word := args[0]
		if !m.isMacroName(word) && !isPunctuation(word) {
			run := make([]string, 0, 2)
			for len(args) > 0 && !m.isMacroName(args[0]) && !isPunctuation(args[0]) {
				run = append(run, m.w.render(args[0]))
				args = args[1:]
			}
			b.WriteString(sep + strings.Join(run, " "))
			sep = " "
			continue
		}
		if isPunctuation(word) {
			b.WriteString(word)
			sep = " "
			args = args[1:]
			continue
		}
		switch word {
		case "Op", "Oo", "Oc", "Bo", "Bc", "Do", "Dc", "Dq", "Qo", "Qc", "Qq", "So", "Sc", "Sq", "Po", "Pc", "Pq", "Ao", "Ac", "Aq", "Bro", "Brc", "Brq", "Eo", "Ec":
			inner := m.inline(args[1:])
			b.WriteString(sep + m.enclose(word, inner))
			args = nil
			sep = " "
		case "Xr":
			target := ""
			args = args[1:]
			if len(args) > 0 {
				target = m.w.render(args[0])
				args = args[1:]
			}
			if len(args) > 0 && !isPunctuation(args[0]) && !m.isMacroName(args[0]) {
				target += "(" + m.w.render(args[0]) + ")"
				args = args[1:]
			}
			b.WriteString(sep + target)
			sep = " "
		case "Fl":
			var flags []string
			args = args[1:]
			for len(args) > 0 && !m.isMacroName(args[0]) && !isPunctuation(args[0]) {
				flags = append(flags, "-"+m.w.render(args[0]))
				args = args[1:]
			}
			if len(flags) == 0 {
				flags = append(flags, "-")
			}
			b.WriteString(sep + wrapFont('B', strings.Join(flags, " ")))
			sep = " "
		case "Ns":
			sep = ""
			args = args[1:]
		case "Ta":
			sep = "   "
			args = args[1:]
		case "Lk", "Mt":
			rest := args[1:]
			uri := ""
			text := ""
			if len(rest) > 0 {
				uri = m.w.render(rest[0])
				rest = rest[1:]
			}
			if len(rest) > 0 {
				text = strings.TrimSpace(m.inline(rest))
			}
			if word == "Mt" {
				uri = "mailto:" + uri
				if text == "" {
					text = strings.TrimPrefix(uri, "mailto:")
				}
			}
			if text == "" {
				text = uri
			}
			b.WriteString(sep + "[" + text + "](" + uri + ")")
			args = nil
			sep = " "
		default:
			if len(args) == 1 {
				args = nil
				continue
			}
			b.WriteString(sep + m.rend(word, m.w.render(args[1])))
			args = args[2:]
			sep = " "
		}
	}
	return strings.TrimSpace(b.String())
}

func (m *mdoc) rend(word, body string) string {
	switch word {
	case "Ar", "Pa", "Ev", "Dv", "Er", "Ad", "Fa", "Ft", "Va", "Vt", "Em":
		return wrapFont('I', body)
	case "Cm", "Ic", "Ms", "Sy", "Cd", "Nm":
		return wrapFont('B', body)
	case "Li", "Tn":
		return "`" + body + "`"
	default:
		return body
	}
}

func (m *mdoc) enclose(word, inner string) string {
	switch word {
	case "Op", "Oo", "Oc", "Bo", "Bc":
		return "[" + inner + "]"
	case "Do", "Dc", "Dq":
		return "“" + inner + "”"
	case "Qo", "Qc", "Qq":
		return "\"" + inner + "\""
	case "So", "Sc", "Sq":
		return "‘" + inner + "’"
	case "Po", "Pc", "Pq":
		return "(" + inner + ")"
	case "Ao", "Ac", "Aq":
		return "<" + inner + ">"
	case "Bro", "Brc", "Brq":
		return "{" + inner + "}"
	case "Eo", "Ec":
		return inner
	}
	return inner
}

func (m *mdoc) currentList() *mdocList {
	if len(m.lists) > 0 {
		return m.lists[len(m.lists)-1]
	}
	return nil
}

// macro dispatches an mdoc macro call or a low-level roff request.
func (m *mdoc) macro(name string, args []string, noBreak bool) {
	args = m.w.renderedArgs(args)
	switch name {
	case "Dd":
		m.date = strings.Join(args, " ")
	case "Dt":
		if len(args) > 0 {
			m.title = args[0]
		}
		if len(args) > 1 {
			m.arch = args[1]
		}
	case "Os":
		if !m.osDone {
			m.osDone = true
			m.emitTitle()
		}
	case "Sh":
		section := strings.Join(args, " ")
		m.synopsis = strings.EqualFold(section, "SYNOPSIS")
		m.w.heading(2, section)
	case "Ss":
		m.w.heading(3, strings.Join(args, " "))
		m.synopsis = false
	case "Pp":
		m.w.flushBlock()
	case "It":
		m.item(args)
	case "El":
		if n := len(m.lists); n > 0 {
			m.lists = m.lists[:n-1]
			m.w.flushBlock()
		}
	case "Bl":
		m.list(args)
	case "Bd":
		m.display(args)
	case "Ed":
		m.w.fenceClose()
	case "Dl":
		m.w.fenceOpen()
		m.w.code = append(m.w.code, m.inline(args))
		m.w.fenceClose()
	case "D1":
		m.w.flushBlock()
		m.w.flush("> "+m.inline(args), itemParagraph, false)
	case "Nm":
		if len(args) == 0 {
			if m.lastName != "" {
				m.w.inline(wrapFont('B', m.w.render(m.lastName)), true)
			}
			return
		}
		m.lastName = args[0]
		m.emitTitleIfNeeded()
		m.w.inline(wrapFont('B', m.w.render(args[0])), true)
	case "Nd":
		m.w.inline("- "+m.inline(args), true)
	case "In":
		text := m.inline(args)
		if m.synopsis {
			m.w.inline(wrapFont('B', "#include <"+text+">"), true)
			return
		}
		m.w.inline("<*"+text+"*>", true)
	case "Fn":
		if len(args) == 0 {
			return
		}
		name := m.w.render(args[0])
		callArgs := make([]string, 0, len(args)-1)
		for _, arg := range args[1:] {
			callArgs = append(callArgs, m.w.render(arg))
		}
		m.w.inline(wrapFont('B', name)+"("+strings.Join(callArgs, " ")+")", true)
	case "An":
		m.w.inline(m.inline(args), true)
	case "Bx", "At", "Ux", "Bf", "Ef", "Bk", "Ek", "Eo", "Ec", "Sm", "Tg", "Fb":
		m.w.inline(strings.Join(m.texts(args), " "), true)
	default:
		if len(args) > 0 {
			m.w.inline(m.inline(append([]string{name}, args...)), true)
		}
	}
}

func (m *mdoc) texts(body []string) []string {
	out := make([]string, 0, len(body))
	for _, token := range body {
		out = append(out, m.w.render(token))
	}
	return out
}

func (m *mdoc) emitTitleIfNeeded() {
	if !m.osDone && m.title != "" {
		m.osDone = true
		m.emitTitle()
	}
}

// emitTitle writes the H1 title once the mdoc prologue completes.
func (m *mdoc) emitTitle() {
	m.w.title([]string{m.title, m.arch, m.date, ""})
}

// item handles .It with the standard mdoc list bodies.
func (m *mdoc) item(args []string) {
	list := m.currentList()
	kind := ""
	if list != nil {
		kind = list.kind
	}
	head := ""
	if len(args) > 0 {
		head = m.inline(args)
	}
	switch kind {
	case "bullet", "dash":
		m.w.flushBlock()
		m.w.item = itemBullet
		m.w.tagStage = 0
		if head != "" {
			m.w.inline(head, true)
		}
	case "enum":
		kindCount := 0
		if list != nil {
			list.count++
			kindCount = list.count
		}
		m.w.flushBlock()
		m.w.item = itemOrdered
		m.w.mark = strconv.Itoa(kindCount) + "."
		m.w.tagStage = 0
		if head != "" {
			m.w.inline(head, true)
		}
	case "column":
		m.w.code = append(m.w.code, head)
	case "tag", "hang", "ohang", "inset", "diag", "item":
		m.w.flushBlock()
		m.w.item = itemTag
		m.w.tags = nil
		m.w.tagStage = 1
		if head != "" {
			m.w.tags = append(m.w.tags, head)
		}
	default:
		m.w.flushBlock()
		if head != "" {
			m.w.inline(head, true)
		}
	}
}

// list handles .Bl: `-type [-width val] [-offset val] [-compact]`.
func (m *mdoc) openList(kind, width string, compact bool) {
	m.lists = append(m.lists, &mdocList{kind: kind, width: width, compact: compact})
}

func (m *mdoc) list(args []string) {
	kind := "item"
	width := ""
	compact := false
	if len(args) > 0 {
		kind = strings.TrimPrefix(args[0], "-")
	}
	i := 1
	for i < len(args) {
		switch args[i] {
		case "-width", "-offset":
			if i+1 < len(args) {
				width = args[i+1]
				i += 2
				continue
			}
			i++
			continue
		case "-compact":
			compact = true
			i++
			continue
		default:
			i++
		}
	}
	m.w.flushBlock()
	m.openList(kind, width, compact)
	if kind == "column" {
		m.w.fenceOpen()
	}
}

// display handles .Bd [-type] [-offset val] [-compact]: literal types become
// code blocks.
func (m *mdoc) display(args []string) {
	literal := false
	for _, arg := range args {
		switch arg {
		case "-literal", "-unfilled", "-ragged":
			literal = true
		}
	}
	if literal {
		m.w.fenceOpen()
		return
	}
	m.w.flushBlock()
}

// finish closes all open mdoc scopes at end of input.
func (m *mdoc) finish() {
	for len(m.lists) > 0 {
		m.lists = m.lists[:len(m.lists)-1]
		m.w.flushBlock()
	}
	if len(m.lists) == 0 && m.w.fenced {
		m.w.fenceClose()
	}
	if !m.osDone && m.title != "" {
		m.osDone = true
		m.emitTitle()
	}
	m.w.flushBlock()
}
