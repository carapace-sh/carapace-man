# Plan: Native Go Manpage-to-Markdown Parser

## Motivation

`man://` lookups currently shell out to the system `man` binary and pipe the
gzipped roff page through the vendored GPL-3 Perl script
(`third_party/github.com/mle86/man-to-md/`, wired via
`pkg/man/manpage.go`). That path has three problems:

1. **License** — GPL-3 code we must not port, paired with GPL groff output;
   the converter we actually want is MIT (see
   `skills/manpages/references/existing-tools.md`).
2. **Runtime deps** — `man`/`perl` on `$PATH`; breaks in
   containers, on Windows, and in anything bundling carapace-man as a
   library.
3. **No cross-references** — the Perl flattens all cross-references to plain
   text. Ideal: highlighted candidates in carapace-bin/tabdance get
   documentation containing carapace-style UIDs as links
   (`man://git-commit/1`), so every rendered page is itself navigable.

A from-scratch MIT Go parser replaces all of it:
- `man://` resolves without `man`/`perl` (still resolves `man` correctly
  when present; offline capable with an archived corpus, see Follow-up).
- Cross-references become carapace UIDs in the markdown output.
- Documentation feeds and tests for an mdbook conversion.

Design inputs: `skills/manpages/references/parser-design.md`,
`markdown-mapping.md`, `man-macros.md`, `mdoc-macros.md`, `roff-core.md`.
Clean-room rule from those docs applies: no code ported from groff/mandoc/
man-to-md — spec-driven plus mandoc's observed `-T markdown` output shape.

---

## Scope

| Inputs | In scope | Out of scope |
|--------|---------|--------------|
| man(7) sources (`.TH/.SH/.TP/.IP/.B/.I/.SY/...`) | yes | full typesetting (spacing, hyphenation, over pages) |
| mdoc(7) sources (`.Dd/.Dt/.Sh/.Bl/.Fl/...`) | yes | tbl/eqn/pic preprocessing (emit block as plain text) |
| GNU extensions used in the wild (`.de/.am/.ig/.if/.tr/...`) | yes | PDF/PS, terminals, pagers |

Preprocessors: `tbl`, `eqn`, `pic` are **detected and skipped** (emit the
region as a neutral code block), not implemented. Targeted at the common
corpora: Linux distribution man-pages, OpenBSD, X11, and pod2man output.

---

## Package layout

```
pkg/manpages/
  reader.go        # logical-line reader: backslash-newline joining, CR/LF normalize, bounded lookahead
  tokenize.go      # control-line detection, request/argument tokenization (quoted args)
  expand.go        # \$n / \*(name) / \n[name] interpolation, .ds/.nr, .de/.am with depth caps, .ig, .if/.ie/.el
  dialect.go       # lazy man(7) vs mdoc(7) detection via first .TH / .Dd/.Dt macro
  man7.go          # man(7) macro table (see man-macros.md)
  mdoc7.go         # mdoc(7) macro table (see mdoc-macros.md)
  state.go         # fill mode, font stack, indent/list state, pending next-line trap, scope stack for mdoc
  writer.go        # streaming markdown writer: paragraph/list/code-fence block decision, escaping
  xref.go          # cross-reference extraction + carapace UID emission
  xref_test.go     # cross-reference tests
  man7_test.go     # golden tests man(7)
  mdoc7_test.go    # golden tests mdoc(7)
  corpus_test.go   # property test over the archive corpus
```

Architecture from parser-design.md: a **streaming writer**, no AST.
Reader → roff expansion → dialect dispatch → macro/state layer → writer.

## CLI / API surface

```bash
carapace-man man-to-md <command>                  # default: native parser; falls back to --help as today
carapace-man man-to-md <command> --section 1      # unchanged interface
carapace-man man-to-md <command> --legacy         # escape hatch to the old Perl path (kept for one release cycle)
carapace-man man-to-md <file.roff>                # file path instead of page name — used by the mdbook conversion
```

`pkg/man.ManToMd` swaps the guts to call `manpages.Parse(io.Reader)`.
Same markdown output contract as today (no style/width handling here —
glamour `Style()` at presentation still does that).

---

## Cross-references as carapace UIDs

Emit cross-references as carapace UIDs in markdown. The output
consumption context is carapace-bin/tabdance, where a highlighted candidate
shows the rendered doc; `man://` UIDs are clickable links to other pages.

| Input construct | Source | Output |
|-----------------|--------|--------|
| `.MR topic section` | mdoc(7) | `[**topic** (*section*)](man://topic/section)` |
| `topic(section)` prose in SEE ALSO (and, after confirming a visible NAME pattern, `word(...)`-shaped refs in other short cross-ref sections) | man(7) — the common format in practice, since man(7) has no semantic cross-ref macro | `[topic(section)](man://topic/section)` |
| `.UR`/`.UE`, `.MT`/`.ME`, `.Lk` | URL/mail links | keep as `[text](url)` links (do **not** UID-fy URLs) |

Linkify rules for `topic(section)`:
- Primarily within SEE ALSO; text before the first explicit section (e.g. in
  a doc's own NAME line, or secondary cross-ref sections) may be scanned
  with the same rules — never across the whole page, where option strings
  and version numbers can accidentally match.
- Only with valid section tokens: single digit (`1`–`9`), `0` (shell/POSIX),
  `n`, `l`, optionally suffixed by `p`/`3p`/`2p`. Anything else stays text.
  No valid section → not a manpage → leave as text.
- Multiple refs in one line, separated by commas/semicolons: link each.
- Referring to the page's own NAME: still link (harmless, keeps rules
  uniform).
