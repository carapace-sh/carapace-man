package manpages

import (
	"strings"
	"testing"
)

type recSink struct {
	text    []string
	unknown []string
	nobreak []bool
}

func (s *recSink) Text(line string) {
	s.text = append(s.text, line)
}

func (s *recSink) UnknownRequest(name string, args []string, noBreak bool) {
	s.unknown = append(s.unknown, name+" "+strings.Join(args, "\x00"))
	s.nobreak = append(s.nobreak, noBreak)
}

func run(src string) (*recSink, *Engine, error) {
	sink := &recSink{}
	sourcex, err := newSource(strings.NewReader(src))
	if err != nil {
		return sink, nil, err
	}
	engine := newEngine(sourcex, sink)
	return sink, engine, engine.run()
}

func TestSourceJoinsContinuations(t *testing.T) {
	source, err := newSource(strings.NewReader("a\\\nb\nafter\n"))
	if err != nil {
		t.Fatal(err)
	}
	line, ok := source.next()
	if !ok || line != "ab" {
		t.Errorf("expected joined line, got %q", line)
	}
	line, ok = source.next()
	if !ok || line != "after" {
		t.Errorf("expected after, got %q", line)
	}
}

func TestSourceKeepsEvenBackslashes(t *testing.T) {
	source, err := newSource(strings.NewReader("a\\\\\nb\n"))
	if err != nil {
		t.Fatal(err)
	}
	line, _ := source.next()
	if line != `a\\` {
		t.Errorf("expected single line, got %q", line)
	}
	if _, ok := source.next(); !ok {
		t.Errorf("expected second line")
	}
}

func TestParseControlShapes(t *testing.T) {
	name, rest, args, noBreak, ok := parseControl(".SH NAME")
	if !ok || name != "SH" || rest != " NAME" || len(args) != 1 || args[0] != "NAME" || noBreak {
		t.Errorf("unexpected .SH parse: %q %q %v", name, rest, args)
	}
	_, _, _, noBreak, ok = parseControl(`'br\}`)
	if !ok || !noBreak {
		t.Errorf("expected no-break control line")
	}
	if _, _, _, _, ok := parseControl(`\&.NET`); ok {
		t.Errorf("escaped dot must not parse as control")
	}
	if _, _, _, _, ok := parseControl("."); ok {
		t.Errorf("bare dot must not parse as control")
	}
}

func TestSplitArgs(t *testing.T) {
	args := splitArgs(` a "b c" 'd e" "' 'f g' "" `)
	want := []string{"a", "b c", `d e" "`, "f g", ""}
	if len(args) != len(want) {
		t.Fatalf("arg count mismatch: got %v", args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("arg %d: got %q want %q", i, args[i], want[i])
		}
	}
}

