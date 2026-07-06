# Plan: Compiled Man Documentation (bbolt)

## Motivation

Today man docs are distributed as many small YAML files — split per
subcommand per host for `cmd://`, and one `map[string]string` file per host
for static schemes. For carapace-bin alone that's hundreds of files. A
compiled format lets providers ship a **single file** containing all
UID→doc mappings, without the spec's flag *structure* (FlagSet, completion
definitions, run blocks, etc.) — just the documentation payload.

bbolt (`go.etcd.io/bbolt`) is a pure-Go, CGO-free, file-backed B+tree store.
It fits the project's dependency profile (no CGO, single binary) and gives us
ordered key iteration for free, which we need for completion listing.

---

## Scope

All UID schemes that are file-backed are in scope:

| Scheme | Source format | UID example |
|--------|-------------|-------------|
| `cmd` | carapace-spec YAML (split per subcommand) | `cmd://git/commit?flag=message` |
| `<other>` (static) | `map[string]string` YAML (one file per host) | `git://color/normal`, `crush://model/aihubmix/DeepSeek-R1` |

`man://` is **out of scope** — it shells out to the system `man` binary and
has no files to compile.

**Coexistence:** Compiled DBs do **not** replace the YAML authoring
workflow (`split-to`, `update`, `sync`). YAML remains the source format for
writing/editing docs. The compiled DB is a **distribution artifact** — what
gets shipped to end-user systems. Lookup checks compiled DBs first, then
falls back to YAML files for backward compatibility and user overrides.

---

## DB Format

### File layout

One bbolt file per provider. File extension: `.db`.

```
$XDG_DATA_HOME/carapace/man/db/
  carapace-bin.db      # carapace-bin: hundreds of cmd hosts + static schemes
  gh.db                # single-command completer: one cmd host
  docker.db            # another single-command completer
  ...
```

A system-level path can be added later (e.g. `/usr/share/carapace/man/db/`)
for distro-packaged DBs. For now: `$XDG_DATA_HOME/carapace/man/db/` +
optional `CARAPACE_MAN_DB_PATH` env var for custom locations.

### Bucket structure — two-level nesting (scheme → host)

**Top-level bucket = scheme.** **Nested bucket = host.** This avoids the
collision where `git` is both a `cmd` host (`cmd://git/...`) and a static
scheme name (`git://color/normal`).

```
Top-level bucket: "cmd"
  Nested bucket: "git"
    "command"              → "git is a fast version control system"
    "commit/command"       → "git commit records changes..."
    "commit/flag/message"  → "use the given message as commit message"
    "commit/flag/m"        → "edit the commit message"
    "commit/diff/command"  → "Show changes between commits..."
  Nested bucket: "gh"
    "command"              → "gh is GitHub on the command line"
    "pr/command"           → "manage pull requests"
Top-level bucket: "git"              ← static scheme
  Nested bucket: "color"
    "normal"               → "Makes no change to the color..."
    "black"                → "Black foreground color. Bright variant: `brightblack`."
    "red"                  → "Red foreground color. Bright variant: `brightred`."
  Nested bucket: "config"
    ...keys from config.yaml...
Top-level bucket: "crush"            ← static scheme
  Nested bucket: "model"
    "aihubmix/DeepSeek-R1" → "DeepSeek's reasoning model via AiHubMix..."
    ...
Top-level bucket: "rails"            ← static scheme
  Nested bucket: "generator"
    ...
```

### Key scheme within a host bucket

**For `cmd` scheme** (from carapace-spec `Documentation` struct):
- `command` → `Documentation.Command`
- `flag/<name>` → each entry in `Documentation.Flag`
- `positional/<i>` → each entry in `Documentation.Positional`
- `positionalany` → `Documentation.PositionalAny`
- `dash/<i>` → each entry in `Documentation.Dash`
- `dashany` → `Documentation.DashAny`
- `example/<name>` → each entry in `Examples`

