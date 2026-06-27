# Multi-Source Artifact Management Tasks

**Design**: `.agents/specs/features/multi-source-artifacts/design.md`
**Status**: T1–T9 ✅ — end-to-end works (`source add` a git repo → its artifacts appear in selection → save vendors + locks them). Remaining: T10 index/`update`, T11 `search`, T12 `upgrade`.

> Tooling note (TLC "ASK about MCPs/Skills"): this is a self-contained Go CLI — tasks need **no MCPs**. The relevant skill during execution is `tlc-spec-driven` itself (verify-per-task, atomic commits). Diagram/exploration skills (`mermaid-studio`, `codenavi`) are not installed; inline mermaid is used.

> Implementation note (decision D8): the `Source` port returns resolved `artifact.Artifact` and has no `Fetch`; `Issue` lives in the `source` package. `Manifest` (serializable projection) and any lazy `Payload` fetch are deferred to the index phase (T10), where they are actually needed. Vendoring (T7) copies `Artifact.Directory` directly.

---

## Execution Plan

### Phase 1 — Source port foundation (Sequential)

```
T1 → T2 → T3
```

### Phase 2 — Git source + reproducibility (Parallel after T3)

```
        ┌→ T4 (git adapter) ─┐
T3 ─────┼→ T5 (sources.yaml) ─┼──→ T7 (vendor) → T8 (wire save+lock)
        └→ T6 (lock + hash) ──┘
```

### Phase 3 — Source commands (Sequential after T8)

```
T8 → T9 (source add/list/remove)
```

### Phase 4 — Index, search, upgrade (Parallel after T9)

```
        ┌→ T10 (index build / update) → T11 (search)
T9 ─────┤
        └→ T12 (upgrade)
```

---

## Task Breakdown

### T1: Define the `Source` port ✅

**What**: Create the source contract and the `Issue` value type (no adapters yet).
**Where**: `internal/source/source.go`
**Depends on**: None
**Reuses**: `internal/artifact` types
**Requirement**: SRC-02

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [x] `Source` interface (`Name`, `Resolve`) and `Issue` defined and documented. (`Manifest`/`Payload`/`Fetch` deferred per D8.)
- [x] Package compiles with no adapter yet.
- [x] `go build ./...` and `go vet ./...` clean.

**Verify**: `go build ./internal/source/`

**Commit**: `feat(source): define Source port with Manifest and Payload`

---

### T2: Extract `LocalDirectory` adapter from the current catalog scan ✅

**What**: Move the existing `scanBase`/`readArtifact` logic into a `LocalDirectory` source.
**Where**: `internal/source/local.go` (new); `internal/catalog/catalog.go` (remove moved code)
**Depends on**: T1
**Reuses**: existing scan logic, `frontmatter`
**Requirement**: SRC-02

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [x] `NewLocalDirectory(name, base, tag)` resolves the same artifacts the old `scanBase` did.
- [x] Invalid-artifact `Issue`s are produced identically.
- [x] `go test ./internal/source/...` covers resolve, name-mismatch issue, ignored dir, empty/missing base.

**Verify**: `go test ./internal/source/... ./internal/catalog/...`

**Commit**: `refactor(source): extract LocalDirectory adapter from catalog scan`

---

### T3: Refactor `catalog.Load` to merge ordered sources with precedence ✅

**What**: Change `Load(home, agentsDir)` to `Load(sources ...source.Source)`; precedence = source order (highest first); keep `All/ByKind/Find/Issues`.
**Where**: `internal/catalog/catalog.go`; callers in `internal/app/app.go`
**Depends on**: T2
**Reuses**: `Identity`, sort order
**Requirement**: SRC-02

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [x] `app.loadCatalog` builds `[project(.agents), home(~/.harness)]` (highest first) and passes them in.
- [x] Catalog tests cover merge / precedence override / order / issue passthrough (black-box, fake source).
- [x] Higher-precedence source wins and flags the shadowed artifact (`OverridesShared`, name to generalize when remote sources land).

**Verify**: `make check` green (build + vet + lint + tests)

**Commit**: `refactor(catalog): merge an ordered list of sources by precedence`

---

### T4: Implement the `GitRepository` adapter (system-git wrapper) [P] ✅

