# The roff Language Core

One-sentence summary: everything a manpage-to-markdown converter must understand
before touching the macro packages — control lines, requests, escapes, strings,
number registers, conditionals, user-defined macros, preprocessors, and the
parsing rules groff/mandoc apply to raw roff input.

> **Source of truth**: mandoc's `roff(7)` <https://mandoc.bsd.lv/man/roff.7.html>
> and groff's `roff(7)` <https://man7.org/linux/man-pages/man7/roff.7.html>.
> For macro-package semantics see [man-macros.md](man-macros.md) and
> [mdoc-macros.md](mdoc-macros.md).

## 1. Control lines and text lines

- A line beginning with `.` is a **control line**; the remainder is a request or
  macro call: `.<name> arg1 arg2 ...`.
- A line beginning with `'` is a **no-break control line**: same as `.` except
  that the implicit break some requests cause is suppressed. Treat `'` exactly
  like `.` when parsing.
- mandoc additionally recognizes the two-character form `\.` as a control
  character.
- All other lines are **text lines**, subject to filling, adjustment and
  hyphenation by the typesetter. Markdown converters care mainly about fill
  mode (see `.nf`/`.fi` below) and paragraph boundaries.
- A literal `.` or `'` starting a text line must be protected by a dummy
  character: `\&.NET is a framework.`.
- A lone control line containing only `.` is a no-op (often used as comment
  padding).
- Blank input lines cause a break and vertical space; conventional man pages
  avoid them, but parsers must handle them (map to paragraph break).

### Argument syntax

- Arguments are separated by spaces or tabs; leading spaces before the request
  name are ignored (`.  PP` works).
- Quoting: `"..."` groups a string with embedded spaces; `""` is an empty
  argument.
- Groff (and optional compatibility): `"..."` and `'...'` both quote in groff;
  historically only `"` quoted. Support both, but prefer `"` semantics on
  ambiguity.

## 2. Comments and line continuation

- `.\" rest of line` — whole-line comment; discard.
- `\"` at end of a text line — comment to end of line; discard.
- `\#` — groff whole-line comment; mandoc treats it like `\"`.
- A backslash as the **last character of an input line** joins the next line to
  the current one (`\newline` continuation). Handle this before any other
  parsing: read the logical line, not the physical line.

## 3. Escapes