The path prefix (subcommand path) is prepended: `commit/command`,
`commit/flag/message`, `commit/diff/command`, etc. Root command uses no
prefix: `command`, `flag/message`.

**For static schemes** (from `map[string]string`):
- Key = the map key directly (e.g. `normal`, `black`, `aihubmix/DeepSeek-R1`)
- Value = the map value (the doc string)

No `command`/`flag/` field suffix — static docs have a flat key namespace.
The UID path (minus leading `/`) *is* the key.

### Resolving `./file` references in static docs

Static doc values starting with `./` are file paths relative to the
scheme/host directory (see `descibe()` in `pkg/man/man.go:136`). At compile
time, these references are **resolved** — the file content is read and
stored as the value. This makes the DB self-contained: no external files
needed at lookup time.

### Values

Raw string bytes (the markdown doc text). No serialization format needed —
the docs are plain strings, matching what `Documentation.Command`,
`Documentation.Flag[...]`, and `map[string]string` values already hold.

### Why not key = full UID string?

Considered a flat `cmd://git/commit?flag=message` → doc mapping. Rejected
because:
- Two-level bucketing gives free, efficient scheme + host enumeration for
  completions (list top-level buckets for schemes, list nested buckets for
  hosts within a scheme).
- Avoids URL-encoding edge cases in keys.
- Keeps each host's keys isolated and ordered for efficient prefix scans.

### What about `Examples`, `Positional`, `Dash`, etc.?

The current UID scheme only addresses `command` (no query) and `flag`
(`?flag=`). Positional/dash/example docs exist in the spec but have no UID
query form yet.

**Decision:** compile everything that exists in the `Documentation` struct +
`Examples` map regardless. The extra keys cost almost nothing and future-proof
the format when the UID scheme grows `?positional=`, `?example=` queries. The
compiler writes them; the lookup simply doesn't query them until the UID
scheme is extended.

---

## Compile Command

```
carapace-man compile <spec> <output.db>              # one cmd spec → one DB
carapace-man compile <spec-dir> <output.db>          # directory of specs → one DB
carapace-man compile - <output.db>                   # spec from stdin → one DB
carapace-man compile --static <yaml> <scheme> <host> <output.db>  # static doc → DB
carapace-man compile --append <spec> <existing.db>   # add/overwrite host in existing DB
```

### Compiling `cmd` scheme specs

1. Load spec(s) — same `command.Command` YAML unmarshal as `split-to`.
2. Walk the subcommand tree (same recursive `collectSpecsInto` traversal as
   `specdiff.go`), producing `(host, path, Documentation)` tuples.
3. Open the bbolt DB (create if not exists).
4. Create top-level bucket `cmd`, nested bucket `<host>`.
5. For each tuple, write keys as described in "Key scheme" above.
6. Skip empty values (don't write keys for empty doc strings — keeps the DB
   compact and makes "missing doc" detectable by key absence).
7. `--append` mode opens an existing DB and adds/overwrites a single host
   bucket (within `cmd`) without touching others. This lets carapace-bin's
   build process compile all completers into one DB incrementally.

### Compiling static scheme docs

1. Load the `map[string]string` YAML file.
2. Open the bbolt DB (create if not exists).
3. Create top-level bucket `<scheme>`, nested bucket `<host>`.
4. For each key→value pair:
   - If value starts with `./`: read the file at
     `<dir-of-yaml>/<value>` relative to the YAML file's location, store the
     file content as the value.
   - Otherwise: store the value as-is.
5. Skip empty values.

### Compiling a whole `man/` directory

A convenience mode to compile an entire synced `man/` tree into one DB:

```
carapace-man compile --man-dir <man-dir> <output.db>
```

This walks `<man-dir>/` and compiles:
- `<man-dir>/cmd/<host>/*.yaml` → `cmd` scheme, each subdirectory = one host
- `<man-dir>/<scheme>/<host>/<host>.yaml` → static scheme, each file = one host
  (skipping the `cmd/` directory which is handled above)

