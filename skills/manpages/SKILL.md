---
name: manpages
description: >
  Use when building, extending, or debugging the manpage-to-markdown converter
  in Go (MIT licensed, implemented from scratch) — roff/troff language core,
  man(7) and mdoc(7) macro languages, parsing strategy, roff-to-markdown
  output mapping, and the survey of existing converters with license
  constraints. Triggers on: "manpage", "man page", "roff", "troff", "nroff",
  "groff", "mandoc", "man(7)", "mdoc(7)", ".TH", ".SH", ".SS", ".TP", ".IP",
  ".PP", ".nf", ".fi", ".de", ".ds", ".nr", ".ig", "NAME section",
  "font macro", "user-defined macro", "pod2man", "man-to-md", "markdown
  conversion", "tbl", "eqn", "preprocessor", "escape sequence", "number
  register".
user-invocable: true
---

# Manpages In-Depth Reference

Reference for building a manpage-to-markdown converter in Go from scratch, MIT
licensed. Covers the roff language, the man(7) and mdoc(7) macro packages, a
parser architecture strategy, the construct-to-markdown mapping, and the
license landscape of existing converters.

## Licensing Constraint (critical)

The converter must be **MIT licensed**. Never port or copy code from GPL-licensed
tools (groff, mle86/man-to-md, pandoc, man2html). The GPL-3 Perl script that used to be vendored at
`third_party/github.com/mle86/man-to-md/` was never ported and has been
**removed** from the repo (the native `pkg/manpages` converter replaced
it). See [references/existing-tools.md](references/existing-tools.md) for the full
license survey and clean-room guidance.

## Sub-Resources

Load the reference that matches your task. When in doubt, load multiple references.

| Keywords | Reference |
|----------|----------|
| roff, troff, nroff, requests, control line, escapes, `\f`, `\&`, `\-`, `\(xx`, number register, string, `.ds`, `.nr`, `.de`, `.am`, `.ig`, `.if`, `.ie`, `.el`, `.so`, `.nf`, `.fi`, `.sp`, `.br`, `.in`, `.ti`, `.tr`, tbl, eqn, pic, grap, refer, preprocessor, copy mode, conditional grouping, nroff mode | [references/roff-core.md](references/roff-core.md) |
| man(7), man macros, `.TH`, `.SH`, `.SS`, `.TP`, `.TQ`, `.IP`, `.HP`, `.PP`, `.P`, `.LP`, `.RS`, `.RE`, `.B`, `.I`, `.SB`, `.SM`, `.BI`, `.BR`, `.IB`, `.IR`, `.RB`, `.RI`, `.SY`, `.YS`, `.OP`, `.EX`, `.EE`, `.MR`, `.MT`, `.ME`, `.UR`, `.UE`, `.PD`, `.AT`, `.UC`, `.IX`, NAME section, section order, man page anatomy, tagged paragraph, hanging paragraph | [references/man-macros.md](references/man-macros.md) |
| mdoc(7), `.Dd`, `.Dt`, `.Os`, `.Sh`, `.Ss`, `.Pp`, `.Nm`, `.Nd`, `.Fl`, `.Ar`, `.Cm`, `.Ic`, `.Op`, `.Pa`, `.Ev`, `.Dv`, `.Er`, `.Fa`, `.Ft`, `.Fn`, `.Fo`, `.Fc`, `.Vt`, `.Va`, `.Mt`, `.Lk`, `.Xr`, `.Sx`, semantic macros, `.Bl`, `.It`, `.El`, `-tag`, `-bullet`, `-enum`, `-column`, `-compact`, `-width`, `-offset`, `.Bd`, `.Ed`, `.Dl`, `.D1`, enclosure macros, `Ds` width | [references/mdoc-macros.md](references/mdoc-macros.md) |
| parser design, Go implementation, architecture, tokenizer, state machine, dialect detection, user macro expansion, conditional evaluation, register defaults, fill mode tracking, next-line trap, font stack, argument parsing, quoted arguments, line continuation, clean-room | [references/parser-design.md](references/parser-design.md) |
| markdown output, mapping table, bold/italic/code, headings, blockquote, list mapping, escaping, `**`, `*`, backtick, hard line break, non-ASCII, HTML entity, mandoc -T markdown, mdoc_markdown, markup loss | [references/markdown-mapping.md](references/markdown-mapping.md) |
| existing tools, mandoc, pandoc, man-to-md, man2md, roff-cli, ManT, license comparison, ISC, GPL-3, Apache-2.0, public domain, go-md2man, debiman, clean-room guidance, what can be reused, pod2man output | [references/existing-tools.md](references/existing-tools.md) |

## Quick Guide

- **How does roff syntax work (control lines, requests, escapes, registers)?** → [references/roff-core.md](references/roff-core.md)
- **What does macro `.X` in a man(7) page do?** → [references/man-macros.md](references/man-macros.md)
- **What does macro `.Xx` in an mdoc(7) page do?** → [references/mdoc-macros.md](references/mdoc-macros.md)
- **How is a man page structured (`.TH` args, NAME section, section order)?** → [references/man-macros.md](references/man-macros.md)
- **How do I structure the Go parser / which requests must it support?** → [references/parser-design.md](references/parser-design.md)
- **How do I render a roff construct as markdown and escape metacharacters?** → [references/markdown-mapping.md](references/markdown-mapping.md)
- **Can I port code from tool X under our MIT license?** → [references/existing-tools.md](references/existing-tools.md)
- **Why do real-world pages (pod2man, Perl pages) break simple parsers?** → [references/parser-design.md](references/parser-design.md)

## Data Flow

```
man/roff source
  -> line reader (control char detection, comments, line continuation)
    -> escape/string/register/argument expansion (roff layer)
      -> dialect dispatch (man(7) vs mdoc(7) via first macro)
        -> macro/state processing (fill mode, fonts, indents, next-line traps)
          -> markdown writer (mapping + metacharacter escaping)
            -> markdown output
```

## Scope

- **In scope**: roff language semantics needed for conversion, man(7) and
  mdoc(7) macros, parser architecture in Go, markdown mapping, license
  constraints on existing implementations.
- **Out of scope**: full typesetting (vertical spacing, page geometry,
  hyphenation, adjustment), output-only concerns of terminals/pagers,
  PDF/PS output, shell completion internals (see the **carapace** skill).

## Cross-Project References

- For man-page verification and rendering in the carapace-man repo itself,
  the current implementation shells out to `man` and the GPL Perl
  `man-to-md`; see [references/existing-tools.md](references/existing-tools.md)
  for why a native MIT converter replaces that path.
- **carapace-spec-man** (MIT, Go, https://github.com/carapace-sh/carapace-spec-man)
  already parses man pages into carapace specs; reuse its observations about
  real-world man page inconsistency, but note its output domain (specs, not
  markdown).
