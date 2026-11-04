package manpages

import (
	"regexp"
	"strings"
)

// Rules scored for fence-language detection: each matching line adds its
// weight to the language, and the highest scoring language above the
// threshold labels the fenced code block.
type scoreRule struct {
	re     *regexp.Regexp
	weight int
	lang   string
}

var scoreRules = []scoreRule{
	// shebangs decide immediately
	{regexp.MustCompile(`^#!\s*\S*perl\b`), 10, "perl"},
	{regexp.MustCompile(`^#!\s*\S*python3?\b`), 10, "python"},
	{regexp.MustCompile(`^#!\s*\S*(ba|z|)sh\b`), 10, "sh"},
	{regexp.MustCompile(`^#!/usr/bin/make\b`), 10, "make"},
	{regexp.MustCompile(`^#!/usr/bin/awk\b`), 10, "awk"},
	{regexp.MustCompile(`^[$%#]\s+\S`), 3, "console"},
	{regexp.MustCompile(`^\w+@\S+[#$%]`), 3, "console"},
	{regexp.MustCompile(`^[A-Z][A-Za-z0-9-]*(?:\s+(?:\[[A-Z][A-Z.\-]*\]|\[OPTION\]|\[FILE\]|\[flag\]))+\.\.\.`), 3, "console"},
	{regexp.MustCompile(`^\S+\s+\[[A-Z][A-Z.\-]*\]`), 3, "console"},

	// c family
	{regexp.MustCompile(`^#include\b`), 3, "c"},
	{regexp.MustCompile(`^\s*(?:typedef|struct|union|enum)\s+(?:[A-Za-z_]\w*)?(?:\s*{)?\s*$`), 2, "c"},
	{regexp.MustCompile(`\b(?:int|long|char|void|double|float|unsigned)\b[^;,{}]*[` + "`\"`" + `]*[;,)]?\s*$`), 1, "c"},
	{regexp.MustCompile(`^\s*[A-Za-z_][\w.<>]*\s+(\*?[a-z]\w*)\s*\(((?:[^()]|\([^()]*\))*)\w*\)(\s*;)?\s*$`), 2, "c"},
	{regexp.MustCompile(`\b(?:int|void|char)\b.*\([^()]*\)\s*{`), 2, "c"},
	{regexp.MustCompile(`^}\s*$`), 1, "c"},
	{regexp.MustCompile(`^.\s*$`), 0, ""},
	// script / interpreter dialects
	{regexp.MustCompile(`^\s*(?:my|our|local)\s+\$\w+`), 3, "perl"},
	{regexp.MustCompile(`^use\s+(?:strict|warnings|POSIX)\b`), 3, "perl"},
	{regexp.MustCompile(`^\s*(?:def\s+\w+|import\s+\w+|from\s+\w+\s+import\b)`), 3, "python"},
	{regexp.MustCompile(`^\s*(?:use\s+strict|BEGIN\s*{|END\s*{)\b`), 2, "perl"},
	{regexp.MustCompile(`^\s*func\s+\w+[({]`), 3, "go"},
	{regexp.MustCompile(`^import\s+"[^"]*"\s*$`), 2, "go"},
	{regexp.MustCompile(`^package\s+\w+$`), 2, "go"},
	{regexp.MustCompile(`^\s*var\s+\w+`), 2, "go"},
	// shell keywords and commands
	{regexp.MustCompile(`^\s*(?:if|then|elif|else\s*$|fi|for|while|case|esac|until|do|done|function)\b`), 2, "sh"},
	{regexp.MustCompile(`^\s*(?:export|set|alias|local|unset|cd|pwd|command|exec|trap|sudo)\b`), 1, "sh"},
	{regexp.MustCompile(`^\s*(?:echo|printf)\b`), 1, "sh"},
	{regexp.MustCompile(`\s&&\s|\s\|\|\s`), 1, "sh"},
	{regexp.MustCompile(`^\s*\w+=\"?[^\"]*\"?\s*[[|][^]]*]?\s*$`), 1, ""},
	// make
	{regexp.MustCompile(`^\s*\.PHONY\b|^\s*[A-Za-z0-9_./ -]+:\s*(?:$|[A-Za-z0-9_./ -]+)`), 2, "make"},
	{regexp.MustCompile(`^\t[a-z-]\S*\s+\S`), 1, "make"},
	{regexp.MustCompile(`\$\((?:CC|RM|CC|CFLAGS|MAKE|RMACHE)\)|\$\(\w+\)`), 1, "make"},
	// sql
	{regexp.MustCompile(`(?i)^\s*(SELECT|INSERT|UPDATE|DELETE|CREATE|GRANT|ALTER|DROP)\b`), 3, "sql"},
	{regexp.MustCompile(`(?i)\b(?:FROM|INNER JOIN|LEFT JOIN|SET|WHERE)\b.*;$?\s*$`), 2, "sql"},
	// markup / config
	{regexp.MustCompile(`^\[[A-Za-z0-9][^\]]*\]\s*$`), 2, "ini"},
	{regexp.MustCompile(`^[A-Za-z_][\w.-]*\s*[:=]\s*[^\s{[]\S*$`), 1, "ini"},
	{regexp.MustCompile(`^\s*</?(?:\w+|!DOCTYPE|!--)`), 2, "html"},
	{regexp.MustCompile(`^\s*(?:\{|\}$|"[^"]+"\s*:|,\s*$|\s*}\s*$)`), 1, "json"},
	{regexp.MustCompile(`^\s*(?:[\+\-]\s|@@@|\+\+\+|---[^\n]*)`), 2, "diff"},
	{regexp.MustCompile(`^\s*(?i)(listen|server|upstream|location)\s*{`), 3, "nginx"},

	{regexp.MustCompile(`^\w+\(.*\)$`), 1, "ini"},
}

// fenceThreshold is the minimum score for a language label.
const fenceThreshold = 3

// detectLanguage labels a fenced code block by content; returns "" when
// nothing crosses the threshold.
func detectLanguage(lines []string) string {
	stage := make(map[string]int, len(scoreRules))
	for _, line := range lines {
		for _, rule := range scoreRules {
			if rule.re.MatchString(line) {
				stage[rule.lang] += rule.weight
			}
		}
	}
	best := ""
	bestScore := fenceThreshold
	for lang, score := range stage {
		if score > bestScore || (score == bestScore && best == "") {
			best, bestScore = lang, score
		}
	}
	return best
}

// detectLanguageOf evaluates trailing whitespace and the SECTION hint
// (synopsis blocks are console sessions).
func detectLanguageFor(code []string, section string) string {
	if lang := detectLanguage(code); lang != "" {
		return lang
	}
	if strings.EqualFold(section, "SYNOPSIS") {
		return "console"
	}
	return ""
}
