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
	// shebangs and decisive headers
	{regexp.MustCompile(`^#!\s*/usr/bin/env\s+(?:ba|z|)sh\b`), 10, "sh"},
	{regexp.MustCompile(`^#!\s*/usr/bin/env\s+python3?\b`), 10, "python"},
	{regexp.MustCompile(`^#!\s*/usr/bin/env\s+perl\b`), 10, "perl"},
	{regexp.MustCompile(`^#!\s*/usr/bin/env\s+ruby\b`), 10, "ruby"},
	{regexp.MustCompile(`^#!\s*/usr/bin/env\s+node\b`), 10, "javascript"},
	{regexp.MustCompile(`^#!\s*\S*perl\b`), 10, "perl"},
	{regexp.MustCompile(`^#!\s*\S*python3?\b`), 10, "python"},
	{regexp.MustCompile(`^#!\s*\S*(?:ba|z|)sh\b`), 10, "sh"},
	{regexp.MustCompile(`^#!\s*\S*ruby\b`), 10, "ruby"},
	{regexp.MustCompile(`^#!\s*\S*node\b`), 10, "javascript"},
	{regexp.MustCompile(`^#!/usr/bin/make\b`), 10, "make"},
	{regexp.MustCompile(`^#!/usr/bin/awk\b`), 10, "awk"},
	{regexp.MustCompile(`^#!\s*\S*(?:powershell|pwsh)\b`), 10, "powershell"},
	{regexp.MustCompile(`^\s*<\?xml\b`), 10, "xml"},

	// console / usage
	{regexp.MustCompile(`^[$%#]\s+\S`), 3, "console"},
	{regexp.MustCompile(`^\w+@\S+[#$%]`), 3, "console"},
	{regexp.MustCompile(`^[A-Z][A-Za-z0-9-]*(?:\s+(?:\[[A-Z][A-Z.\-]*\]|\[OPTION\]|\[FILE\]|\[flag\]))+\.\.\.`), 3, "console"},
	{regexp.MustCompile(`^\S+\s+\[[A-Z][A-Z.\-]*\]`), 3, "console"},

	// c / c++ family
	{regexp.MustCompile(`^#include\b`), 3, "c"},
	{regexp.MustCompile(`^\s*(?:typedef|struct|union|enum)\b`), 2, "c"},
	{regexp.MustCompile(`^\s*[A-Za-z_][\w\s\*]*\b[A-Za-z_]\w*\s*;\s*(?:/\*.*\*/|//.*)?$`), 1, "c"},
	{regexp.MustCompile(`^\s*[A-Za-z_][\w.<>]*\s+(\*?[a-z]\w*)\s*\(((?:[^()]|\([^()]*\))*)\w*\)(\s*;)?\s*$`), 2, "c"},
	{regexp.MustCompile(`\b(?:int|void|char)\b.*\([^()]*\)\s*{`), 2, "c"},
	{regexp.MustCompile(`^}\s*(?:[A-Za-z_]\w*\s*)?;\s*$`), 1, "c"},
	{regexp.MustCompile(`^\s*#pragma\b`), 2, "cpp"},
	{regexp.MustCompile(`^\s*(?:namespace|template)\b`), 3, "cpp"},
	{regexp.MustCompile(`^\s*class\s+[A-Za-z_]\w*\b`), 3, "cpp"},
	{regexp.MustCompile(`\bstd::\w+`), 2, "cpp"},
	{regexp.MustCompile(`^\s*(?:public|private|protected)\s*:`), 2, "cpp"},
	// script / interpreter dialects
	{regexp.MustCompile(`^\s*(?:my|our|local)\s+\$\w+`), 3, "perl"},
	{regexp.MustCompile(`^use\s+(?:strict|warnings|POSIX)\b`), 3, "perl"},
	{regexp.MustCompile(`^\s*(?:use\s+strict|BEGIN\s*{|END\s*{)\b`), 2, "perl"},
	{regexp.MustCompile(`^\s*(?:def\s+\w+|import\s+\w+|from\s+\w+\s+import\b)`), 3, "python"},
	{regexp.MustCompile(`^\s*func\s+\w+[({]`), 3, "go"},
	{regexp.MustCompile(`^import\s+"[^"]*"\s*$`), 2, "go"},
	{regexp.MustCompile(`^package\s+\w+$`), 2, "go"},
	{regexp.MustCompile(`^\s*var\s+\w+`), 2, "go"},
	{regexp.MustCompile(`^\s*(?:fn|let\s+mut|impl|pub\s+(?:fn|struct|enum|trait|mod|use))\b`), 3, "rust"},
	{regexp.MustCompile(`\b(?:println!|vec!|Option<|Result<)\b`), 2, "rust"},
	{regexp.MustCompile(`^\s*(?:require\s+['"][^'"]+['"]|module\s+\w+|class\s+\w+\s*(?:<\s*\w+)?\s*$)`), 2, "ruby"},
	{regexp.MustCompile(`^\s*(?:def|end)\b`), 1, "ruby"},
	{regexp.MustCompile(`\b(?:puts|attr_reader|attr_accessor)\b`), 2, "ruby"},
	{regexp.MustCompile(`^\s*(?:const|let|var|function|class|async\s+function)\b`), 2, "javascript"},
	{regexp.MustCompile(`^\s*(?:import\b.*\bfrom\b|export\s+(?:default|const|let|var|function|class|async|{|\*|type|interface)\b)`), 2, "javascript"},
	{regexp.MustCompile(`\bconsole\.log\b|=>`), 2, "javascript"},
	{regexp.MustCompile(`^\s*(?:interface|type)\s+\w+`), 3, "typescript"},
	{regexp.MustCompile(`:\s*(?:string|number|boolean|any|unknown|void)\b`), 2, "typescript"},

	// shell / powershell
	{regexp.MustCompile(`^\s*(?:if|then|elif|else\s*$|fi|for|while|case|esac|until|do|done|function)\b`), 2, "sh"},
	{regexp.MustCompile(`^\s*(?:export|set|alias|local|unset|cd|pwd|command|exec|trap|sudo)\b`), 1, "sh"},
	{regexp.MustCompile(`^\s*(?:echo|printf|source|\.)\b`), 1, "sh"},
	{regexp.MustCompile(`\s&&\s|\s\|\|\s|2>&1|<<-?'?\w+'?`), 1, "sh"},
	{regexp.MustCompile(`^\s*set\s+-[euxo]+\b`), 2, "sh"},
	{regexp.MustCompile(`^\s*\[\[.*\]\]\s*$`), 2, "sh"},
	{regexp.MustCompile(`^\s*(?:param\s*\(|begin\s*{|process\s*{|end\s*{)`), 3, "powershell"},
	{regexp.MustCompile(`\b(?:Get|Set|New|Remove|Invoke|Write|Test|Import|Export)-[A-Za-z]\w+`), 3, "powershell"},
	{regexp.MustCompile(`\$(?:PS\w+|env:|args|true|false|null)\b`), 2, "powershell"},
	{regexp.MustCompile(`\s-(?:eq|ne|match|like|contains|in|notin|and|or)\s`), 2, "powershell"},

	// make
	{regexp.MustCompile(`^\s*\.PHONY\b`), 3, "make"},
	{regexp.MustCompile(`^[A-Za-z0-9_./-]+(?:\s+[A-Za-z0-9_./-]+)*\s*:(?:\s*$|\s+[A-Za-z0-9_./$()%+.,-]+(?:\s+[A-Za-z0-9_./$()%+.,-]+)*\s*$)`), 1, "make"},
	{regexp.MustCompile(`^\t\S`), 1, "make"},
	{regexp.MustCompile(`\$\([A-Za-z0-9_]+\)`), 1, "make"},

	// sql
	{regexp.MustCompile(`(?i)^\s*(SELECT|INSERT|UPDATE|DELETE|CREATE|GRANT|ALTER|DROP|WITH)\b`), 3, "sql"},
	{regexp.MustCompile(`(?i)\b(?:FROM|INNER\s+JOIN|LEFT\s+JOIN|RIGHT\s+JOIN|SET|WHERE|GROUP\s+BY|ORDER\s+BY|VALUES)\b`), 2, "sql"},

	// dockerfile
	{regexp.MustCompile(`^\s*(?:FROM|RUN|CMD|COPY|ADD|ENTRYPOINT|ENV|ARG|WORKDIR|EXPOSE|USER|LABEL|VOLUME|HEALTHCHECK|SHELL|STOPSIGNAL|ONBUILD)\b`), 3, "dockerfile"},

	// markup / config / data
	{regexp.MustCompile(`^\s*---\s*$`), 2, "yaml"},
	{regexp.MustCompile(`^\s*[A-Za-z0-9_.-]+\s*:\s*(?:\||>[-+]?|&\S+|\*\S+|\[[^\]]*\]|\{[^}]*\}|".*"|'.*'|[^#\s].*)?\s*$`), 2, "yaml"},
	{regexp.MustCompile(`^\s*-\s+\S`), 1, "yaml"},
	{regexp.MustCompile(`^\s*\[\[[A-Za-z0-9_.-]+\]\]\s*$`), 3, "toml"},
	{regexp.MustCompile(`^\s*[A-Za-z0-9_.-]+\s*=\s*(?:"[^"]*"|'[^']*'|\[[^\]]*\]|\{[^}]*\}|true|false|\d)`), 2, "toml"},
	{regexp.MustCompile(`^\[[A-Za-z0-9][^\]]*\]\s*$`), 1, "ini"},
	{regexp.MustCompile(`^[A-Za-z_][\w.-]*\s*[:=]\s*[^\s{[]\S*$`), 1, "ini"},
	{regexp.MustCompile(`^\s*</?(?:\w+|!DOCTYPE|!--)`), 2, "html"},
	{regexp.MustCompile(`^\s*(?:\{|\}$|"[^"]+"\s*:|,\s*$|\s*}\s*$)`), 1, "json"},
	{regexp.MustCompile(`^\s*(?:diff --git\b|index [0-9a-f]+\.\.[0-9a-f]+|@@|---\s+\S|\+\+\+\s+\S)`), 3, "diff"},
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