- Emphasis already present around a ref: respect it; wrap the visible text
  as-is, the target is always the UID, never hyphenated or re-cased.

`man://` resolution (writing-scheme rules carried over from
`pkg/man/man.go`): `man://<topic>/<section>` — same `Describe()` that
today resolves `man://` shells out through the new parser (once archived
files exist, the DB lookup hits first — see Follow-up).

---

## Milestones

1. **M1 — reader + tokenizer + roff expansion layer**
   Logical lines, control/text classification, `.ds/.nr/.de/.am/.ig/.tr`,
   `.if/.ie/.el` boolean evaluator, interpolation with depth caps.
   Unit-testable on request-level fixtures (no markdown yet).
2. **M2 — dialect detection + man(7) macro layer + writer v1**
   A synthetic page exercising every man(7) macro in `man-macros.md` plus a
   handful of real pages (installed man corpus, git man pages) as golden
   tests. Writer conventions locked in once: fenced code blocks, list
   discipline (same marker → same list, new marker → new list), and the
   `.TP`/`.TQ` tag/body style — no flip-flopping between pages.
3. **M3 — mdoc(7) layer**
   Full semantic macro table, `.Bl` types (tag/bullet/dash/enum/column,
   `-compact`, `-width`, `-offset`), `.Bd/-literal`, `.Dl`, synopsis special
   casing, scope close-on-EOF.
4. **M4 — cross-references + escape-hardened output**
   `xref.go` (man:// UID emission per table above), metacharacter escaping
   from markdown-mapping.md, UTF-8 output decision (pick UTF-8, per the
   mapping doc), hardening items 1–10 of parser-design.md (undefined macros
   as no-op, unclosed scopes, `.ig` mis-termination, recursion caps, `\&.`
   handling, etc.).
5. **M5 — man-to-md switchover**
   Default path becomes the native parser; `--legacy` keeps the Perl path.
   `man` binary only used in `--legacy`; the native path locates the page
   via `man --location`, then gunzips the roff file
   in-process (`compress/gzip`).
6. **M6 — remove the vendored GPL path**
   Delete `third_party/github.com/mle86/man-to-md/` and the `--legacy`
   escape hatch once M5 has run a full release cycle in real usage.

Corpus for tests (from parser-design.md): Debian man-pages, OpenBSD
`/usr/share/man`, X11 docs, the roff pages of already-as-adoc doc suites
(git), and pod2man output from a Perl install (worst case), plus whatever
the archive used for the mdbook conversion contains. Property test per
corpus: exit 0, no panic, no raw roff directives left in output; track
"raw directive %" as a regression metric. Golden tests for named pages;
`mandoc -T markdown` and `pandoc -f man` only as external cross-checks
(never vendored/referenced behavior in code).

---

## Follow-up: archived manpages → mdbook (also the verification harness)

After M6, a **man page archive** (roff sources collected locally, e.g.
extracted from the kernel.org `man-pages-<version>.tar.xz` bundle or
distro package sets, plus pages collected from `man://` lookups) converts
to an mdbook — a static site of all pages — which doubles as the
verification test for the parser. Fetching the archive stays outside the
tool (documented `wget`/`tar` one-liner); a CI workflow on master that
regenerates gh-pages from the book follows later, in the style of other
carapace projects.

```bash
carapace-man mdbook <archive-dir> <book-dir>
```

- Walks the archive, parses every roff file with the new parser, writes
  mdbook layout: `book.toml`, `src/*.md`, one md file per page named
  `<page>(<section>).md`, and an alphabetized `SUMMARY.md` grouped by
  section.
- **Cross-references become relative book links**: the parser emits link
  *metadata* (target + section) rather than final URLs; the default
  consumer stamps `man://` UIDs and the mdbook writer rewires the same
  metadata to relative `../<page>(<section>).md` links. No parser change
  needed for the book — only a different link emitter at write time.
- **Verification property (the test)**: processing the whole archive must
  be crash-free for the parser; every cross-link *resolved* in the archive
  (a link whose target section/title does not correspond to an archived
  page is listed as missing in a report) — and the output builds via
  `mdbook build` in CI with zero linking errors. Coverage keeps growing as
  new pages land in the archive; this is a whole-corpus regression net that
  golden files alone can't give.
- Output target (gitignored, CI artifact — not committed): rendered HTML
  book, add a small linking step later if a hosted book is desired.
- The corpus is per-page licensed (BSD/MIT/GPL/copyleft mix); never
  commit archive contents or the generated book to this repository.

### Potential follow-up: chroma-based fence language detection

The current fence language labels come from the hand-rolled heuristic in
`pkg/manpages/syntax.go` (see `skills/manpages/references/markdown-mapping.md`).
A better-calibrated option for the *book build tool only* is
`github.com/alecthomas/chroma`'s content analysis (MIT, go-native;
research and API semantics in
`skills/manpages/references/chroma-analysis.md`):

- `manpages.Parse` gains an optional `LanguageDetector func([]string) string`
  hook that defaults to the internal heuristic; `pkg/mdbook` wires
  `lexers.Analyse` into it (sample a page prefix, map the picked
  lexer's `Config.Name`/aliases to a fence identifier, keep `console`
  and plain-fence fallbacks).
- Only users of the mdbook build pay the dependency and binary-size
  cost (~few MB from the embedded lexer tables plus `regexp2/v2`);
  `man-to-md`, `inspect` and library consumers of `Parse` keep the
  zero-dependency heuristic output.
- Non-goals: porting chroma analyser rules by hand (option C), changing
  the `man://`/unlabeled-fence contract for terminal or UID contexts.
