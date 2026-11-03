# man(7) Macro Language

One-sentence summary: the full man(7) macro set — the **physical** (font/
indentation based) markup system most Unix man pages use — with syntax,
semantics, and the gotchas a converter hits in the wild.

> **Source of truth**: groff's `groff_man(7)`
> <https://man7.org/linux/man-pages/man7/man.7.html> and
> <https://man7.org/linux/man-pages/man7/man-pages.7.html>; BSD alternative
> <https://man.openbsd.org/man.7>. For raw roff mechanics see
> [roff-core.md](roff-core.md); for the semantic sibling format see
> [mdoc-macros.md](mdoc-macros.md).

## 1. What man(7) is

man(7) is a macro package loaded on top of roff (groff's `an.tmac`). It is
**physical markup**: `.B text` means "bold", not "this is a command";
semantics are implied only by convention (uppercase `.SH` section names,
`\-` in NAME, `.IP "\(bu"` bullets). Unlike mdoc, it knows nothing about
flags, paths or function signatures.

An old-style page is

```roff
.TH FOO 1 "2026-03-14" "foo 2.1" "User Commands"
.SH NAME
foo \- do something useful
.SH SYNOPSIS
.B foo
[\fB\-v\fP]
.IR file ...
.SH DESCRIPTION
.PP
Narrative description.
.SH OPTIONS
.TP
.B \-v
Verbose output.
.SH "SEE ALSO"
.BR bar (1),
.BR baz (8)
```

Only **one** `.TH` may occur, and it must be the first macro call in the file.


## 2. `.TH` title line

`.TH title section [date [source [manual]]]`

| Argument | Meaning |
|----------|---------|
| `title` | page name, conventionally uppercase (`LS`) |
| `section` | manual section: `1`..`8`, optionally suffixed (`1p`, `3p`, `3pm`, `3ssl`, `1m`) |
| `date` | date of last nontrivial change, prefer ISO `YYYY-MM-DD` |
| `source` | supplying project/package (groff: *footer-middle* & *footer-inside*) |
| `manual` | volume title for the centered header (groff: *header-middle*) |

For markdown, at minimum capture `title` and `section`; date/source can become
a metadata comment or a trailing line. Section numbers map to the standard
manual categories (1 user commands, 2 syscalls, 3 library, 4 devices, 5 file
formats, 6 games, 7 misc, 8 admin, 9 kernel).

Conventional section order (man-pages(7)): NAME, LIBRARY, SYNOPSIS,
CONFIGURATION, DESCRIPTION, OPTIONS, EXIT STATUS, RETURN VALUE, ERRORS,
ENVIRONMENT, FILES, ATTRIBUTES, VERSIONS, STANDARDS, HISTORY, NOTES, BUGS,
EXAMPLES, COPYRIGHT, SEE ALSO. Use these for auto-numbering headings, but
pages in the wild deviate freely; do not enforce.

## 3. Structure macros

| Macro | Syntax | Behavior / converter notes |
|-------|--------|---------------------------|
| `.TH` | as above | title line; header for the markdown doc |
| `.SH [text]` | `.SH NAME` or `.SH` (no args → next line is heading) | markdown H1; the quoted form `.SH "SEE ALSO"` may arrive as two args, so join arguments with spaces |
| `.SS [text]` | subsection, indented | markdown H3 |
| `.P` / `.PP` / `.LP` | synonyms | paragraph break (no args); clears latest `IP/TP/HP` indent but not `.RS` insets |
| `.TP [indent]` | tagged paragraph; **next input line is the tag**, following text is indented body | emit tag (typically bolded/italicized) followed by body; ideal rendering is `-tag` style: `**tag**` then body as a list or a definition-ish layout |
| `.TQ` | extra tag for the preceding `TP` with no vertical space (GNU, mandoc ≥1.14.5) | emit additional tag row |
| `.IP [mark [indent]]` | indented paragraph with optional mark | list items: `"\(bu"` or `"\(**"` or `"- "` marks → bullet list; unmarked `.IP 4` → indented text; this is how man pages build all lists, including ordered lists via convention |
| `.HP [indent]` | hanging paragraph | approximate with a list item or paragraph; body typically on the same line |
| `.RS [amount]` | relative inset start (nests) | list/blockquote nesting indicator |
| `.RE [level]` | relative inset end | `1`=close one level, no-arg same, `n` leaves n-1 open |
| `.EX` / `.EE` | begin/end example (groff, mandoc, Plan 9, Heirloom; not Solaris) | ≡ `.nf`/`.fi` + monospace: fenced/indented code block |
| `.PD [n]` | paragraph distance | ignore |
| `.DT` | reset tab stops | ignore |
| `.B`, `.I`, `.SM`, `.SB`, `.SH`-style no-arg forms | font/heading macros with no argument take the **next text line** as their text | the "next-line trap"; every such macro is a pending-tag state |


## 4. Font macros

| Macro | Meaning |
|-------|---------|
| `.B text` / `.B` | bold |
| `.I text` | italic |
| `.SM [text]` | one size smaller (next line if no arg) |
| `.SB [text]` | small bold (deprecated) |
| `.BI b i ...` | alternating bold then italic; arguments concatenated, spaces between args dropped |
| `.BR`, `.IB`, `.IR`, `.RB`, `.RI` | same idea with roman |
| `.SY k [suffix]` / `.YS` | synopsis block (GNU + mandoc + Heirloom only): bold keyword, continuation indents |
| `.OP option-name [option-argument]` | `[**name** arg]` bracketed optional option (deprecated quirk) |

String joining rule: `.BI`/`.BR`-style macros strip their inter-argument
whitespace in output; a quoted argument preserves spaces. Reproduce this
when re-emitting: `.BR foo (1)` becomes `**foo**(1)`.

Font escapes `\fB \fI \fR \fP \f(BI \f(CW` act inline; implement a font stack
in the writer (see markdown-mapping.md).

## 5. Hyperlinks (GNU extensions, mandoc supported)

| Macro | Render as |
|-------|-----------|
| `.MR topic [section [trailing]]` (groff 1.23+) | `topic(section)` — markdown link text `[`topic(section)`](man:topic(section))` or plain text |
| `.MT addr` ... `.ME [trailing]` | `[text](mailto:addr)` |
| `.UR uri` ... `.UE [trailing]` | `[text](uri)` |

If a converter does not support hyperlink output for a given scheme, render
`\fB<text>\fP <uri>` from the mandoc fallback (`UR` text in bold, target in
angle brackets). Cross references conventionally appear as
`.BR foo (1)`, `.B foo(1)`: recognize the `name(section)` pattern and either
leave it as text or generate a `man://` style link.

## 6. Deprecated / legacy macros

`.AT [system [release]]` and `.UC [version]` write the old AT&T/BSD footers —
parse, drop or capture into metadata. `.IX` (pod2man index macro) is for
indexing only; mandoc defines it to nothing and so should a converter —
**never leak `.IX` into output text**.

Pages in the wild also contain local extensions: Solaris and SGI pages ship
extra macros via `man.local`-style includes, and some generated pages invent
their own wrappers. Any statement about "the man(7) macro set" is approximate:
be tolerant to unknown macro names as documented in [roff-core.md](roff-core.md).

## 7. NAME section (machine-read)

`.SH NAME` must be the first section and contain exactly:

```roff
foo \- do something useful
```

- names may be comma separated (`foo, bar \- descriptions`)
- the separator **must** be `\-`, not `-` (plain `-` is breakable)
- no macros or fonts are allowed inside NAME; parsers/indexers (apropos,
  makewhatis, man-db) read it literally
- real pages sometimes contain extra text before the separator or stray fonts
  in NAME — treat these as anomalies and preserve the visible text.

Carapace integration note: `pkg/man/man.go`'s `man-to-md` route currently
formats a rendered page; a native converter should be **parse-level** and able
to also produce the NAME-derived summary, the same way the man(1)/apropos
indexers do.

## 8. Common converter failure modes (real-world pages)

1. **pod2man pages**: preamble defines `.Sp`, `.Vb`, `.Ve`, `.IX`; conditionals
   branch on `\n(.g` etc. — user-macro expansion is mandatory (see
   parser-design.md).
2. **`"\(bu"` bullets**: `.IP "\(bu"` / `.IP "\(bu 4"` are the classic bullet
   convention; also `"\(**"`. Map to markdown list items.
3. **`.IP 1`-style ordered items**: a numeric mark starts an ordered/definition
   list look-alike; convert only when the mark parses cleanly as `N.`, else
   keep it as plain indented text.
4. **`.B`-with-no-argument trap**: a blank line after `.B` will bold the
   following paragraph — the state machine needs the pending-text-line capture.
5. **Pages without `.SH` headings**: `.TH` present but content between
   title and first section; keep text flowing rather than failing.
6. **`.DS/.DE/.NH`** (ms macros) inside man pages: undefined in man(7); treat
   like `.EX` (`nf` + display) for `.DS/.DE` and drop or heading-ify `.NH`.
7. **`.so man.local` / `.so $srcdir$`** — include files; unless you resolve
   them, parse-and-ignore.

## References

- groff man(7): <https://man7.org/linux/man-pages/man7/man.7.html>
- groff_man style guide: <https://man7.org/linux/man-pages/man7/groff_man.7.html>
- man-pages(7) section order: <https://man7.org/linux/man-pages/man7/man-pages.7.html>
- OpenBSD man(7): <https://man.openbsd.org/man.7>
- mandoc(1) (diagnostics/validation): <https://mandoc.bsd.lv/man/mandoc.1.html>