**What**: Clone-or-pull a git source and resolve artifacts over the checked-out tree.
**Where**: `internal/source/git.go`; `internal/source/gitcli/gitcli.go`
**Depends on**: T3
**Reuses**: `LocalDirectory` for scanning the clone
**Requirement**: SRC-01, SRC-05, SRC-06

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [x] `gitcli.Available()` via `exec.LookPath("git")` with actionable `ErrNotFound` (SRC-06).
- [x] Commands use `exec.CommandContext("git", args...)` slice form, `-c core.autocrlf=false -c core.eol=lf`, env `GIT_TERMINAL_PROMPT=0`; `ctx` first param (SRC-05).
- [x] Clone into a staging dir on the same volume then `os.Rename` on success.
- [x] `Sync(ctx)` clones when absent, fetches+checks out the ref when present; `Resolve()` reads the checkout offline; `Commit(ctx)` reports the SHA.
- [x] Tests against a local `file://` git fixture (no network): resolve + idempotent re-sync.

**Verify**: `go test ./internal/source/... -run Git`

**Commit**: `feat(source): add GitRepository adapter over the system git binary`

---

### T5: `sources.yaml` config load/save + new config paths [P] ✅

**What**: Read/write `~/.harness/sources.yaml`; add `sources/`, `index/`, `harness.lock` path helpers.
**Where**: `internal/config/paths.go`; `internal/config/sources.go` (new)
**Depends on**: T3
**Reuses**: yaml pattern from `workspace/manifest.go`
**Requirement**: SRC-01, SRC-09

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [x] `SourcesConfig` round-trips through yaml.v3 (go-cmp test); `Save` creates the parent dir; `Find` looks up by name.
- [x] Path helpers `SourcesConfigPath`, `SourcesDir`, `SourceCloneDir(name)`, `IndexDir`, `LockPath(projectRoot)` added.
- [x] Missing file → empty config (no error), mirroring `LoadManifest`.

**Verify**: `go test ./internal/config/...`

**Commit**: `feat(config): add sources.yaml and source/index/lock paths`

---

### T6: Content hash + `harness.lock` read/write [P] ✅

**What**: Stable directory content hash and lockfile (de)serialization.
**Where**: `internal/lock/lock.go`
**Depends on**: T3
**Reuses**: yaml pattern; sha256
**Requirement**: SRC-04, SRC-07

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [x] `ContentHash(dir)` walks in sorted forward-slash order, normalizes `\r\n`→`\n`, mixes in the relative path, excludes mode, returns `sha256:...`.
- [x] Test asserts identical hash for a tree differing only in line endings (proves SRC-07); plus content- and path-sensitivity.
- [x] `Lockfile` Load/Save round-trips (go-cmp); `path` stored forward-slash; missing file → empty.

**Verify**: `go test ./internal/lock/...`

**Commit**: `feat(lock): add cross-platform content hashing and harness.lock`

---

### T7: Vendor a remote artifact into the project ✅

**What**: Copy a remote artifact's directory into `.agents/<container>/<name>/`, return a `lock.Entry`.
**Where**: `internal/vendor/vendor.go`
**Depends on**: T4, T5, T6
**Reuses**: `lock.ContentHash`; `config.AgentsDir`
**Requirement**: SRC-03, SRC-04

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [x] Directory tree copied recursively (not symlinked), preserving perms; container dir created on demand.
- [x] Returns the now-local artifact plus an `Entry` with source, commit, content hash, forward-slash path.
- [x] Re-vendoring the same revision is idempotent (stable hash); `artifact.Origin` distinguishes remote from local.
- [x] `go test ./internal/vendor/...` covers copy of nested files + idempotency.

**Verify**: `go test ./internal/vendor/...`

**Commit**: `feat(vendor): materialize remote artifacts into the project`

---

### T8: Wire save flow to vendor remote selections and write the lock ✅

