# Chroma's lexer.Analyse (dependency research)

One-sentence summary: what `alecthomas/chroma`'s content-analysis API does,
how it scores and selects lexers, and what adopting it for our fence
language labeling would cost — decision input for replacing the
hand-rolled heuristic in `pkg/manpages/syntax.go`.

> **Source of truth**: <https://github.com/alecthomas/chroma> (v3 master;
> the Analyse API is unchanged since v2). For the current heuristic see
> [markdown-mapping.md](markdown-mapping.md) "Fence language labels" and
> `pkg/manpages/syntax.go` in the carapace-man repo.

## 1. API shape

```go
lexer := chroma.NewLexerRegistry().Analyse(text)   // registry.go
lexer := lexers.Analyse(source)                            // lexers package wrapper
```

- `Lexer.AnalyseText(text) float32` returns a 0.0–1.0 confidence; lexers
  register it via `SetAnalyser(func(text string) float32)` or, for embedded
  XML lexers, via an `<analyse>` config element.
- `Registry.Analyse` is a **linear scan over every registered lexer**,
  taking the strictly-highest score (`>` comparison, so ties go to registry
  order). No lexer scoring above 0 → returns **nil**, not a fallback.

## 2. Scoring semantics

Two modes for XML lexers (`xml.go`):

| Mode | Semantics |
|------|-----------|
| default (sum) | every matching regex's `score` adds up, capped at 1.0 |
| `first="true"` | the **first** regex in document order that matches decides (list order, not best score) |

Key properties:

- **No occurrence weighting** — `regexp2.MatchString` is boolean; a pattern
  appearing 100 times scores like one appearing once. Repeated-token
  weighting requires a custom analyser (`FindAllString`).
- Scores above 1.0 are clamped only for the XML analyser; custom
  `SetAnalyser` values are unclamped.
- Analyser regexes inherit only `CaseInsensitive` from config — write
  `(?m)`/`(?s)` inline; invalid regexes silently score 0; **no match
  timeout** on analyser regexes (tokenisation rules use 250 ms).
- Analysers are configured lazily (`sync.OnceValue` compiles regexes on
  first `AnalyseText`).

## 3. Example analyser configs (upstream)

```xml
<analyse first="true">
  <regex pattern="(?m)^#!.*/bin/(?:env |)(?:bash|zsh|sh|ksh)" score="1.0"/>
</analyse>
```

Code-based analysers are string-subset checks, e.g. Go:

```go
if strings.Contains(text, "fmt.") && strings.Contains(text, "package ") {
    return 0.5
}
```

Upstream behaviour samples: `#!/bin/bash` → Bash 1.0;
`package main` + `import "fmt"` → Go 0.5 (0.1 alone); php inside html →
PHTML 0.5; plain prose → nil.

## 4. Gotchas

- `Config.Priority` is **ignored** by `Analyse` (only `Match`/`Get` use it).
- **Delegating/wrapper lexers forward analysis to their root**; several
  wrappers sharing a root overwrite each other's analyser (`SetAnalyser`
  on the root wins).
- Analysers are heuristics, not parsers — ambiguous snippets misclassify;
  prefer filename hints and fall back to `lexers.Fallback` (which we
  currently model as "unlabeled fence").
- Full dispatch costs a scan of all embedded lexers on every call;
  sample a prefix instead of an entire large page when calling repeatedly.

## 5. Adoption trade-offs for carapace-man

| | Chroma-based detection | Current heuristic (`syntax.go`) |
|---|---|---|
| Coverage | ~200 lexers with tuned per-language analysers | 11 language families, hand-tuned |
| Directness | returns a `chroma.Lexer`, whose `Config.Name`/aliases map to fence identifiers (`Bash`→`bash`, `console` exists as `shell-session`/`console` alias) | returns the string directly |
| Cost | new MIT-licensed dependency; embedded lexer tables grow the binary by a few MB; adds `regexp2/v2` | zero dependencies |
| Determinism | rank via scores only: ties break on registry order | deterministic by construction |
| Fit | powerful for mdbook's highlighting targets (mdbook itself uses syntect, so fence identifiers only need to be lexically common) | conservative: only labels when a language *scoring threshold is crossed* |

Recommendation options (unchanged until decided):

- **A. Keep** the heuristic; adopt chroma only if mdbook book rendering
  quality matters more than binary size.
- **B. Adopt** `lexers.Analyse` behind a narrow seam: `manpages` exposes an
  optional `LanguageDetector func([]string) string` hook on `ParseOptions`
  (or a settable package var), defaulting to the internal heuristic;
  `pkg/mdbook` wires the chroma detector into parsing so only the book
  build pays the dependency cost.
- **C. Port** chroma's analyser ideas (per-language regex-sets with
  confidence, occurrence weighting) into the heuristic without the
  dependency.

## Related docs

- [markdown-mapping.md](markdown-mapping.md) — current fence labeling.
- [edge-cases.md](edge-cases.md) — corpus feedback on label distribution.
