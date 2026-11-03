# Parser Design in Go

One-sentence summary: a concrete architecture for a from-scratch,
MIT-licensed Go converter — dialect detection, the roff expansion layer,
the macro/state layer, and the failure modes of real-world pages
(pod2man above all) that the design must survive.

> Clean-room rule: build behavior from the **documented specs** ([roff-core.md](roff-core.md),
> [man-macros.md](man-macros.md), [mdoc-macros.md](mdoc-macros.md)) and
> mandoc's own man pages, not from reading and translating GPL/other code.
> See [existing-tools.md](existing-tools.md) for per-project license status.

## 1. Pipeline overview

```
io.Reader
  | logical-line reader (backslash-newline joins, CR/LF normalization, optional UTF-8)
  v
roff layer            (control-char detection, comments, request dispatch,
                      .ds/.nr/.de/.ig/.if/.ie/.el/.tr evaluation)
  | exposes: control lines with expanded args OR raw text lines with
  |          fill-mode / state context
  v
dialect layer         (once the first macro is .TH or .Dd/.Dt: dispatch to
                      man(7) or mdoc(7) macro tables; before that: buffer or
                      look ahead lazily)
  |
  v
state layer           (fill mode, font stack, indent stack, list stack,
                      pending next-line trap, current heading level,
                      paragraph need, current section name)
  |
  v
markdown writer       (node-free streaming writer deciding when to open
                      paragraphs/lists/code fences; metacharacter escaping)
```

No AST needed: man documents are read top-to-bottom and state from earlier
content depends only on macros already seen; a **streaming writer with a
stack of open scopes** is simpler and adequate. (mandoc builds a tree because
it also validates and reorders; a converter does not need to.)

## 2. Logical-line reader

- Join backslash-terminated input lines before anything else; cap the join
  length to defend against runaway input.
- Normalize CRLF/CR if desired.
- Identify each logical line as one of text-line / control-line.
- Detect the dialect lazily: scan forward for the first line whose
  control character is followed by `TH` or `Dd`/`Dt`; before that, keep a
  bounded lookahead buffer. Groff uses `andoc.tmac`; mandoc uses
  `choose_parser()`-style look-ahead. Fall back to man(7) if unseen.

## 3. Tokenization and argument parsing

- Control line grammar: `[.'] name args...`, name is 1-3 chars for mdoc,
  arbitrary for requests/macros in man land. roff request names can be two
  characters (`.ds`, `.nr`, `.in`) or more (`.ig`, `.so`, `.if`; there are
  also `.ds1`, `.nr1` forms in mandoc).
- Arguments: split on space/tab unless quoted with `"..."` (and `'...'` in
  groff-compat environments). Empty args via `""`.
- The ambiguous v7 man mode where a single argument is the rest of the line?
  Do that keyed by macro: `.SH`, `.SS`, `.TP` and no-extra-args forms of
  `.B/.I/.SM/.SB` are the ones in practice where quoting matters.

## 4. Expansion order (crucial)

1. **Unquote** (strip protective quotes; keep body verbatim).
2. **Macro-argument interpolation** `\$1`..`\$9`, `\$*` etc. — innermost macro
   body only.
3. **String interpolation** `\*(name)` / `\*N` — recursive (a string may
   contain more interpolations). Guard recursion depth.
4. **Register interpolation** `\n[name]` / `\nN` with numeric expression
   evaluation for `\R`, `.nr`, `.if`.
5. **Request/conditional evaluation** — `.if`/`.ie` `.el` dispatch on boolean
   value.
6. **Escape recognition** in text passes — map each documented escape; delete
   the "presentation-only" ones; the writer decides bold/italic boundaries
   via the font state machine.

Note that roff evaluates expressions differently depending on request; a
middle path: implement a small evaluator supporting integers, `+ - * / %`,
comparisons, boolean `&&`, `||`, `!`, `( )`, unit suffixes (`n`, `m`, `i`,
`v`, `p`), and the `n`/`t` environmental keywords — good enough for the
groff/documentation stack in practice.

## 5. State machine you must model

State that survives across lines:

