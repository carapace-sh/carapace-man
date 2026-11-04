# Markdown Output Mapping

One-sentence summary: the construct → markdown mapping table for both man(7)
and mdoc(7), plus the metacharacter escaping rules, modeled on mandoc's
`-T markdown` conventions (the de-facto reference behavior; man(7) input is
not supported by mandoc's markdown writer, so that part is a design
decision, not a copy of mandoc behavior).

> **Source of truth**: mandoc(1) and its observed `-T markdown` behavior
> <https://mandoc.bsd.lv/man/mandoc.1.html>; implementation notes for mdoc
> exist in mandoc's tree (`mdoc_markdown.c`). For our own converter, the
> mapping below is a design document for a from-scratch, MIT-licensed writer.

## 1. Headings and document structure

| roff construct | markdown |
|----------------|----------|
| `.TH NAME SEC ...` | document title `NAME(SEC)` as H1, metadata in comment or trailing line |
| man: `.SH X` | `## X` |
| man: `.SS X` | `### X` |
| `.Pp` / `.PP` / `.sp` | blank line (paragraph break) |
| `.br` | two trailing spaces + newline (hard line break) |
| mdoc `.Sh X` | `# X` |
| mdoc `.Ss X` | `## X` |
| `.ig`, comments, `.IX`, `.DT`, `.PD` | dropped entirely |

## 2. Inline states

Condensed table based on mandoc's mapping and on typical conversions:

| roff/mdoc | markdown |
|-----------|---------|
| `.B text`, `.BR`, font `\fB`, mdoc `.Sy`, `.Nm`, `.Fl x`, `.Cm`, `.Ic`, `.Ms` | `**text**` |
| `.I text`, font `\fI`, mdoc `.Ar`, `.Em`, `.Ad`, `.Pa`, `.Va`, `.Vt`, `.Ft`, `.Fa`, `.Sx` | `*text*` |
| roff `\f(CW`, `\f(I`, `.Dl`, `.Bd -literal`, nf blocks | code (backticks / fenced block for multi-line) |
| mdoc `.Dv`, `.Er`, `.Ev`, `.Li`, `.Tn`, `.Ql` | `` `text` `` |
| `.EX` ... `.EE`, `.nf` ... `.fi`, `.Vb` ... `.Ve` | fenced code block (```) or indented code; pick fenced blocks for reliability |
| template `.RS/.RE` insets | usually rendered as nested lists / blockquotes; a depth >3 grows noisy — fall back to plain paragraphs or `-tag` definition style |
| `.IP "\(bu"` rows | bullet list items (`- ...`) |
| `.IP "\(bu ... 4"` inside `.RS` blocks | nested bullet list (depth from the `.RS`/`.IP` nest level) |
| `.IP "1. something"` and other numeric/dot marks | markdown ordered list (parse the mark; if it does not parse as `N.`, emit as plain text) |
| `.TP` / `.TQ` tag/body pairs | `**tag**` followed by indented body (mandoc style), or `[tag] body` with a blockquoted body — keep consistent sitewide |
| `.HP text` | list item with an empty body-line approach / plain paragraph |
| `.SY` ... `.YS` | bold usage line then options list (`**cmd** [**-a**] ...`) |
| `.OP name [arg]` | `[**name** *arg*]` |
| `.MR topic [section]` | `[topic(section)](man:topic(section))` or plain text |
| `.MT addr`...`.ME` | `[*text*](mailto:addr)` |
| `.UR uri`...`.UE` | `[*text*](uri)` |
| `.Mt addr` | `&lt;addr@example.org&gt;`-style autolink (CommonMark treats bare addresses as mailto links; output the roff text unmodified if in doubt) |
| `.Lk uri [text]` | `[text](uri)` |
| `\-` | literal `-` escaped (not raw) when fragile |
| `\(em` / `\[em]` | emit a plain dash (`-`) or the em dash, per converter style decision |
| `<` and `>` in prose | `&lt;` / `&gt;` (avoids raw HTML) |

## 3. Metacharacter escaping

Markdown output requires care so that the emitted markdown does not swallow
your layout's meaning; bits that need escape treatment:

| char / context | escape |
|----------------|--------|
| `#`, `+`, `-` at line start | backslash: `\#` |
| `.` or `)` after a leading integer (`1.  text`) | backslash: `1\. text` |
| `(` after a leading `]` (CommonMark requires it) | escape `](` boundaries |
| `*`, `[`, `]`, `_`, backslash in prose | backslash-escape in prose |
| `` ` `` in prose | backslash-escape, or change delimiter to double-backticks and escape inside code spans |
| `<` / `>` | `&lt;` / `&gt;` (avoid raw HTML) |
| backslash `\` | when it arises from `\e` interpolation, emit `\\` |
| non-breaking space (`\~`) | a normal space is fine for a converter (or `&nbsp;` if faithfulness matters) |
| trailing space | strip, or use `&zwnj;` when a trailing-space would become a hard break |
| adjacent `\*` sequences | separate with `\*\\\*` or use `&zwnj;` so emphasis does not span the boundary |
| URL targets | percent-encode `%`, `(`, `)`, `<`, `>`, space |

Non-ASCII handling: mandoc's markdown writer is **ASCII-only** and emits
non-ASCII as decimal HTML entities (`&#233;`), except inside code spans/fenced
blocks where entity expansion fails — there it transliterates to the closest
ASCII approximation. Our converter may select one of: (a) ASCII-entity mode
(what mandoc does), (b) full UTF-8 output (recommended for sanity when
consumers are modern renderers), or (c) a per-glyph table. Decision: pick
UTF-8 output and transliterate only truly unrepresentable glyphs, since the
output feeds static docs, not terminals.

### Fence language labels

Fenced blocks carry a language label scored by `pkg/manpages/syntax.go::detectLanguageFor`: shebangs, interpreter prompts (`$`/`%`/`#`, `user@host#`), usage-synopsis lines (`prog [OPTION]...`), and language dialects (c, perl, go, sql, ini, json, html, diff, make, nginx, python) label the fence; a `console` fallback applies inside SYNOPSIS sections, and fences below the scoring threshold stay plain ``` blocks.

## 4. Structure fallbacks

More shapes that markdown represents poorly (mandoc's guidance, applied as
design rules):

- **`.Bl -column`** — markdown has no column layout that is not a table;
  mandoc falls back to an indented code block with padding. Alternatives:
  a markdown table, or a padded code block. Choose one and be consistent.
- **`.Bl -tag` with long heads** — no markdown primitive; mandoc uses tag
  + blockquote (`> body`) or a bullet list. For our converter: prefer
  `**head**` + body as a subsequent indented bullet list section (flagged
  as "definition list style"), as it renders decently in most renderers.
- **`-ohang`, `-inset`, `-diag`** — approximate with plain paragraphs and
  line breaks; never synthesise fake list markup you cannot justify.
- **`.Bl -bullet`/`-dash`/`-enum`** and man's `.IP "\(bu"` are the only
  constructs that are "true" markdown lists; do not over-convert structures
  that only look like lists.
- **Complex nesting of codes/quotes/lists** — lock your writer into a
  chosen tight binding (code inside blockquote, list inside code, etc.) via
  a stack of open scopes and a repeated-`>` prefix emitter or the indentation
  required for nested code blocks.

## 5. Samples (indicative, not normative)

Input (man(7)):

```roff
.TH FOO 1 "2026-03-14" "foo 2.1" "User Commands"
.SH NAME
foo \- do something useful
.SH SYNOPSIS
.B foo
[\fB\-v\fP]
.IR file ...
```

Possible output:

````markdown
# foo(1)

## NAME

foo - do something useful

## SYNOPSIS

**foo** [**-v**] *file* ...
````

Input (mdoc):

```roff
.Dd $Mdocdate$
.Dt FOO 1
.Os
.Sh NAME
.Nm foo
.Nd do something useful
.Sh SYNOPSIS
.Nm foo
.Op Fl v Ar file
```

Possible output (mdoc style, more semantic):

````markdown
# foo(1)

## NAME

`foo` - do something useful

## SYNOPSIS

**foo** [*-v* *file*]
````

## Related docs

- [parser-design.md](parser-design.md) for the state machine feeding the writer.
- [mdoc-macros.md](mdoc-macros.md) for the full semantic macro set.
- mandoc output conventions: <https://mandoc.bsd.lv/man/mandoc.1.html>