**What**: On TUI confirm, vendor remote-source selections, write `harness.lock`, then existing `AGENTS.md` + `harness.yaml`.
**Where**: `internal/app/app.go` (`Run`, `materialize`, `writeLock`); `loadCatalog`/`projectSources` include git sources
**Depends on**: T7
**Reuses**: `workspace.Apply`, `tui.Run` result, `vendor.Vendor`, `lock`
**Requirement**: SRC-03, SRC-04

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [x] Selecting a remote artifact and saving copies it under `.agents/` and writes a sorted `harness.lock`; the lock is removed when nothing remote is selected.
- [x] Local/home selections pass through untouched (referenced in place).
- [x] `loadCatalog` resolves configured git sources (existing clones, no network) so remote artifacts appear in selection.
- [x] End-to-end smoke from a `file://` fixture passes (`source add` → `list` shows remote artifacts).
- [ ] **Deferred**: verifying vendored content against an existing lock and erroring on mismatch (SRC-04 ac#4) belongs to a `verify`/frozen-apply flow, not the normal save (where re-selecting legitimately changes content). Tracked as deferred idea D9.

**Verify**: end-to-end `file://` smoke; `go test ./internal/app/...`

**Commit**: `feat(app): vendor and lock remote selections on save`

---

### T9: `harness source add | list | remove` ✅

**What**: CLI subcommands managing `sources.yaml` and clones.
**Where**: `main.go` dispatch; `internal/app/source.go` (new)
**Depends on**: T8
**Reuses**: `config.Sources*`, `source` adapters
**Requirement**: SRC-01, SRC-09

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [x] `add <url> [--name] [--ref]` validates the name, clones (temp+rename via the adapter), rejects duplicates; URL is first arg (git-style).
- [x] `list` prints name/type/url/ref.
- [x] `remove` deletes the config entry and clone; leaves vendored artifacts and locks untouched. (Index removal lands with T10.)
- [x] `go test ./internal/app/... -run Source` covers add/list/remove + duplicate rejection against a `file://` fixture.

**Verify**: `go test ./internal/app/... -run Source`

**Commit**: `feat(app): add source add/list/remove commands`

---

### T10: Manifest index build + `harness update` [P]

**What**: Refresh sources and persist per-source manifest files for offline use.
**Where**: `internal/index/index.go`; `internal/app/update.go`
**Depends on**: T9
**Reuses**: `source.Resolve`, yaml
**Requirement**: SRC-07

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [ ] `update` syncs each git source and writes `~/.harness/index/<source>.yaml`.
- [ ] Unreachable source keeps its prior index file and reports a warning (no abort).
- [ ] `go test ./internal/index/...` covers build + stale-keep.

**Verify**: `go test ./internal/index/...`

**Commit**: `feat(index): build manifest index and add update command`

---

### T11: `harness search <query>` (offline)

**What**: Case-insensitive substring search over the local index.
**Where**: `internal/index/index.go` (`Search`); `internal/app/search.go`
**Depends on**: T10
**Reuses**: `index.Load`
**Requirement**: SRC-08

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [ ] Matches name+description across sources; prints `<source>/<name> | kind | description`.
- [ ] Builds index on demand if absent.
- [ ] Reads no network (test with index present, sources removed).

**Verify**: `go test ./internal/index/... -run Search`

**Commit**: `feat(app): add offline search command`

---

### T12: `harness upgrade` (re-resolve + report) [P]

**What**: Re-resolve a project's locked artifacts against current source refs; re-vendor changed; update lock; report diffs.
**Where**: `internal/app/upgrade.go`
**Depends on**: T9
**Reuses**: `vendor`, `lock`, `source`
**Requirement**: SRC-10

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [ ] Changed artifacts are re-vendored and the lock updated (new commit/hash).
- [ ] Reports each change as old→new commit.
- [ ] Missing-commit (force-push) yields a clear error; vendored content left intact.
- [ ] `go test ./internal/app/... -run Upgrade` against a mutated `file://` fixture.

**Verify**: `go test ./internal/app/... -run Upgrade`

**Commit**: `feat(app): add upgrade to re-resolve locked artifacts`

---

## Parallel Execution Map

```
Phase 1 (Sequential):
  T1 ──→ T2 ──→ T3

Phase 2 (after T3):
    ├── T4 [P]  (git adapter)
    ├── T5 [P]  (sources.yaml + paths)
    └── T6 [P]  (lock + hash)
        then T7 (vendor) ──→ T8 (wire save+lock)

Phase 3 (after T8):
  T9 (source add/list/remove)

Phase 4 (after T9):
    ├── T10 (index/update) ──→ T11 (search)
    └── T12 [P] (upgrade)
```

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1 Source port + types | 1 file, contracts | ✅ Granular |
| T2 LocalDirectory adapter | 1 adapter (moved logic) | ✅ Granular |
| T3 catalog refactor | 1 function signature + callers | ✅ Granular |
| T4 Git adapter | 1 adapter + cli wrapper | ⚠️ Cohesive pair, OK |
| T5 sources.yaml + paths | 1 config concern | ✅ Granular |
| T6 hash + lock | 1 file, cohesive | ✅ Granular |
| T7 vendor | 1 function | ✅ Granular |
| T8 wire save | 1 integration point | ✅ Granular |
| T9 source commands | 3 thin subcommands, 1 concern | ⚠️ Cohesive, OK |
| T10 index/update | 1 builder + 1 command | ⚠️ Cohesive, OK |
| T11 search | 1 function + 1 command | ✅ Granular |
| T12 upgrade | 1 command | ✅ Granular |

---

## Coverage Check

All 10 requirements map to tasks: SRC-01→T4/T5/T9 · SRC-02→T1/T2/T3 · SRC-03→T7/T8 · SRC-04→T6/T7/T8 · SRC-05→T4 · SRC-06→T4 · SRC-07→T6/T10 · SRC-08→T11 · SRC-09→T5/T9 · SRC-10→T12. **0 unmapped.**