Every escape starts with `\`. The converter needs to recognize all of them even
if it ultimately renders most to nothing (a markdown writer cannot express a
zero-width thin space).

| Escape | Meaning | Typical markdown fate |
|--------|---------|----------------------|
| `\e` | the escape character (backslash) | emit `\` (escaped) |
| `\-` | minus sign glyph | emit `-` |
| `\(ab` | two-character special glyph, e.g. `\(em`, `\(aq`, `\(bu`, `\(co` | glyph table lookup → char |
| `\[name]` | long-name glyph, e.g. `\[em]`, `\[ti]`, `\[u2014]` | glyph table lookup → char |
| `\&` | zero-width dummy character | delete |
| `\^`, `\|` | thin spaces (1/12 em, 1/6 em) | delete (zero-width on terminals) |
| `\0` | space of digit width | emit space |
| `\ ` (backslash-space) | unbreakable word space | emit space |
| `\~` | adjustable non-breaking space | emit space |
| `\%` | hyphenation point at word start | delete |
| `\:` | non-printing break point | delete |
| `\c` | continue current output line on next input line | hard-join the two source segments |
| `\t` | tab | emit space(s) or tab |
| `\n` | number-register interpolation (`\nX`, `\n(ab`, `\n[name]`) | expand |
| `\*` | string interpolation (`\*X`, `\*(ab`, `\*[name]`) | expand |
| `\f` | font switch (`\fB`, `\fI`, `\fR`, `\fP`, `\f[]`, `\f(BI`) | bold/italic/code state |
| `\s` | type size (`\s+2`, `\s-1`, `\s0`) | delete |
| `\m`/`\M` | stroke/fill color | delete |
| `\h'...'`, `\v'...'` | horizontal/vertical motion | delete (or space for `\h`) |
| `\w'...'` | width of argument | delete |
| `\z` | zero-width next glyph | delete, keep glyph |
| `\o'abc'` | overstrike glyphs | emit first/fallback glyph |
| `\b'abc'` | vertical bracket pile | emit glyph list |
| `\l'...'`, `\L'...'` | horizontal/vertical line drawing | emit dashes when terminal meaning applies |
| `\D'...'` | drawing command | delete |
| `\k`/`\R'name ±N'` | position capture / register assignment | expand (`\R` sets a register) |
| `\V[VAR]` | environment variable | delete or expand if safe |
| `\A'...'`, `\B'...'` | validity tests (truth values "1"/"0") | expand |
| `\N'n'` | glyph by index | delete or fallback |
| `\{` / `\}` | conditional grouping delimiters | structural |
| `\"`, `\#` | comments | delete |
| `\$1`..`\$9`, `\$(ab`, `\$$`, `\$*`, `\$@`, `\$0` | macro arguments / invocation name | expand only inside user macros |
| `\!`, `\?` | transparent-through / output comparison (diversions) | delete, high risk |
| `\p` | break output line | hard line break |
| `\(` bare two-char form, `\C'glyph'` | alternate special-glyph spellings | glyph table |

### Glyph table

Maintain a small table from mandoc's `chars.c` equivalents (documented in
groff's `groff_char(7)`). Common ones:

| roff | char |
|------|------|
| `\(em` / `\[em]` | — (em dash; markdown style: avoid, use `-` or keep as `-` per plain-hyphen rule) |
| `\(aq` | `'` |
| `\(ga` | `` ` `` |
| `\(bu` | `•` (or `*`) |
| `\(co` | © |
| `\(mi` | - |
| `\(ti` / `\(ha` | `~` / `^` |
| `\(lq` `\(rq` | `"` `"` |
| `\(dq` | `"` |
| `\(rs` | `\` |
| `\(aa`, `\(ff`, `\(fi` | combining accents, ligatures — approximate |

## 4. Strings

```roff
.ds NAME value
.ds NAME "  value with preserved leading spaces
.as NAME appended value
```

Interpolation: `\*N`, `\*(NM`, `\*[NAME]`, groff also `\*[NAME arg1 arg2]`.

Predefined man(7) strings: `\*R` (®), `\*S` (type size), `\*(lq`/`\*(rq`
(quotes), `\*(Tm` (™). `\*[.T]` is the device name.

**Strings are essential for real-world input**: pod2man preamble defines
`.ds --`, `.ds Aq`, `.ds C+`, etc., then interpolates them everywhere. A parser
that ignores `.ds`/`.as` and `\*` will corrupt pod2man output.

## 5. Number registers

```roff
.nr CNT 10 2
.nr CNT +1
.af CNT i
```

Interpolation: `\nN`, `\n(NM`, `\n[name]`.

Numeric expressions support `+ - * / % **`, parentheses, comparisons, `&`,
`:`, `!`, `?:`. Scaling suffixes: `i` inch, `c` cm, `p` point, `P` pica, `m` em,
`n` en, `u` basic unit, `v` line space. Default unit depends on the request; for
man indents it is `n`.

Registers a converter must model (with sensible defaults):

| Register | Default | Why it matters |
|----------|---------|---------------|
| `.g` | 1 in groff, 0 outside groff | pod2man preambles branch on `.if \n(.g` |
| `.H`, `.V` | device-dependent | pod2man conditionals |
| `rF`, `IX` etc. | user-defined | pod2man uses `.nr rF` as a flag |
| `.T` | device (mandoc sets `markdown`) | may leak into output |
| `.l`/`.LL` | line length | format decisions only |

Read-only introspection registers (`.g`, `.H`, `.V`, `.l`, `.$`, `.n`, `.d`,
`.p`, `.v`) should be predefined so conditionals referencing them evaluate
deterministically rather than erroring.

## 6. Conditionals

```roff
.if cond rest-of-line        " execute rest if true
.ie cond then-rest           " if/then/else via .ie/.el
.el else-rest
```

The "rest" is interpreted as an input line (text or request). Multi-line bodies
use `\{` ... `\}` grouping:

```roff
.ie n \{\
.  ds -- \(em
.\}
.el \{\
.  ds -- \(em
.\}
```

Conditions: numeric expressions (`'` quotes for string comparisons:
`'s1's2'`), `n` (nroff mode; assume **true** for terminal-friendly output),
`t` (troff mode; assume false), `e`/`o` (even/odd page; assume false),
`r name` (register defined), `d name` (string/request/macro defined),
`c char`, `F font`, `m color`, `S style`.

**Key decision**: pick a fixed "environment" (typically nroff-like, ASCII
terminal) and evaluate conditionals against it. Groff's man.local and pod2man
preambles use `.if n ... .el ...`. Groff's `\n[.g]` is available; mandoc does
the same.

`.ig`: ignore block until `..` or optional end macro name:
`.ig END` ... `.END`. Discard the body entirely.

`.while`, `.break`, `.continue` are loop constructs; mandoc implements while
and break. For a converter, evaluate the loop (bounded iterations to avoid
infinite loops on malformed input).

## 7. User-defined macros (.de/.am) — mandatory

`.de NAME` ... `..` defines a macro; `.am` appends; `.de1`/`.am1` enforce
compatibility mode. Inside the body, arguments are available via `\$1`..`\$9`,
`\$(ab`, `\$[nnn]` (numbered args), `\$*` (all args space-separated), `\$@`
(all args quoted), `\$^` (args as-if for `ds`); `\$0` is the invocation name.
Bodies are parsed in **copy mode**: most escapes are stored literally and
interpreted at call time; `\\$1` in the body means `\$1` when invoked.

`.shift` discards the first macro argument, renumbering the rest (decrementing
`.n(.$`). `.return` exits the macro early.

**This is not optional**: virtually every pod2man page defines `.Sp`, `.Vb`,
`.Ve`, `.IX` and uses them inline. Without executing `.de` and expanding calls,
`.Vb 10` produces noise instead of a code fence opening and verbatim block.

When to expand: queue the call's arguments, then interpret the body lines
exactly as if typed there, with argument substitution. Cap depth/iteration to
guard against malformed input.

## 8. Requests a man converter must implement (vs ignore)

| Request | Behavior | Converter handling |
|---------|----------|-------------------|
| `.nf` / `.fi` | no-fill / fill mode toggle | track fill mode; verbatim blocks |
| `.sp [n]` | vertical space | paragraph break (or line break) |
| `.br` | break output line | hard line break |
| `.in [n]` / `.ti` / `.po` | indent / temporary indent / page offset | track numerically for list nesting |
| `.ll [n]` | line length | parse, ignore |
| `.ta [stops]` / `.DT` | tab stops / reset | parse, ignore (or approximate tabs) |
| `.ce`/`.rj` | center / right justify | parse, ignore |
| `.ad`/`.na`/`.nh`/`.hy` | adjust/hyphenation control | parse, ignore |
| `.ne`, `.mk`, `.rt`, `.bp`, `.pl`, `.vs`, `.ls`, `.ss` | page/spacing control | parse, ignore |
| `.ds`/`.as` | define/append string | implement |
| `.nr`/`.af`/`.rr`/`.R`' | registers | implement |
| `.de`/`.am`/`.de1`/`.am1` | macros | implement |
| `.ig` | skip block | implement |
| `.if`/`.ie`/`.el` | conditionals | implement |
| `.so [file]` | include file | implement only if safe (relative path, in-tree); else parse-and-ignore with a note |
| `.tr`/`.char`/`.cc`/`.ec`/`.eo` | translation / control char change | `.tr` useful (pod2man uses `.tr \(*W-`); implement basic versions |
| `.sh`, `.pi`, `.open`, `.write`, `.cf`, `.nx` | shell exec / output files / restart | insecure; parse-and-ignore |
| `.ft` | font change (equivalent of `\f` is used by `.Vb` via `.ft CW`) | implement for font state |
| `.liest`, `.ds`, `.rr`, etc. | misc | ignore |
| `.TH`, `.SH`, `.SS`, `.TP`, `.IP`... | man(7) macros | see man-macros.md |
| `.Dd`, `.Sh`, `.Bl`... | mdoc(7) macros | see mdoc-macros.md |
| unknown macro names | groff errors, mandoc defines empty | **treat as no-op and drop** — this is what makes pod2man's `.IX` harmless (mandoc explicitly ignores `.IX` too) |

## 9. Preprocessor blocks

The converter usually meets man pages **after** preprocessing—`.TS/.TE`
(tbl), `.EQ/.EN` (eqn), etc. blocks remain in the source as scaffolding.
Practical handling:

| Pair | Preprocessor | Converter approach |
|------|--------------|-------------------|
| `.TS` ... `.TE` | tbl(1) tables | ASCII-art approximation or a fenced block; tables are data-rich and semantically valid to preserve, but scoping a tbl mini-parser is large — start with fall-through to a code block and note "tbl data" |
| `.EQ` ... `.EN` | eqn(1) math | fenced block containing the eqn source, or skip |
| `.PS` ... `.PE` | pic(1) diagrams | skip or fenced block with "pic data" note |
| `.G1` ... `.G2` | grap(1) charts | skip |
| `.R1` ... `.R2`, `.[`/`.]` | refer(1) citations | skip |
| `.cstart` ... `.cend` | chem | skip |

Preprocessor source blocks can be detected by the same macro-pair logic; the
parser only needs to enter a "capture raw until end pair" state and decide on
presentation—as a fenced code block preserves information losslessly.

## 10. Fill mode and tabs

`.nf` disables filling; each input line is emitted as-is. `.fi` re-enables.
Man pages use `nf/fi`, `.EX/.EE` (groff/mandoc: equivalent), `.Vb`/`.Ve`
(pod2man expands to `.ft CW`/`.nf`/`.ne`), etc. The markdown mapping for both
is the indent-4-space code block (or fenced block, your choice).

## 11. Fill-mode-independent plumbing rules for a converter

Even without typesetting:

1. Read logical lines (continue over backslash-newline).
2. Detect control char: `.`, `'`, or `\.`.
3. Strip comments (`\.#` and trailing `\"`/`\#`).
4. Expand: user macro args (`\$`) only inside macro bodies; strings (`\*`),
   registers (`\n`), then escape recognition.
5. Dispatch: known request table → roff handler; known macro table → macro
   handler; user-defined macro → expand; unknown → drop with a note.
6. State machine tracks: fill mode, font stack (bold/italic/roman), outdent
   depth (`.in`, `.RS/.RE`, `IP/TP/HP` indents), pending next-line trap
   (`.SH`, `.SS`, `.TP`, `.B/.I/.SM/.SH/.SS` with no arguments capture the
   **next text line**), list state.

These horizontal/vertical spacing details (converting `\h'...'`, `\v'...'`)
can be dropped: markdown cannot express them.

**Trap**: a `.de` body in copy mode contains no actual expansion; expansion
must be deferred to invocation with the invocation environment (font state at
definition time is usually not restored — treat `.Vb/.Ve` style by tracking
font state at call time).

## References

- mandoc roff(7): <https://mandoc.bsd.lv/man/roff.7.html>
- groff roff(7): <https://man7.org/linux/man-pages/man7/roff.7.html>
- groff_char(7) glyph table: <https://man7.org/linux/man-pages/man7/groff_char.7.html>
- mandoc source (reference for behavior; do **not** copy into the MIT module):
  `mdoc_markdown.c`, `roff.c`, `read.c` at <https://cvsweb.openbsd.org/src/usr.bin/mandoc/>
- tbl(1): <https://man7.org/linux/man-pages/man1/tbl.1.html>
- eqn(1): <https://man7.org/linux/man-pages/man1/eqn.1.html>
