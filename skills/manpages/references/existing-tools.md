# Existing Tools, Licenses, and Reuse Guidance

One-sentence summary: the survey of existing manpage parsers and
converters with their licenses, plus the rules for what may be studied,
referenced, or incorporated while keeping the new converter **MIT licensed**.

> Background on our license scope: **MIT** for the converter. Code that is
> GPL/copyleft must never be vendored, copied, translated, or "ported" into
> the module. Vendoring is fine only when the license permits redistribution
> **and** attribution details are carried, and only when the maintainer
> accepts the hybrid file setup (a GPL file cannot live inside an MIT module).

## 1. The excluded vendor in this repo

`third_party/github.com/mle86/man-to-md/` **was** GPL-3.0 Perl — a macro-oriented
`man(7)` reading engine (~69★, last release 2020, v0.17.0, author mle86) used
via `go:embed` of `man-to-md.pl`. The native `pkg/manpages` converter replaced
it and the vendored tree has been deleted from the repo. It was:

- **man(7) only** (no mdoc support);
- GPL-3: **never port, translate, or vendor into the MIT converter**;
- The converter described by this skill is its replacement; no translated
  logic, no matching identifiers, and no copied macro-table ordering from it
  may appear in the new module.

## 2. License survey (man/roff → markdown)

| Tool | URL | License | Language | Approach / notes |
|------|-----|---------|----------|------------------|
| **mandoc (mdocml)** | <https://mandoc.bsd.lv/> | **ISC** | C | The gold standard validating compiler/parser for man/mdoc + built-in tbl/eqn; `-T html`/`-T utf8`/`-T markdown` (only for **mdoc(7)** input). Extractable behavior specs: parse & renderer conventions. Study behavior freely; a from-scratch implementation based on the documented spec avoids even the ISC notice question. |
| **man2md (phillbush)** | <https://github.com/phillbush/man2md> | **Public domain** | POSIX awk | The best "start from scratch" reference for a minimal macro-to-markdown filter: awk source can be consulted freely. man(7) only. |
| **man2md (rudojaksa)** | <https://github.com/rudojaksa/man2md> | **GPL-3.0** | Perl | Works on *rendered* man output, not roff source — smaller useful overlap. |
| **roff-cli / man_parser** | <https://github.com/ljh-sh/roff> | **roff-cli**: Apache-2.0; **man_parser crate**: MIT | Rust | Macro-oriented parser for mdoc and man; outputs JSON, markdown, HTML. New (2026). MIT crate parts are inherently usable; Apache-2.0 parts require the NOTICE chain. |
| **ManT (libmandoc-rs)** | <https://github.com/BryanHeBY/ManT> | **Apache-2.0** (bundles ISC libmandoc) | Rust | Full parse of mdoc via vendored libmandoc (ISC), outputs markdown / JSON. Useful as a behavioral cross-reference; an Apache+ISC license audit is required before any reuse. |
| **pandoc** | <https://github.com/jgm/pandoc> | **GPL-2.0-or-later** | Haskell | Real `man` and `mdoc` **readers**; good external cross-check for behavior; **never ported**. |
| **groff** | <https://www.gnu.org/software/groff/> | **GPL-3.0-or-later** | C++ | The native typesetting engine; the upstream implementation of the man(7) macros (`an.tmac`, `an-ext.tmac`) — readable only as documentation of behavior, never as a source of code. |
| **Heirloom doctools** | <https://github.com/n-t-roff/heirloom-doctools> | **CDDL-1.1** | C | Historical troff; no markdown. |
| **unroff** | <https://github.com/rockola/unroff> (mirror) | permissive Oliver Laumann (BSD-like, attributed) | C+Scheme | Historical (1995) full troff->HTML; doc-quality reference. |
| **troffcvt** | <http://www.snake.net/software/troffcvt/> (Debian salsa mirror) | **Public domain** | C | Legacy troff-to-HTML via tc2html and companion filters. |
| **man2html (man-1.6g, Lucifredi)** | (Debian src) | **GPL-2.0+** | C | The classic man-db HTML renderer; **excluded**. |
| **Carapace-family man-to-md (mle86)** | see §1 | **GPL-3.0** | Perl | **Excluded and now deleted.** |
| **sio/man** | <https://github.com/sio/man> | **Apache-2.0** | Go | Client for manpages.debian.org; uses groff — not a parser. |
| **debiman** | <https://github.com/Debian/debiman> | **Apache-2.0** | Go | manpages.debian.org builder; delegates parsing to **mandoc**. Useful as an architecture reference for "Go orchestration + mandoc subprocess", not as a parser. |

