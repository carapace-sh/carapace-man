package manpages

import (
	"strings"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	cases := []struct {
		name string
		code string
		want string
	}{
		{"shell shebang", "#!/bin/sh\nset -e\necho hi\n", "sh"},
		{"bash shebang", "#!/usr/bin/bash\nif [ -f x ]; then\n  export FOO=y\nfi\n", "sh"},
		{"shell prompt", "$ ls -la\ntotal 0\n$ du -h\n", "console"},
		{"percent prompt", "% make install\n", "console"},
		{"shell keywords", "for f in *.go; do\n  echo \"$f\"\ndone\n", "sh"},
		{"usage line", "ls [OPTION]... [FILE]...\n", "console"},
		{"c include", "#include <stdio.h>\nint main(void) {\n  printf(\"hi\");\n  return 0;\n}\n", "c"},
		{"c struct", "typedef struct {\n  unsigned long size;\n} meta;\n", "c"},
		{"c typedef struct comments", "typedef struct {\n    int displayed_index;\t/* which buffer is being displayed */\n    int update_action;\t/* Undefined, Background, Untouched, Copied */\n    Multibuffer *buffers;\t/* Buffers */\n} XmbufWindowAttributes;\n", "c"},
		{"c typedef struct simple", "typedef struct {\n    int update_hint;\t/* Frequent, Intermittent, Static */\n} XmbufSetWindowAttributes;\n", "c"},
		{"cpp class", "#include <string>\nnamespace demo {\nclass Widget {\npublic:\n  void run();\n};\n}\n", "cpp"},
		{"perl shebang", "#!/usr/bin/perl\nuse strict;\nmy $x = 1;\n", "perl"},
		{"perl vars", "my $self = shift;\nour $VERSION = '1.0';\nuse POSIX qw(strftime);\n", "perl"},
		{"python shebang", "#!/usr/bin/python3\nimport os\n\ndef main():\n  pass\n", "python"},
		{"python env shebang", "#!/usr/bin/env python3\nimport sys\n\ndef main():\n  return 0\n", "python"},
		{"go file", "package main\n\nimport \"fmt\"\n\nfunc main() {\n  fmt.Println(1)\n}\n", "go"},
		{"rust", "use std::fmt;\n\nfn main() {\n    let mut name = String::from(\"hi\");\n    println!(\"{}\", name);\n}\n", "rust"},
		{"ruby", "#!/usr/bin/env ruby\nrequire 'json'\n\ndef greet(name)\n  puts name\nend\n", "ruby"},
		{"javascript", "import fs from 'fs';\n\nconst run = () => {\n  console.log('hi');\n};\nexport default run;\n", "javascript"},
		{"typescript", "interface User {\n  name: string;\n  active: boolean;\n}\nexport type ID = string;\n", "typescript"},
		{"sql", "SELECT id FROM users;\nINSERT INTO t (a) VALUES (1);\n", "sql"},
		{"make", ".PHONY: all\nall: build\n\tgo build ./...\n", "make"},
		{"dockerfile", "FROM alpine:3.20\nWORKDIR /app\nCOPY . .\nRUN echo hello\nCMD [\"./app\"]\n", "dockerfile"},
		{"yaml", "---\nname: demo\nitems:\n  - one\n  - two\n", "yaml"},
		{"toml", "[package]\nname = \"demo\"\nversion = \"0.1.0\"\n\n[[bin]]\nname = \"app\"\n", "toml"},
		{"ini", "[core]\nrepositoryformatversion = 0\n[remote \"origin\"]\nurl = git@x:y\n", "ini"},
		{"json", "{\n  \"a\": 1,\n  \"b\": [2]\n}\n", "json"},
		{"xml", "<?xml version=\"1.0\"?>\n<config>\n  <item>1</item>\n</config>\n", "xml"},
		{"html", "<!DOCTYPE html>\n<html><body>x</body></html>\n", "html"},
		{"diff", "--- a/file\n+++ b/file\n@@ -1 +1 @@\n-old\n+new\n", "diff"},
		{"powershell", "param(\n  [string]$Name\n)\nGet-Item env:PATH\nWrite-Host \"hi $Name\"\n", "powershell"},
		{"unlabeled prose", "just some\nplain explanation\n", ""},
		{"mixed below threshold", "x = 1\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectLanguage(strings.Split(c.code, "\n"))
			if got != c.want {
				t.Errorf("detect = %q, want %q", got, c.want)
			}
		})
	}
}

func TestDetectLanguageSynonyms(t *testing.T) {
	if got := detectLanguageFor([]string{"ls [OPTION]..."}, "SYNOPSIS"); got != "console" {
		t.Errorf("synopsis hint: got %q", got)
	}
	if got := detectLanguageFor([]string{"x = 1"}, "DESCRIPTION"); got != "" {
		t.Errorf("plain hint: got %q", got)
	}
}
