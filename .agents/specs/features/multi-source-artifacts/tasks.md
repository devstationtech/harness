# Multi-Source Artifact Management Tasks

**Design**: `.agents/specs/features/multi-source-artifacts/design.md`
**Status**: Phase 1 done (T1–T3 ✅); Phase 2 next

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

### T4: Implement the `GitRepository` adapter (system-git wrapper) [P]

**What**: Clone-or-pull a git source and resolve artifacts over the checked-out tree.
**Where**: `internal/source/git.go`; `internal/source/gitcli/gitcli.go`
**Depends on**: T3
**Reuses**: `LocalDirectory` for scanning the clone
**Requirement**: SRC-01, SRC-05, SRC-06

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [ ] `exec.LookPath("git")` with actionable error when absent (SRC-06).
- [ ] Commands use `exec.Command("git", args...)` slice form, `-c core.autocrlf=false -c core.eol=lf`, env `GIT_TERMINAL_PROMPT=0`.
- [ ] Clone into temp dir then `os.Rename` on success.
- [ ] `Sync()` clones when absent, fetches+checks out the ref when present.
- [ ] Test against a local `file://` git repo fixture (no network): resolves its artifacts.

**Verify**: `go test ./internal/source/... -run Git`

**Commit**: `feat(source): add GitRepository adapter over the system git binary`

---

### T5: `sources.yaml` config load/save + new config paths [P]

**What**: Read/write `~/.harness/sources.yaml`; add `sources/`, `index/`, `harness.lock` path helpers.
**Where**: `internal/config/paths.go`; `internal/config/sources.go` (new)
**Depends on**: T3
**Reuses**: yaml pattern from `workspace/manifest.go`
**Requirement**: SRC-01, SRC-09

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [ ] `SourcesConfig` round-trips through yaml.v3.
- [ ] Path helpers `SourcesConfigPath`, `SourceCloneDir(name)`, `IndexDir`, `LockPath(projectRoot)` added.
- [ ] Missing file → empty config (no error), mirroring `LoadManifest`.

**Verify**: `go test ./internal/config/...`

**Commit**: `feat(config): add sources.yaml and source/index/lock paths`

---

### T6: Content hash + `harness.lock` read/write [P]

**What**: Stable directory content hash and lockfile (de)serialization.
**Where**: `internal/lock/lock.go`
**Depends on**: T3
**Reuses**: yaml pattern; sha256
**Requirement**: SRC-04, SRC-07

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [ ] `ContentHash(dir)` walks sorted, normalizes `\r\n`→`\n` for text, returns `sha256:...`.
- [ ] Test asserts identical hash for a tree differing only in line endings (proves SRC-07).
- [ ] `Lockfile` Load/Save round-trips; `path` stored forward-slash.

**Verify**: `go test ./internal/lock/...`

**Commit**: `feat(lock): add cross-platform content hashing and harness.lock`

---

### T7: Vendor a remote artifact into the project

**What**: Copy a `Payload` directory into `.agents/<container>/<name>/`, return a `lock.Entry`.
**Where**: `internal/vendor/vendor.go`
**Depends on**: T4, T5, T6
**Reuses**: on-demand dir creation from `workspace/writer.go`
**Requirement**: SRC-03, SRC-04

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [ ] Directory tree copied (not symlinked); container dir created on demand.
- [ ] Returns `Entry` with source, commit, content hash, forward-slash path.
- [ ] Re-vendoring identical content is idempotent; differing content surfaces a hash change.
- [ ] `go test ./internal/vendor/...` covers copy + idempotency.

**Verify**: `go test ./internal/vendor/...`

**Commit**: `feat(vendor): materialize remote artifacts into the project`

---

### T8: Wire save flow to vendor remote selections and write the lock

**What**: On TUI confirm, vendor remote-source selections, write `harness.lock`, then existing `AGENTS.md` + `harness.yaml`.
**Where**: `internal/app/app.go` (`Run`); `internal/workspace` as needed
**Depends on**: T7
**Reuses**: `workspace.Apply`, `tui.Run` result
**Requirement**: SRC-03, SRC-04

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [ ] Selecting a remote artifact and saving copies it under `.agents/` and writes a lock entry.
- [ ] Local/home selections behave exactly as today (no vendoring).
- [ ] Re-apply from an existing lock verifies hashes and errors on mismatch (SRC-04 ac#4).
- [ ] End-to-end manual test from a `file://` fixture passes.

**Verify**: `go build . && ./harness` against a local fixture source; inspect `.agents/` and `harness.lock`.

**Commit**: `feat(app): vendor and lock remote selections on save`

---

### T9: `harness source add | list | remove`

**What**: CLI subcommands managing `sources.yaml` and clones.
**Where**: `main.go` dispatch; `internal/app/source.go` (new)
**Depends on**: T8
**Reuses**: `config.Sources*`, `source` adapters
**Requirement**: SRC-01, SRC-09

**Tools**: MCP: NONE · Skill: NONE

**Done when**:

- [ ] `add` records the source and clones it (temp+rename); rejects duplicate names.
- [ ] `list` prints name/type/url/ref.
- [ ] `remove` deletes config entry, clone, and indexed manifests; leaves vendored artifacts and locks untouched.
- [ ] `go test` covers add/list/remove against a `file://` fixture.

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