This is what carapace-bin's build process would use to produce its bundled DB.

### Host derivation

- For a single-spec compile (`compile <spec> <db>`), the host is derived from
  the root command name (first word of `cmd.Name`), matching how `split-to`
  derives filenames.
- For a directory compile (`compile <spec-dir> <db>`), each `*.yaml` file in
  the directory is a separate host, filename = host name (matching the
  `man/cmd/<host>/` directory convention).
- For static compile, scheme and host are explicit arguments (since they
  can't be derived from the YAML file alone).
- For `--man-dir` compile, scheme = directory name, host = subdirectory name
  (for static schemes) or directory name (for `cmd`).

### What about documentation merging?

`split-to`/`update` merge existing `documentation:` entries to preserve
human-written docs. The compiler reads from already-merged YAML files (the
output of `split-to`/`update`), so it just serializes what's there. No merge
logic needed in the compiler — it's a pure transformer of YAML → bbolt.

---

## Lookup / Resolution Changes

### Current flow

```
cmd(uid)    → cmd_user(uid)  [user specs at $XDG_CONFIG_HOME/carapace/specs/]
            → repo YAML      [$XDG_CONFIG_HOME/carapace/man/cmd/<host>/...]

descibe(uid) → repo YAML     [$XDG_CONFIG_HOME/carapace/man/<scheme>/<host>/<host>.yaml]
```

### Proposed flow

```
cmd(uid)     → cmd_user(uid)  [user specs — unchanged, highest priority]
             → cmd_db(uid)    [compiled bbolt DBs — NEW, second priority]
             → repo YAML      [synced YAML — fallback, lowest priority]

descibe(uid) → describe_db(uid) [compiled bbolt DBs — NEW, first priority]
             → repo YAML         [synced YAML — fallback]
```

### `cmd_db(uid)` — in a new `pkg/man/db.go`

1. Resolve scheme=`cmd`, host=`uid.Host` from the UID.
2. Look up `(scheme, host)` in the (scheme,host)→DB index (see next section).
3. If found, open that DB read-only, get top-level bucket `cmd`, nested
   bucket `<host>`.
4. Build the key from `uid.Path` + query:
   - `?flag=X` → `<path>/flag/X`
   - no query → `<path>/command`
5. Read the key. If present, return the value. If absent, return `("", nil)`
   (not found — fall through to YAML).
6. If the host isn't in any DB, return not-found (fall through to YAML).

### `describe_db(uid)` — in `pkg/man/db.go`

1. Resolve scheme=`uid.Scheme`, host=`uid.Host`.
2. Look up `(scheme, host)` in the index.
3. If found, open DB read-only, get top-level bucket `<scheme>`, nested
   bucket `<host>`.
4. Key = `strings.TrimPrefix(uid.Path, "/")` (the map key directly).
5. Read the key. Return value or fall through to YAML.

### Read-only, no locking issues

bbolt supports concurrent read-only transactions. Multiple `carapace-man`
processes can open the same DB read-only simultaneously without contention.
No daemon needed.

---

## Loading Multiple DBs (the discussion point)

### The problem

- carapace-bin ships one DB containing hundreds of `cmd` hosts plus multiple
  static schemes (`git`, `crush`, `rails`, etc.).
- A single-command completer (e.g. `gh`) ships one DB with one `cmd` host.
- Multi-completers (carapace-bin) break the "one DB = one host = derive
  filename from host name" assumption.

We need to know **which DB file contains a given (scheme, host)** without
opening every DB and scanning all buckets on every lookup.

### Proposed solution: build a (scheme,host)→DB index on each invocation

Since `carapace-man` is a short-lived CLI (not a daemon), build the index
fresh each invocation:

1. Enumerate all `*.db` files in the search paths
   (`$XDG_DATA_HOME/carapace/man/db/` + `CARAPACE_MAN_DB_PATH`).
2. For each DB, open read-only, list top-level bucket names (= schemes).
3. For each scheme bucket, list nested bucket names (= hosts).
4. Build `map[scheme]map[host]dbPath` (or a flattened `map[scheme+"\x00"+host]dbPath`).
5. Use the map for the rest of the lookup.

### Cost

- bbolt's `BucketNames()` is O(number of buckets) but each bucket read is a
  single B+tree page. For carapace-bin (~500 cmd hosts + ~20 static hosts)
  this is a handful of page reads.
- With ~5-10 DB files on a typical system, total index build is a few
  milliseconds. Acceptable for a CLI that's already parsing YAML and
  rendering markdown through glamour.

### Why not cache the index?

Could write a `db-index.json` cache with mtime invalidation. But:
- The index build is cheap (single-digit ms).
- Cache invalidation adds complexity and stale-cache bugs.
- **Defer caching until profiling shows it's needed.** Keep v1 simple.

### Why not embed a manifest in each DB?

A `_meta` bucket listing scheme/host pairs would let us read one bucket
instead of listing all. But `BucketNames()` on top-level + nested buckets
already does exactly this. A separate manifest would be redundant with the
bucket structure itself.

### Conflict handling

If two DBs claim the same (scheme, host) pair, **first one wins**
(deterministic: sort DB file paths alphabetically, first match wins). Print
a warning to stderr. This lets user-local DBs (in `$XDG_DATA_HOME`) override
system DBs by convention of search-path ordering.

### Search path ordering (first match wins)

1. `CARAPACE_MAN_DB_PATH` (colon-separated, if set)
2. `$XDG_DATA_HOME/carapace/man/db/`
3. (future) system path like `/usr/share/carapace/man/db/`

---

## Completion Integration

`ActionUids` (`pkg/actions/man/man.go`) currently enumerates schemes/hosts/UIDs
by reading the filesystem (`man.Schemes()`, `man.Hosts("cmd")`, `man.Uids()`,
and the static scheme logic in `actionOther()`).

### Changes needed

Add DB-sourced enumeration functions in `pkg/man/`:

- `SchemesFromDBs()`: return all scheme names (top-level bucket names) across
  all DBs.
- `HostsFromDBs(scheme)`: return all hosts for a scheme (nested bucket names).
- `UidsFromDBs(scheme, host)`: list keys in the (scheme, host) bucket,
  reconstruct UIDs. For `cmd`: filter `*/command` keys for command UIDs,
  enumerate `*/flag/*` keys for `?flag=` completion. For static schemes:
  keys are the UID paths directly.

`actionCmds()` and `actionOther()` merge results from both filesystem (YAML)
and DB sources, deduplicating. DB sources take precedence in the displayed
list (since they're what the lookup will actually use).

---

## New Package / File Layout

```
pkg/man/
  db.go              NEW — bbolt open/close, (scheme,host)→DB index,
                          cmd_db() + describe_db() lookup
  db_compile.go      NEW — walk spec tree / static map, write to bbolt
  man.go             MODIFIED — Describe() calls describe_db() before descibe()
  command.go         MODIFIED — cmd() calls cmd_db() as second-priority fallback
pkg/actions/man/
  man.go             MODIFIED — actionCmds() + actionOther() merge DB-sourced data
cmd/carapace-man/cmd/
  compile.go         NEW — cobra command for `carapace-man compile`
```

### Dependency

Add `go.etcd.io/bbolt` to `go.mod`. Pure Go, no CGO, no transitive deps
beyond the standard library. Compatible with Go 1.25.

---

## Open Questions / Decisions Needed

### 1. Should compiled DBs include `Examples`?

Examples (`map[string]string`) are in the spec but not addressable by the
current UID scheme. Including them in the DB is cheap and future-proof, but
bloats the DB slightly. **Recommendation: include them** (see "What about
Examples" above).

### 2. Should `compile` support a `--merge` flag to combine multiple specs into an existing DB?

`--append` (add/overwrite one host) is planned. A broader `--merge` (copy
all buckets from one DB into another) could be useful for carapace-bin's
build process if they compile each completer separately and then merge.
**Recommendation: start with `--append`, add `--merge` if the build process
needs it.**

### 3. Key encoding for paths with special characters?

Host names and paths are derived from command names, which are typically
alphanumeric + hyphens. bbolt keys are arbitrary byte slices, so no encoding
needed. Flag names can contain dots (`--global.flag`) — these work fine as
raw key bytes. Static doc keys can contain slashes (`aihubmix/DeepSeek-R1`)
— also fine as raw key bytes. No URL-encoding needed since we're not using
URL strings as keys.

### 4. Should we version the DB format?

A `_meta` bucket with a format version key would let us evolve the schema
without silent breakage. **Recommendation: yes** — write `_meta/format =
"1"` on compile, check on open. Cheap insurance. The `_meta` bucket would
sit alongside the scheme buckets at the top level and be skipped during
scheme/host enumeration.

### 5. Garbage collection / stale keys on recompile?

When `--append` overwrites a host bucket, should it first clear the bucket
to remove keys for subcommands that no longer exist? **Recommendation: yes**
— `boltbucket.DeleteBucket()` + recreate, or iterate and delete missing
keys. This prevents stale docs for removed subcommands.

### 6. Should the `sync` command also download compiled DBs?

Currently `sync` clones the `carapace-sh/man` repo (YAML). A separate
`sync-db` or `sync --db` could download pre-compiled DBs from a release
artifact. **Recommendation: defer** — let providers ship DBs via package
managers or their own install scripts. `sync` stays YAML-focused.

### 7. Should `--man-dir` compile also handle user specs at `$XDG_CONFIG_HOME/carapace/specs/`?

User specs are per-host YAML files that override repo specs. They're not
part of the `man/` synced tree. **Recommendation: no** — `--man-dir`
compiles the synced tree only. User specs remain a runtime fallback
(`cmd_user()` has highest priority at lookup time, regardless of DBs).

---

## Implementation Phases

### Phase 1 — Core compile + lookup (MVP)

1. Add `go.etcd.io/bbolt` dependency.
2. Implement `pkg/man/db_compile.go`: walk cmd spec tree, write to bbolt
   (two-level buckets: scheme → host → keys).
3. Implement `pkg/man/db_compile.go` static doc compilation: load
   `map[string]string`, resolve `./` file references, write to bbolt.
4. Implement `pkg/man/db.go`: open DB, (scheme,host)→DB index, `cmd_db()`
   and `describe_db()` lookup functions.
5. Add `cmd/carapace-man/cmd/compile.go` cobra command (spec, static, and
   `--man-dir` modes).
6. Wire `cmd_db()` into `cmd()` in `command.go` as second-priority
   resolution. Wire `describe_db()` into `Describe()` in `man.go` before
   `descibe()`.
7. Write basic integration test: compile a small cmd spec → lookup a UID
   from the DB → verify output matches YAML path. Same for a static doc.

### Phase 2 — Completion integration

8. Implement `SchemesFromDBs()` / `HostsFromDBs()` / `UidsFromDBs()` in
   `pkg/man/`.
9. Modify `actionCmds()` and `actionOther()` to merge DB + filesystem
   sources.

### Phase 3 — Polish

10. `--append` mode for incremental compilation.
11. `_meta` format versioning.
12. Stale-key cleanup on recompile (bucket wipe + rewrite).
13. `CARAPACE_MAN_DB_PATH` env var support.
14. Conflict detection + stderr warning.
15. `--man-dir` convenience mode for full-tree compilation.

### Phase 4 — Ecosystem (out of this repo)

16. carapace-bin build process: `compile --man-dir` the entire `man/` tree
    into one `.db`, ship it as a release artifact or package it with the
    binary.
17. Individual completers: `compile` their own spec, ship the `.db`.
18. Distro packaging: install `.db` files to `/usr/share/carapace/man/db/`.
