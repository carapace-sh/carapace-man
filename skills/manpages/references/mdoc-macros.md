# mdoc(7) Macro Language

One-sentence summary: the **semantic** BSD macro system for man pages — macros
describe what something is (flag, path, errno), not how it looks — with the
block/list machinery and prologue the converter must handle.

> **Source of truth**: <https://man.openbsd.org/mdoc.7> and FreeBSD
> <https://man.freebsd.org/cgi/man.cgi?query=mdoc&sektion=7&format=ascii>.
> For mandoc's rendering behavior see [markdown-mapping.md](markdown-mapping.md);
> for raw roff see [roff-core.md](roff-core.md).

## 1. Prologue

Exactly three macros open every mdoc page, in order:

```roff
.Dd $Mdocdate$          " or 2026-03-14 or "March 14, 2026"
.Dt PROGNAME 1          " optional arch: .Dt FOO 9 i386
.Os                     " optional system/version text
```

`Dd` is the first macro of the file; mandoc's dialect detection keys on
`Dd`/`Dt` being the first macro (vs `.TH` for man).

## 2. Structure and text macros

| Macro | Role |
|-------|------|
| `.Sh TITLE` | section heading (H1-level in mandoc -T markdown: `# TITLE`) |
| `.Ss TITLE` | subsection heading (`## TITLE`) |
| `.Pp` | paragraph break; not needed around `Sh`/`Ss` or after lists unless `-compact` |
| `.Bd [-type]` ... `.Ed` | display block: `-literal`/`-unfilled`/`-ragged`/`-centered`/`-filled`/`-offset val`/`-compact`; literal and unfilled types become code blocks |
| `.D1 text` | one-line indented display ⇒ blockquote in markdown output |
| `.Dl text` | one-line indented literal display ⇒ code block |
| `.Bk -words` ... `.Ek` | keep-words block (prevent internal breaking; markdown uses `&nbsp;` in mandoc) |
| `.Bf [-Em|-Sy|-Li]` ... `.Ef` | font block (emphasis/symbolic/literal) |
| `.Eo cc` ... `.Ec [termp]` | generic enclosure pair |

## 3. Semantic in-line macros (commands)

| Macro | Meaning | Typical text |
|-------|---------|-------------|
| `.Nm [name]` | program name; with no arg reuses previous; in SYNOPSIS, starts a suppressed line | bold |
| `.Nd text` | one-line description (only in NAME) | `- text` |
| `.Fl [l]` | command-line flag; each argument gets a `-` prefix | bold `-flag` |
| `.Ar [arg]` | argument placeholder; default `file ...` | italic |
| `.Cm str` | literal modifier/keyword passed to a command | bold |
| `.Ic str` | internal/interactive command | bold |
| `.Op body` | optional element, brackets `[ ... ]` | brackets |
| `.Oo` ... `.Oc` | multi-line optional block |
| `.Ns` | suppress the space to the next macro (glue: `.Fl o Ns Ar file` → `-ofile`) |
| `.No text` | normal (roman) — closes in-line macro scope / forces space |
| `.Pa [path]` | filesystem path | italic |
| `.Ev var` | environment variable | code (mandoc) |
| `.Dv const` | defined constant / `#define` | code |
| `.Er errno` | value of `errno` (ERRNOS) | code |
| `.Cd str` | kernel configuration input | bold |
| `.Ad addr` | memory address | italic |
| `.Ms sym` | math symbol | bold |
| `.Tn str` | trade name (rarely used) | code in mandoc |
| `.Mt addr` | mail address → `[text](mailto:addr)` |
| `.Bt`, `.Ex str`, `.Rv` | builtin return/exit value templates; mostly plain text |
| `.Lk uri [display]` | generic link | `[display](uri)` |
| `.Xr name [section]` | cross reference `name(section)` — plain text in markdown |
| `.Sx sec` | internal section cross ref | italic |
| `.Tg [key]` | definition tag (hidden; index/tag only) |
| `.An [author]` | author name; splitting control |
| `.At`, `.Bx`, `.Ux`, `.Ox` | system names (Unix, BSD, ...) |
| `.St std` | standards text (p1003.1 ...) |

## 4. Semantic in-line macros (libraries, section 2/3 SYNOPSIS)

| Macro | Meaning |
|-------|---------|
| `.Ft type` | function return type |
| `.Fn name arg...` | function invocation; args comma-separated and quoted if they contain spaces; in SYNOPSIS starts a new output line |
| `.Fo name` ... `.Fa arg` ... (repeat `Fa`) ... `.Fc` | multi-line function block |
| `.Vt type` | variable type; in SYNOPSIS also declares globals |
| `.Va name` | variable name (outside SYNOPSIS) |
| `.In header` | include file (SYNOPSIS: `#include`) |
| `.Fd directive` | preprocessor directive |
| `.Fa [argtype [argname]]` | function argument / struct field |

