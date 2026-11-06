# Edge Cases Found in the Wild

One-sentence summary: catalog of edge cases discovered by probing the
fetch-scale `carapace-man` corpus (~34k pages from `/usr/share/man`) and the
manpages skill references, with the state of each in `pkg/manpages`.

> Probe: `MANPAGES_PROBE=1 go test ./pkg/manpages -run TestProbeFullCorpus`
> (see `corpus_test.go` in the carapace-man repo). It parses the entire
> system man database and reports errors, panics, and raw roff directives
> leaked outside code fences.

## 1. State from the last probe

- 34077 pages: **0 failures, 0 panics, 0 directive leaks** outside code fences.
- Bases where leaks were previously reported (legit content inside code
  fences like `xxd`'s hexdumps, `groff.7` examples) turned out to be false
  positives once fences were accounted for.

## 2. Crash- and hang-class edge cases (all fixed)

| Edge case | Trigger | Status |
|-----------|---------|--------|
| `.It` outside an open `.Bl` | any mdoc page with a stray `It` | guarded: treated like a plain item |
| Numeric `5-3`/`x-1` expressions (binary minus directly before a digit with no space) | `.nr`/`.if` conditions in real pages | hang in `numParser.parseAdditive` semantic threshold — fixed by treating `-` as a binary operator; unary minus still handled in `parseUnary` |
| Numeric register references in conditions with `-` after an `(`-expression | `\n(x)-1`-style conditions | covered by the binary fix above |

## 3. Edge cases now covered by the converter

| Edge case | Construction | Behavior |
|-----------|--------------|----------|
| tbl/eqn/pic/grap/refer/chem regions | `.TS/.TE`, `.EQ/.EN`, `.PS/.PE`, `.G1/.G2`, `.R1/.R2`, `.cstart/.cend` | captured raw and emitted as a neutral code block (`expand.go::preprocessorBlock` → `writer.PreprocessorBlock`), per the plan's "detected and skipped" rule; the `.TS` option line, `.T&`, and tab/cell syntax stay verbatim |
| OSC-8 device extensions in modern groff pages (`\X'tty: link URL'content\X'tty: link'`) | coreutils pages, recent man-db output | rendered as `[content](URL)` (`interpolate.go::ttyLinks`); leftovers of the closing token are stripped |
| `.de` capture lines ending with `\" comment` | pod2man preambles | comment stripped, the remaining token stays the macro name; a non-token name never becomes the end-marker |
| `'br\}`-style group closers | pod2man `.ie/.el` groups where the break closes the group | the `br` request executes and the closure terminates the body |
| Dotted integer list marks | `.IP "1.\ First"` | ordered item with the number preserved; bare numeric marks (`.IP 4`) stay plain paragraphs (indent semantics, not items) |
| Bullet glyphs beyond `\(bu` | `\(bu`, `\(**`, `\(sr` | first familiy mapped to `- ` output; unknown markers rotate them out rather than leaking glyphs |
| `.B "text"` quoted multi-word | `pod2man` output, most core pages | one bold span |
| `Name(section)` in pre-section content (NAME region) and SEE ALSO | carapace UID ecosystem | linkified with `man://topic/section` (bold-marker aware; section tokens must match `\d`, `0`, `n`, `l`, optional `p`, `3p`, `2p`) |
| `.TH` date/source | conventional man-mm-dd formatting | captured into an HTML comment metadata header |
| Markdown-hostile text | `<`, `>`, `\`, line-leading `#`, `+`, `-`, `*`, `=` and `1.`-style prefixes | metacharacter-escaped (`escape.go`), `<`/`>` become entities |
| mdoc angle/enclosure delimiters | `.Aq x`, `.Ao x Ac`, `.In stdio.h` (both inside and outside SYNOPSIS) | the delimiters added by `mdoc7.go` are themselves entity-escaped (`&lt;`/`&gt;`) — the inner text was escaped but the surrounding `<`/`>` leaked raw, tripping mdbook's "unclosed HTML tag" linter (found via a ~34k-page mdbook build) |
| `\c` no-space joins | `.B "a\\c"` + next word | carried through font wraps via an internal marker and joined without space |
| User macros without expansions defined by pod2man | `.IX`/`.IX Header` | dropped as undefined macros (mandoc-compatible) |
| `.ig` with a defined end macro | `.ig END` | body skipped; missing terminator drops to EOF (no error) |
| Nested mdoc scopes left open | hand-written pages | closed at EOF by `mdoc.finish()` |

## 4. Known approximations (documented behavior, not bugs)

| Area | Notes |
|------|-------|
| `.so` file inclusion | requests are dropped; the native `man-to-md` path only ever handles a single located page |
| `-line` vertical spacing, hyphenation, adjustment | parsed and ignored (no typesetting semantics in markdown) |
| mdoc `-column` lists | rendered as a code block with three-space cell separation (mandoc does the same) |
| mdoc multi-line enclosures (`Oo/Oc`, `Eo/Ec`, `Qo/Qc` spanning lines) | open/close tokens handled inline; the span is approximated when bodies cross lines (\emph{see `mdoc-macros.md`}) |
| `Xt`-style `\u2014` hybio glyph fidelity | rendered via the glyph table; uncommon names fall back to silence (e.g. `\(ff` ligature approximations) |
| `.Ys`/`.SY` | emitted inline (per-line) rather than as a single reflowed usage block |

## References

- Full corpus probe: `pkg/manpages/probe_test.go` in the carapace-man repo.
- [parser-design.md](parser-design.md) for the hardening checklist that
  seeded these cases (items 1–10).
- [roff-core.md](roff-core.md), [man-macros.md](man-macros.md) and
  [mdoc-macros.md](mdoc-macros.md) for the underlying language semantics.