func TestDefineString(t *testing.T) {
	_, engine, err := run(strings.Join([]string{
		".ds name value",
		`.ds lead "  spaced value`,
		".ds empty",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if engine.strs["name"] != "value" || engine.strs["lead"] != "  spaced value" || engine.strs["empty"] != "" {
		t.Errorf("string definitions wrong: %q %q %q", engine.strs["name"], engine.strs["lead"], engine.strs["empty"])
	}
}

func TestAppendString(t *testing.T) {
	_, engine, err := run(strings.Join([]string{
		".ds x ab",
		".as x cd",
		".as x ef",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if engine.strs["x"] != "abcdef" {
		t.Errorf("expected abcdef, got %q", engine.strs["x"])
	}
}

func TestNumberRegisters(t *testing.T) {
	_, engine, err := run(strings.Join([]string{
		".nr cnt 5",
		".nr cnt +1",
		".nr out -3",
		".nr neg 0 -2",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if engine.regs["cnt"] != 6 {
		t.Errorf("cnt = %d, want 6", engine.regs["cnt"])
	}
	if engine.regs["out"] != -3 {
		t.Errorf("out = %d, want -3", engine.regs["out"])
	}
}

func TestInterpolation(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		".nr size 5",
		".ds font-name value",
		"\\n[size] \\*[font-name] \\*[.T]",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.text) != 1 {
		t.Fatalf("expected single output line, got %v", sink.text)
	}
	if sink.text[0] != "5 value utf8" {
		t.Errorf("interpolation mismatch: %q", sink.text[0])
	}
}

func TestRegisterFormat(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		".nr page 9",
		".af page i",
		"\\n[page]",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "IX" {
		t.Errorf("got %v", sink.text)
	}
}

func TestEscapesInText(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		`a\-b`,
		`\&dot`,
		`\(aq`,
		`em: \(em`,
		`tiny \|\|gap`,
		"word1 \\c",
		"word2",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a-b", "dot", "'", "em: —", "tiny gap", "word1 " + joinMarker, "word2"}
	if len(sink.text) != len(want) {
		t.Fatalf("line count: got %v", sink.text)
	}
	for i := range want {
		if sink.text[i] != want[i] {
			t.Errorf("line %d: got %q want %q", i, sink.text[i], want[i])
		}
	}
}

func TestIgnoredBlockSkipped(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		"first",
		".ig",
		"hidden line",
		"..",
		"last",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "first|last" {
		t.Errorf("got %v", sink.text)
	}
}

func TestIgnoredBlockNamedEnd(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		".ig END",
		"hidden",
		".END",
		"last",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "last" {
		t.Errorf("got %v", sink.text)
	}
}

func TestDefineAndExpandMacro(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		".de greet",
		`.ds who \$1`,
		`hello \*[who]`,
		"..",
		".greet world",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "hello world" {
		t.Errorf("got %v", sink.text)
	}
}

func TestDefineAndExpandMacroDollarStar(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		".de two",
		`\$*`,
		"..",
		".two alpha beta gamma",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "alpha beta gamma" {
		t.Errorf("got %v", sink.text)
	}
}

func TestMacroConditionBranches(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		".de pick",
		`.if \n(.$=1 .ds in solo`,
		`.ds count \n(.$`,
		`\*[in] \*[count]`,
		"..",
		".pick onlyone",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "solo 1" {
		t.Errorf("got %v", sink.text)
	}
}

func TestEvaluateConditionNumeric(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		`.if n .ds has-n yes`,
		`.if \n(.g=1 GroffYes`,
		`.if 3 >= 2 ThreeWins`,
		`.if !0 twoD`,
		`.if t TroffNo`,
		`.if ((\n(.g)) BranchOnG`,
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "GroffYes|ThreeWins|twoD|BranchOnG" {
		t.Errorf("got %v", sink.text)
	}
}

func TestConditionalGroup(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		`.if n \{\`,
		".ds branch terminal",
		`'br\}`,
		`value: \*[branch]`,
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "value: terminal" {
		t.Errorf("got %v", sink.text)
	}
}

func TestConditionalGroupSkipsWhenFalse(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		`.if t \{\`,
		"hidden",
		`'br\}`,
		`visible`,
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "visible" {
		t.Errorf("got %v", sink.text)
	}
}

func TestIeElseConditional(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		`.ie n .ds res on`,
		`.el .ds res off`,
		`\*[res]`,
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "on" {
		t.Errorf("got %v", sink.text)
	}
}

func TestIeElseWithGroups(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		`.ie t \{\`,
		".ds res troff",
		`'br\}`,
		`.el \{\`,
		".ds res nroff",
		`'br\}`,
		`\*[res]`,
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "nroff" {
		t.Errorf("got %v", sink.text)
	}
}

func TestTranslation(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		`.tr \(*W-`,
		`\(*W`,
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "-" {
		t.Errorf("got %v", sink.text)
	}
}

func TestCommentLines(t *testing.T) {
	sink, _, err := run(strings.Join([]string{
		`.\" full comment`,
		`text \" trailing`,
		"after",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sink.text, "|") != "text|after" {
		t.Errorf("got %v", sink.text)
	}
}

func TestUnknownRequestForwarded(t *testing.T) {
	sink, _, err := run(`.IX Item stuff`)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.unknown) != 1 || sink.unknown[0] != "IX Item\x00stuff" {
		t.Errorf("got %v", sink.unknown)
	}
}