## 3. Markdown → manpage (opposite direction; license-friendly prior art)

| Tool | URL | License | Language | Why interesting |
|------|-----|---------|----------|-----------------|
| **go-md2man** | <https://github.com/cpuguy83/go-md2man> | **MIT** | Go | Mature blackfriday-based md→roff; a solid indication of the roff output habits in Go ecosystems (but not a parser). |
| **kramdown-man** | <https://github.com/postmodern/kramdown-man> | **MIT** | Ruby | md→roff. |
| **md2man (sunaku)** | <https://github.com/sunaku/md2man> | **MIT** | Ruby | md→man/HTML. |
| **md2roff (davidnewhall)** | <https://github.com/davidnewhall/md2roff> | **MIT** | Go | md→man. |
| **mandown (markhuge)** | <https://github.com/markhuge/mandown> | **GPL-3.0** | Go | md→manpage; **excluded**. |

## 4. Rules of engagement (clean-room)

1. **Readable-as-spec material (safe)**: the documentation considered
   freely redistributable and explanatory: `roff(7)`, `man(7)`/`groff_man(7)`,
   `mdoc(7)`, `mandoc(1)`'s `-T markdown` behavior notes, `man-pages(7)`,
   `groff_char(7)`. These man pages describe what the macros *mean*; use
   them as the specification source for every macro decision.
   (<https://mandoc.bsd.lv/man/>, <https://man7.org/linux/man-pages/>,
   <https://www.gnu.org/software/groff/manual/>).
2. **Public-domain material (copyable, no attribution duty)**: `man2md`
   (phillbush, awk), `troffcvt`. Note: "public domain" is asserted by the
   project, not legally guaranteed; risk is near zero, but keep the copy
   provenance documented in the repo.
3. **MIT material (copyable with notice)**: man_parser crate (Rust),
   go-md2man, kramdown-man, md2man, md2roff. Ruby/Go + MIT portions may be
   vendored with license text preserved; carry the notice into a
   `third_party/<project>` folder with the original LICENSE.md and a NOTICE
   package line.
4. **ISC (mandoc)**: ISC is a permissive license and derivative work is
   allowed with the copyright/license notice preserved; a hand translation
   of mandoc's C into Go would therefore be redistributable but must ship
   the ISC notice. The chosen plan is stricter and cleaner: rely on
   mandoc's **documented** behavior only and write our own implementation,
   so no notice obligations arise at all.
5. **Apache-2.0 material (ManT, debiman, sio/man)**: reuse is legally
   possible with NOTICE persistence, but Apache-2.0 and MIT combine
   awkwardly inside one Go module (patent grant, NOTICE files, attribution
   lines). Do not vendor; read only as background design material.
6. **GPL-licensed material (groff, mle86/man-to-md, pandoc, man2html,
   markhuge/mandown, rudojaksa/man2md)**: **excluded**. Not vendored, not
   ported, not translated, not copied with renamed identifiers.

## 5. Verification tools (external cross-checks only)

Never vendor these; use them as CLI cross-check oracle during testing:

```
mandoc -T markdown mdoc-page.7        # compare against our mdoc converter output
pandoc -f man / -f mdoc -t markdown   # generalized, loss-prone, cross-check only
groff -man -Tutf8                     # authoritative rendering for man(7) visual fidelity
```

Real-world test corpus sources (man page content is freely redistributable):

- Debian manpages corpus (package `manpages`) — the huge real-world set.
- OpenBSD `/usr/share/man` snapshot — mdoc style exemplars.
- Git's own `Documentation/` manpage exports, X11 man docs.
- **`github.com/carapace-sh/man`** — the corpus carapace-man `sync`
  actually targets; the converter's honest acceptance suite.
- Perl documentation (`perlop`, `pod2man`-generated pages) — the hardest
  stress tests: user-defined macros, conditionals, font escapes.

## Related docs

- [roff-core.md](roff-core.md), [parser-design.md](parser-design.md) for why
  these tool categories matter to the architecture.
