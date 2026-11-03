package manpages

import (
	"strings"
)

type itemKind int

const (
	itemNone itemKind = iota
	itemBullet
	itemOrdered
	itemTag
	itemParagraph
	itemHeading
	itemCode
)

type linkState struct {
	target string
	text   []string
}

// writer renders engine output as markdown. It receives engine-interpolated
// text lines and dispatched macro calls (man(7) macros via manRequest,
// remaining roff requests via Break/Space).
type writer struct {
	out    strings.Builder
	render func(text string) string

	fenced bool
	code   []string

	block []string

	depth int

	item     itemKind
	mark     string
	tags     []string
	tagStage int

	pendingFont byte
	pendingHead int

	link *linkState

	lastKind itemKind
	wrote    bool
}

func newWriter(render func(string) string) *writer {
	return &writer{render: render}
}

func wrapFont(kind byte, text string) string {
	switch kind {
	case 'B':
		return "**" + text + "**"
	case 'I':
		return "*" + text + "*"
	}
	return text
}

func (w *writer) inline(seg string, space bool) {
	if len(w.block) == 0 {
		w.block = append(w.block, seg)
		return
	}
	last := len(w.block) - 1
	if space {
		w.block[last] += " " + seg
		return
	}
	w.block[last] += seg
}

// Break renders a `.br` line break. Inside a paragraph block it starts a new
// block line so list items stay together; inside code it is a plain newline.
func (w *writer) Break() {
	if w.fenced {
		return
	}
	if len(w.block) > 0 {
		w.block = append(w.block, "")
		return
	}
}

// Space renders `.sp` vertical space as a paragraph separator.
func (w *writer) Space(n int) {
	w.flushBlock()
}

func (w *writer) fenceOpen() {
	w.flushBlock()
	if w.fenced {
		return
	}
	w.fenced = true
	w.code = nil
}

func (w *writer) fenceClose() {
	if !w.fenced {
		return
	}
	body := strings.Join(w.code, "\n")
	w.code = nil
	w.fenced = false
	if strings.TrimSpace(body) == "" {
		return
	}
	w.flush("```\n"+body+"\n```", itemCode, false)
}

// Text renders a text line from the engine into the open construct. Code
// blocks, links, heading and font traps precede tag and paragraph handling.
func (w *writer) Text(line string) {
	if w.fenced {
		w.code = append(w.code, line)
		return
	}
	if w.link != nil {
		w.link.text = append(w.link.text, line)
		return
	}
	if w.pendingHead != 0 {
		level := w.pendingHead
		w.pendingHead = 0
		w.item = itemNone
		w.heading(level, w.render(line))
		return
	}
	if w.pendingFont != 0 {
		kind := w.pendingFont
		w.pendingFont = 0
		w.inline(wrapFont(kind, w.render(line)), true)
		return
	}
	if w.item == itemTag && w.tagStage == 0 {
		w.tags = append(w.tags, w.render(line))
		w.tagStage = 1
		return
	}
	w.inline(line, true)
}

func (w *writer) heading(level int, text string) {
	w.flushBlock()
	w.flush(strings.Repeat("#", level)+" "+text, itemHeading, false)
	w.item = itemNone
}

func (w *writer) title(arguments []string) {
	name := ""
	section := ""
	date := ""
	source := ""
	if len(arguments) > 0 {
		name = strings.TrimSpace(w.render(arguments[0]))
	}
	if len(arguments) > 1 {
		section = w.render(arguments[1])
	}
	if len(arguments) > 2 {
		date = w.render(arguments[2])
	}
	if len(arguments) > 3 {
		source = w.render(arguments[3])
	}
	var title strings.Builder
	title.WriteString("# " + name + "(" + section + ")\n")
	metadata := []string{}
	if date != "" {
		metadata = append(metadata, "date: "+date)
	}
	if source != "" {
		metadata = append(metadata, "source: "+source)
	}
	if len(metadata) > 0 {
		title.WriteString("\n<!-- " + strings.Join(metadata, "; ") + " -->\n")
	}
	w.out.WriteString(title.String())
	w.wrote = true
}

func parseNonNegative(s string) int {
	value := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		value = value*10 + int(c-'0')
	}
	return value
}

// parseItemMark classifies an `.IP` mark: bullet glyph, ordered item, or a
// plain shift that yields no list rendering.
func (w *writer) parseItemMark(raw string) (itemKind, string, int) {
	mark := strings.TrimSpace(strings.Trim(raw, "\"'"))
	for _, prefix := range []string{"•", "*"} {
		if strings.HasPrefix(mark, prefix) {
			indent := 0
			if rest := strings.TrimSpace(strings.TrimPrefix(mark, prefix)); rest != "" {
				indent = parseNonNegative(rest)
				if indent < 0 {
					indent = 0
				}
			}
			return itemBullet, prefix, indent
		}
	}
	first := strings.Split(mark, " ")[0]
	if strings.HasSuffix(first, ".") {
		if number := parseNonNegative(strings.TrimSuffix(first, ".")); number > 0 {
			return itemOrdered, mark, number
		}
	}
	return itemNone, mark, 0
}

func (w *writer) bodyContent() string {
	return strings.TrimRight(strings.Join(w.block, "\n"), " ")
}

func indentBody(s string, depth int) string {
	lines := strings.Split(strings.Trim(s, "\n"), "\n")
	for i := 1; i < len(lines); i++ {
		if lines[i] != "" {
			lines[i] = strings.Repeat("  ", depth) + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func (w *writer) flushBlock() {
	if w.fenced || w.link != nil {
		return
	}
	if len(w.block) == 0 && w.item != itemTag {
		return
	}
	kind := w.item
	if kind == itemNone {
		kind = itemParagraph
	}
	content := w.bodyContent()
	switch w.item {
	case itemBullet:
		content = indentBody("- "+content, w.depth)
	case itemOrdered:
		content = indentBody(w.mark+" "+content, w.depth)
	case itemTag:
		head := strings.Join(w.tags, ", ")
		if content != "" {
			content = head + "\n" + indentBody(content, w.depth)
		} else {
			content = head
		}
	}
	if strings.TrimSpace(content) == "" {
		w.resetBlock()
		return
	}
	same := w.lastKind == w.item && (w.item == itemBullet || w.item == itemOrdered || w.item == itemTag)
	w.flush(content, kind, same)
	w.resetBlock()
}

func (w *writer) flush(content string, kind itemKind, same bool) {
	if w.wrote {
		if same {
			w.out.WriteString("\n")
		} else {
			w.out.WriteString("\n\n")
		}
	}
	w.out.WriteString(content)
	w.wrote = true
	w.lastKind = kind
}

func (w *writer) resetBlock() {
	w.block = nil
	w.tags = nil
	w.tagStage = 0
	w.mark = ""
}