## 5. Enclosure macros

In-line form (delimit one argument pair) / block form pair:

| In-line | Block | Output |
|---------|-------|--------|
| `.Dq text` | `.Do`...`.Dc` | typographic “...”|
| `.Qq text` | `.Qo`...`.Qc` | "..." |
| `.Sq text` | `.So`...`.Sc` | ‘...’ |
| `.Pq text` | `.Po`...`.Pc` | (...) |
| `.Bq text` | `.Bo`...`.Bc` | [...] |
| `.Brq text` | `.Bro`...`.Brc` | {...} |
| `.Aq text` | `.Ao`...`.Ac` | ⟨...⟩ |
| `.Oo`/`.Oc` | see `.Op` above | [...] |

## 6. Lists: `.Bl` / `.It` / `.El`

```
.Bl -tag -width Ds
.It Fl v
Print verbose information.
.El
```

`Bl` requires a list type as the **first** argument:

| Type | Heads (`It` args)? | Markdown rendering (mandoc -T markdown) |
|------|-------------------|------------------------------------------|
| `-bullet` | no heads | `*` bullet list |
| `-dash` (alias `-hyphen`) | no heads | `-` bullet list |
| `-enum` | no heads | `1.`, `2.`, ... (capped at 99) |
| `-item` | heads allowed, invisible | plain text with line breaks |
| `-inset` | heads on same line | line breaks / indentation |
| `-diag` | heads unparsed (DIAGNOSTICS) | bold/heading-ish; no body indent |
| `-hang` | head, body follows on same line | like tag |
| `-ohang` | head; body starts on **next line** at column 0 | close to plain paragraphs |
| `-tag` | head, body indented `-width` | tag + blockquoted body |
| `-column` | cells separated by `.Ta` or tab | indented code block with column padding |

Modifiers: `-compact` (tight spacing), `-width val` (`6n` default for tag,
`Ds`/`6n` is the accepted string for it), `-offset val` (outer indent).

`.It` argument styles: heads allowed only in `-hang/-ohang/-inset/-diag`;
`-tag` optionally; `-column` requires cells (`It a Ta b`).

Nested lists are accepted; keep a list-stack in the parser.

## 7. Conventional `.Sh` section order

NAME, (LIBRARY, rarely), SYNOPSIS, DESCRIPTION, CONTEXT (section 9 only),
IMPLEMENTATION NOTES, RETURN VALUES, ENVIRONMENT, FILES, EXIT STATUS,
EXAMPLES, DIAGNOSTICS, ERRORS, SEE ALSO, STANDARDS, HISTORY, AUTHORS, CAVEATS,
BUGS, SECURITY CONSIDERATIONS.

NAME section must contain at least one `.Nm` and a `.Nd`.

## 8. Inline vs block (callable) macros

- **In-line** macros (`Em`, `Sy`, `Fl`, `Ar`, `Op`, `Ns`, `Mt`, `Vt`, `Fa`,
  `Fn`, `Pp`, `D1`, ...) act on the rest of their arguments, auto-close at
  the line end or at a following macro that starts new scope (`No`).
- **Block pairs** span lines: `Bd`/`Ed`, `Bl`/`El`, `Bk`/`Ek`, `Bf`/`Ef`,
  `Oo`/`Oc`, `Fo`/`Fc`, `Rs`/`Re` (bibliography: `%A` `%T` `%J` `%D` `%E`
  `%N` `%O` `%U`). The parser needs a nesting stack.
- The parser is a mix of "semantics + current section": SYNOPSIS changes how
  `Nm`/`Ft`/`Fn`/`Vt`/`In` render (new output line, header formatting).

## 9. Converter notes

- mdoc pages are far friendlier to MD conversion than man(7): the semantics
  map almost 1:1 to markdown emphasis/strong/code; that is what mandoc's
  `-T markdown` exploits.
- `Ds` is **not** a macro: the common `.Bl -tag -width Ds` uses it as a
  width token resolving to `6n`. Don't attempt to define a `Ds` macro.
- The BLOCK pairs need matching push/pop handling; unclosed macros (common in
  hand-written pages) must be tolerated: close all open scopes at EOF.
- `.Do/.Dq/.Qq` / `.Eo/.Ec` differ per argument-count (in-line style when one
  arg, block form when none and the text appears on later lines).
- Keep a section tally: markdown heading auto-level (H1 per `Sh`, H2 per `Ss`)
  makes `.Sh/.Ss` trivially map to `#`/`##`.

## References

- mdoc(7): <https://man.openbsd.org/mdoc.7>
- mandoc(1): <https://mandoc.bsd.lv/man/mandoc.1.html>
- mandoc Markdown writer reference: `mdoc_markdown.c` (ISC, reference only;
  not for code porting into an MIT module):
  <https://cvsweb.openbsd.org/src/usr.bin/mandoc/mdoc_markdown.c>
