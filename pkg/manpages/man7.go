package manpages

import "strings"

// manRequest dispatches man(7) macros and the remaining low-level roff
// requests that reach the writer.
func (w *writer) UnknownRequest(name string, args []string, noBreak bool) {
	args = w.renderedArgs(args)
	switch name {
	case "TH":
		w.title(args)
	case "SH":
		w.sectionHeading(2, args)
	case "SS":
		w.sectionHeading(3, args)
	case "P", "PP", "LP", "HP":
		w.flushBlock()
		w.item = itemNone
		w.mark = ""
		w.tag.items = nil
		w.tag.pending = false
	case "RS":
		w.flushBlock()
		w.depth++
	case "RE":
		w.flushBlock()
		if w.depth > 0 {
			w.depth--
		}
	case "TP", "TQ":
		w.taggedParagraph(name == "TQ", args)
	case "IP":
		w.indentedParagraph(args)
	case "EX":
		w.fenceOpen()
	case "EE":
		w.fenceClose()
	case "nf":
		w.fenceOpen()
	case "fi":
		w.fenceClose()
	case "ft":
		return
	case "SY":
		w.synopsisBlock(args)
	case "YS":
		w.flushBlock()
	case "OP":
		w.op(args)
	case "B", "SM", "SB":
		w.fontMacro('B', name, args)
	case "I":
		w.fontMacro('I', name, args)
	case "BI", "BR", "IB", "IR", "RB", "RI":
		w.alternatingFontMacro(name, args)
	case "MR":
		w.mr(args)
	case "MT", "UR":
		if name == "MT" && len(args) > 0 {
			args = append([]string{"mailto:" + args[0]}, args[1:]...)
		}
		w.linkStart(args)
	case "ME", "UE":
		w.linkEnd(args)
	case "IX", "PD", "DT", "AT", "UC", "PT", "BT":
		return
	default:
		return
	}
}

// renderedArgs applies escape and glyph rendering to macro arguments.
func (w *writer) renderedArgs(args []string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = w.render(arg)
	}
	return out
}

func (w *writer) sectionHeading(level int, args []string) {
	w.flushBlock()
	if len(args) == 0 {
		w.trap.head = level
		w.item = itemNone
		return
	}
	w.heading(level, strings.Join(args, " "))
}

func (w *writer) taggedParagraph(continuation bool, args []string) {
	if w.item == itemTag && !w.tag.pending {
		w.tag.pending = true
		return
	}
	if !continuation {
		w.flushBlock()
		w.item = itemTag
		w.tag.items = nil
		w.tag.pending = false
		return
	}
	w.tag.pending = true
}

func (w *writer) indentedParagraph(args []string) {
	if len(args) > 0 {
		kind, mark, _ := w.parseItemMark(args[0])
		if kind != itemNone {
			w.flushBlock()
			w.item = kind
			w.mark = mark
			w.block = nil
			return
		}
	}
	w.flushBlock()
	w.item = itemNone
}

// fontMacro renders single-font macros. When a tag is expected, the styled
// text becomes the tag row; otherwise the text joins the current block.
func (w *writer) fontMacro(kind byte, name string, args []string) {
	text := strings.Join(args, " ")
	if w.item == itemTag && !w.tag.pending {
		if text == "" {
			w.trap.font = kind
			return
		}
		w.tag.items = append(w.tag.items, wrapFont(kind, w.renderArgsJoin(args, " ")))
		w.tag.pending = true
		return
	}
	if text == "" {
		w.trap.font = kind
		return
	}
	w.inline(wrapFont(kind, w.renderArgsJoin(args, " ")), true)
}

// renderArgsJoin renders arguments and joins them with the separator.
func (w *writer) renderArgsJoin(args []string, separator string) string {
	segments := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" {
			continue
		}
		segments = append(segments, w.render(arg))
	}
	return strings.Join(segments, separator)
}

// fontPairs maps alternating-font macros to their per-argument fonts.
var fontPairs = map[string][2]byte{
	"BI": {'B', 'I'},
	"BR": {'B', 'R'},
	"IB": {'I', 'B'},
	"IR": {'I', 'R'},
	"RB": {'R', 'B'},
	"RI": {'R', 'I'},
}

// alternatingFontMacro renders .BI/.BR/.IB/.IR/.RB/.RI: font runs alternate
// per argument and adjacent pieces join without whitespace.
func (w *writer) alternatingFontMacro(name string, args []string) {
	pair := fontPairs[name]
	font := 0
	var text strings.Builder
	for _, arg := range args {
		if arg == "" {
			continue
		}
		text.WriteString(wrapFont(pair[font%2], w.render(arg)))
		font++
	}
	if text.Len() == 0 {
		return
	}
	if w.item == itemTag && !w.tag.pending {
		w.tag.items = append(w.tag.items, text.String())
		w.tag.pending = true
		return
	}
	w.inline(text.String(), true)
}

func (w *writer) synopsisBlock(args []string) {
	w.flushBlock()
	if len(args) > 0 {
		kw := w.render(args[0])
		pieces := []string{wrapFont('B', kw)}
		if len(args) > 1 {
			pieces = append(pieces, wrapFont('B', w.renderArgsJoin(args[1:], " ")))
		}
		w.inline(strings.Join(pieces, " "), true)
	}
}

func (w *writer) op(args []string) {
	if len(args) == 0 {
		return
	}
	pieces := []string{wrapFont('B', w.render(args[0]))}
	if len(args) > 1 {
		pieces = append(pieces, wrapFont('I', w.render(args[1])))
	}
	w.inline("["+strings.Join(pieces, " ")+"]", true)
}

func (w *writer) mr(args []string) {
	if len(args) == 0 {
		return
	}
	topic := w.render(args[0])
	section := ""
	trail := ""
	if len(args) > 1 {
		section = w.render(args[1])
	}
	if len(args) > 2 {
		trail = w.render(args[2])
	}
	target := "man://" + topic + "/" + section
	visible := wrapFont('B', topic)
	if section != "" {
		visible += " (" + wrapFont('I', section) + ")"
	}
	visible += trail
	w.inline("["+visible+"]("+target+")", true)
}

func (w *writer) linkStart(args []string) {
	w.flushBlock()
	if len(args) == 0 {
		return
	}
	target := w.render(args[0])
	var text []string
	w.link = &linkState{target: target, text: text}
}

func (w *writer) linkEnd(args []string) {
	if w.link == nil {
		return
	}
	text := strings.Join(w.link.text, " ")
	target := w.link.target
	w.link = nil
	w.inline("["+text+"]("+target+")", false)
	if len(args) > 0 {
		w.inline(w.render(args[0]), true)
	}
	w.flushBlock()
}