| State | Set by | Markdown impact |
|-------|--------|----------------|
| fill mode | `.nf/.fi/.EX/.EE/.Vb/.Ve` | code block open/close |
| font stack | `.B/.I/.SM/(alternates)/.ft/\f` | `**bold**`, `*italic*`, code |
| pending next-line trap | `.SH/.SS/.TP/.B/.I/.SM/.SB/.SY` with no args | next text line is consumed as the tag/heading/argument |
| current indent (`in`) | `.IP/.TP/.HP/.RS/.RE/.in/.ti`, mdoc `-offset/-width` | list nesting depth |
| pending list type | `.IP "\(bu"` bullets, `.TP`, mdoc `.Bl` types | list emission discipline (same marker → same list; a new marker → new list) |
| current heading level | `.TH/.SH/.SS` / `.Sh/.Ss` | `#`/`##`/`###` levels |
| current mdoc scope stack | `Bd/Ed`, `Bl/El`, `Bk/Ek`, `Bf/Ef`, `Oo/Oc`, `Fo/Fc`, `Rs/Re` | push/pop scope, close-on-EOF |
| current section name | last `.SH`/`.Sh` | SYNOPSIS-specific rendering in mdoc |
| current mdoc list type | `.Bl -tag`, `-enum`, ... | item discipline |

Recognize the `"\(bu"` / `"\(**"` bullet markers so `.IP "\(bu"` sequences
become `* body text` list items; any other marker becomes a bolded inline
prefix or a plain item — collapse unusual markers rather than emitting them
verbatim.

## 6. Markdown output decisions

Emit list items as contiguous markdown list lines; list nesting maps to
the indent column. Trailing whitespace cleanup and reflow discretion:
- In **fill mode**, prose lines are free-form: wrap at a configurable width
  (or never wrap). Text state within a paragraph can be merged.
- In **nf mode**, emit exactly the input lines (after escape processing) as
  a fenced code block; add a friendly delimiter when the input line begins
  with `` ` `` or has odd trailing spaces.

`.Vb`/`.Ve` (pod2man) and `.Bd -literal`/`.Dl` (mdoc) play the same role;
fence both rather than reflow.

## 7. Hardening / failure modes to handle in code

1. **Undefined macro use**: treat as a no-op (mandoc also does this); a
   converter must not die on `.IX`, `.IX Title "..."`, or any other macro
   emitted by generated pages.
2. **Unclosed scopes**: closing all open `.Bl/.Bd/.RS/.Ss` scope at EOF.
3. **Ignored-block mis-termination** (`.ig` vs `.ig END` delimiter naming):
   on a missing terminator, discard until EOF.
4. **Odd quoting**: unmatched `"` in an argument line — treat the remainder
   of the line as one argument.
5. **Escapes in headings**: `.SH NAME\-MORE` should never leave a backslash
   in the heading: process escapes in heading text like anywhere else.
6. **Infinite `.de` recursion** (a macro calling itself): cap macro-body
   recursion depth (e.g. 100) and total generated lines.
7. **`.so man.local` / `.so $srcdir$`**: if the file exists within the same
   directory, include it; otherwise skip with a note (mandoc does the same:
   relative paths only, no `../`).
8. **The `.$` macro-argument-count register** must exist so generated pages
   that test `\n(.$` behave.
9. **Text starting with `\&.` within prose** (declarations such as
   `\&.NET`... `.EX` blocks restated as `\& =pod`): the `\&` is consumed,
   `.` becomes literal text.
10. **Trailing `\c`** in a `.TP` tag continues the tag with the next line —
    join segments in the writer, not the tokenizer.

## 8. Testing strategy

- Build a corpus: Debian/man-pages pages, OpenBSD (`/usr/share/man`),
  X11 docs, git's `Documentation/git-*.adoc`-exported pages, a large set of
  **pod2man output from a Perl install** (the worst case), and the pages in
  the `github.com/carapace-sh/man` repo that carapace-man syncs.
- Property test: run the converter over every file with an "exit 0 / no
  panic / no raw roff directives left" assertion; over time, keep the
  "raw directive escaping percentage" metric to track regressions.
- Golden tests comparing specific documents man-file → markdown.
- Diff against `mandoc -T markdown` where the input is mdoc (mandoc cannot
  read man(7) into markdown), against `pandoc -f man` for sanity — external
  tools as cross-checks only, never as vendored code.

## Related docs

- [roff-core.md](roff-core.md) for the request/escape layer details.
- [man-macros.md](man-macros.md) and [mdoc-macros.md](mdoc-macros.md) for the
  macro tables to feed the dispatcher.
- [markdown-mapping.md](markdown-mapping.md) for the writer's mapping and
  escaping rules.
