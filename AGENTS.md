# AGENTS.md

Guide for AI agents working in the `carapace-man` repository.

## Project Overview

`carapace-man` is a Go CLI tool in the [carapace](https://github.com/carapace-sh) shell-completion ecosystem. It renders command documentation (man pages, carapace-spec command docs) as styled terminal output, and provides tooling to manage the `man/cmd/` YAML documentation specs consumed by carapace-bin completers.

Module: `github.com/carapace-sh/carapace-man` · Go 1.25.8

## Essential Commands

```bash
go build ./...                    # build all packages
go build -o carapace-man ./cmd/carapace-man   # build the binary
go vet ./...                      # static analysis (passes clean)
go test ./...                     # run tests (no tests currently exist)
```

There is no Makefile, no CI workflow beyond dependabot auto-merge, and no lint configuration. `go vet` is the only static check wired up.

### Running the tool

```bash
carapace-man <uid>                          # render docs for a UID
carapace-man <uid> --raw                    # raw markdown (no styling)
carapace-man <uid> --style carapace         # custom style (default)
carapace-man <uid> --style dark --wrap 120  # glamour style + width

carapace-man sync                           # clone/pull https://github.com/carapace-sh/man
carapace-man sync --cmd-source /path/to/man/cmd   # also copy cmd/ YAML docs

carapace-man update <spec> <output-dir>     # one-step: diff + split-to (recommended)
carapace-man update - man/cmd/<completer>   # read spec from stdin
carapace-man update --dry-run <spec> <dir>  # preview changes without writing

carapace-man split-to <spec> <output-dir>   # split spec, preserve existing docs
carapace-man spec-diff <spec> <existing-dir>   # report drift vs committed docs
carapace-man man-to-md <command>            # convert system man page to markdown
carapace-man man-to-md <command> --section 1

carapace-man inspect <db>                   # list all UIDs in a compiled .db
carapace-man inspect <db> <uid>             # render docs for a UID from a .db
carapace-man inspect <db> <uid> --raw       # raw output (no styling)
```

Typical doc-update workflow (from the `man-docs` skill / `update.go` long help):

```bash
carapace <completer> spec | carapace-man update - man/cmd/<completer>
```

## Architecture

### UID scheme (central concept)

Everything is addressed by a URL-based UID parsed in `pkg/man/man.go:Describe`:

| Scheme | Meaning | Handler |
|--------|---------|---------|
| `cmd://<host>/<path>?flag=<name>` | carapace-spec command doc | `cmd()` in `command.go` |
| `man://<host>/<section>` | system man page | `manpage()` in `manpage.go` |
| `<other>://<host>/<path>` | generic YAML map descriptions | `descibe()` in `man.go` |

- `cmd` first checks user specs at `$XDG_CONFIG_HOME/carapace/specs/<host>.yaml`, then falls back to the synced repo at `$XDG_CONFIG_HOME/carapace/man/cmd/<host>/<host><path-with-dots>.yaml`.
- The `?flag=` query selects `documentation.flag[<name>]` instead of `documentation.command`.
- Generic schemes use a YAML `map[string]string` where values starting with `./` are treated as file paths relative to the scheme/host dir.

### Data location

All synced content lives under `$XDG_CONFIG_HOME/carapace/man/` (see `pkg/man/man.go:Location`). The `sync` command shallow-clones `https://github.com/carapace-sh/man` into that directory.

### Code layout

```
cmd/carapace-man/
  main.go                      # entrypoint → cmd.Execute()
  cmd/
    root.go                    # root cobra command (render UID), flag/style/wrap + completion wiring
    sync.go                    # clone/pull man repo + optional cmd/ sync
    update.go                  # composite: spec-diff + split-to, stdin support
    split.go / splitto.go      # split spec into per-subcommand files
    specdiff.go                # diff fresh spec vs existing cmd/ docs
    mantomd.go                 # system man page → markdown (falls back to --help)
    inspect.go                 # inspect compiled .db: list UIDs or render a UID's docs
    compile.go                 # compile specs/static docs/man-dir into bbolt .db
    util/
      split.go / splitto.go    # actual split logic (split.go = legacy, writes to tmpdir)
      specdiff.go              # DiffResult collection + PrintDiff
pkg/man/
  man.go                       # Location, Schemes, Hosts, Uids, Describe, Style, descibe
  command.go                   # cmd scheme handler (user specs + repo specs)
  manpage.go                   # man scheme handler (shells out to `man` + perl man-to-md)
  git.go                       # repo struct, clone/pull with functional options
  style.go                     # custom "carapace" glamour style, registered in init()
  db.go                        # bbolt lookup (cmdDB, describeDB), host index, inspect queries
  db_compile.go                # bbolt compilation (CompileSpec, CompileStatic, CompileManDir)
pkg/actions/man/
  man.go                       # carapace completion actions (ActionUids, ActionInspectUids)
third_party/github.com/mle86/man-to-md/
  mantomd.go                   # go:embed of man-to-md.pl
  man-to-md.pl                 # vendored Perl script (GPL-3), requires `perl` + `man`
```

### Control flow

1. `main.go` → `cmd.Execute()` → cobra dispatches to subcommand.
2. Root command: parse UID → `man.Describe(uid, glamour opts...)` → dispatch by scheme → optionally `Style()` with glamour renderer → print.
3. `update`: validate spec → `util.SpecDiff` (print) → `util.SplitTo` (write files, merging existing `documentation:`).
4. `man-to-md`: `man --location <section> <name>` → open the gzipped roff file → filter from `.TH` → pipe through embedded Perl script → output markdown. Falls back to `<command> --help` on failure.
5. `inspect`: open .db read-only → without uid: enumerate all scheme/host/uid pairs and print sorted list. With uid: `man.InspectDescribe` → optionally `Style()` → print. Supports `--style`/`--raw`/`--wrap` flags like the root command.

## Conventions & Gotchas

### go.mod replace directive

`go.mod` replaces `github.com/spf13/pflag` with `github.com/carapace-sh/carapace-pflag` (a carapace fork). The replace directive is required by carapace ecosystem dependencies (carapace, carapace-spec) — do not remove it or switch to upstream pflag.

### YAML spec schema header

Split/enriched files are written with a schema header for yaml-language-server:

```
# yaml-language-server: $schema=https://carapace.sh/schemas/command.json
```

`split-to`'s `save()` prepend this; the legacy `split()` function does **not**.

### Documentation preservation on split

`split-to` (and thus `update`) **overwrites structural fields** (name, description, flags, subcommands) but **merges existing `documentation:`** entries via `mergeDocumentation` in `util/splitto.go`. Only missing keys are filled from existing files; existing AI/human docs are never clobbered by empty fresh values.

### Carapace completion integration

Every command wires completions via `carapace.Gen(<cmd>)...` in `init()`. The root command additionally:
- Sets a fake hidden help command (`_carapace_help`) to suppress cobra's default help command.
- Calls `spec.AddMacro("Uids", ...)` and `spec.Register(rootCmd)` to register macros and spec support.

### Custom glamour style

`pkg/man/style.go` registers a `"carapace"` style into `styles.DefaultStyles` via `init()`. The root command's `--style` flag defaults to `"carapace"` and accepts any glamour style name, the literal `"carapace"`, or a path to a `.json` style file.

### `split` vs `split-to` (two different commands)

- `split` (`util/split.go`): legacy, writes to `os.TempDir()/carapace-man/<name>/`, uses `println` for filenames, does **not** add the schema header, does **not** preserve existing docs.
- `split-to` (`util/splitto.go`): current, writes to the user-specified directory, adds schema header, **preserves existing documentation**. This is what `update` uses.

Prefer `split-to` / `update` for any real work.

### External runtime dependencies

- `man-to-md` and the `man://` UID scheme require `man` and `perl` on `$PATH`.
- `sync` requires `git`.

### No tests

There are no `*_test.go` files anywhere. `go test ./...` reports `[no test files]` for every package. If adding tests, you'll be establishing the pattern from scratch.

### Typo in source

`pkg/man/man.go:115` defines `func descibe(...)` (missing the 'r' in "describe") — it's called from `Describe` at line 99. Do not "fix" this without updating the call site.

### `gopls` diagnostics (pre-existing, non-blocking)

- `pkg/man/man.go:136` — `HasPrefix`+`TrimPrefix` can be `CutPrefix` (hint).
- `pkg/man/manpage.go:57` — `bufio.Scanner` loop missing `scanner.Err()` check (warn).
- `pkg/man/manpage.go:61` — inefficient string concat in `WriteString` (warn).
These are pre-existing and not caused by your changes unless you touch those lines.

## Naming & Style Patterns

- Go package names match the directory leaf (`man`, `util`).
- Cobra commands are package-level `var <name>Cmd = &cobra.Command{...}` with an `init()` that calls `rootCmd.AddCommand(...)` and wires completions.
- Functional options pattern for `repo` (`WithLocation`, `WithProgress`) in `pkg/man/git.go`.
- Pointer helpers `boolPtr`/`stringPtr`/`uintPtr` in `style.go` for glamour style config.
- The codebase uses `println` (builtin) for debug-ish filename logging in legacy `split`; new code uses `fmt.Println`.
