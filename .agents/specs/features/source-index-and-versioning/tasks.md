# Source Index and Versioning Tasks

**Design**: `.agents/specs/features/source-index-and-versioning/design.md`
**Status**: DONE — V1–V7 ✅. Package-level SemVer via `harness.artifacts.yaml` (index-driven resolution with convention fallback); version surfaced in list/search/upgrade; project manifest at the root with source/version/digest; `harness.lock` retired; `harness apply` reconciles + verifies. All `make check` green.

> Tooling: self-contained Go CLI, no MCPs. Execution skill: `tlc-spec-driven`. Every task ends `make check` green.

---

## Execution Plan

```
Phase 1 (versioning core, sequential):
  V1 (SemVer + artifact.Version) → V2 (artifacts manifest parse) → V3 (LocalDirectory index-aware) → V4 (surface version in list/search/upgrade)

Phase 2 (root manifest, after V1):
  V5 (manifest → root, enrich Selection, retire lock) → V6 (Run records version+digest)

Phase 3 (after V5, V6):
  V7 (harness apply + digest verify)
```

---

## Task Breakdown

### V1: SemVer validation + `artifact.Version`

**What**: Add a validated `Version` to the artifact model.
**Where**: `internal/artifact/version.go` (new), `internal/artifact/artifact.go`
**Depends on**: None
**Requirement**: SIV-02, SIV-03

**Done when**:

- [ ] `ValidateVersion` accepts `1.3.0` and `1.0.0-rc.1+build`, rejects `1.0`, `v1`, ``.
- [ ] `artifact.Artifact` has `Version string` (empty = unversioned).
- [ ] `go test ./internal/artifact/...` covers valid/invalid cases.

**Verify**: `go test ./internal/artifact/...`
**Commit**: `feat(artifact): add validated SemVer version`

---

### V2: Parse `harness.artifacts.yaml`

**What**: Load and shape a source's package manifest.
**Where**: `internal/source/artifactsmanifest.go`
**Depends on**: V1
**Requirement**: SIV-01

**Done when**:

- [ ] `LoadArtifactsManifest(path)` returns the manifest, a presence bool, and a clear error on malformed YAML.
- [ ] `ArtifactEntry{Kind,Name,Version,Path}` round-trips.
- [ ] Test covers present / absent / malformed.

**Verify**: `go test ./internal/source/...`
**Commit**: `feat(source): parse harness.artifacts.yaml package manifest`

---

### V3: `LocalDirectory` index-aware resolution

**What**: Resolve from the package manifest when present, else by convention; `GitRepository` inherits it.
**Where**: `internal/source/local.go`
**Depends on**: V2
**Requirement**: SIV-01, SIV-03, SIV-04, SIV-05

**Done when**:

- [ ] With an index: resolves listed entries from their `path`, validates `frontmatter.name == entry.Name` and the version, stamps `Version`.
- [ ] Without an index: today's convention scan, `Version == ""`.
- [ ] Invalid version, missing path, path escape, name mismatch, duplicate → `Issue` (skipped), rest continue.
- [ ] Git fixture test: index in non-conventional paths resolves with versions; no-index fixture resolves by convention.

**Verify**: `go test ./internal/source/...`
**Commit**: `feat(source): index-driven resolution with convention fallback`

---

### V4: Surface version in list / search / upgrade

**What**: Show versions to the user and report transitions.
**Where**: `internal/app/app.go` (List), `internal/index` (Record + search), `internal/app/upgrade.go`
**Depends on**: V3
**Requirement**: SIV-01

**Done when**:

- [ ] `list` and `search` show the version (or `unversioned`).
- [ ] index `Record` carries `version`; `update` writes it.
- [ ] `upgrade` reports `name X → Y` using versions when available.

**Verify**: `go test ./...`; smoke `search`
**Commit**: `feat(app): surface artifact versions in list, search and upgrade`

---

### V5: Project manifest → root; enrich `Selection`; retire lock

**What**: Move `harness.yaml` to the project root, record source/version/digest, drop `harness.lock`.
**Where**: `internal/config/paths.go`, `internal/workspace/manifest.go`, `internal/workspace/writer.go`
**Depends on**: V1
**Requirement**: SIV-06, SIV-07

**Done when**:

- [ ] `config.ManifestPath` → `<projectRoot>/harness.yaml`; `LockPath`/lock paths removed.
- [ ] `Selection` gains `Source`, `Version`, `Digest`; `manifestVersion = 2`.
- [ ] `Apply` writes the root manifest and removes a stale `.agents/harness.yaml` / `harness.lock`.
- [ ] Tests updated (workspace).

**Verify**: `go test ./internal/workspace/... ./internal/config/...`
**Commit**: `refactor(workspace): root manifest with source/version/digest, retire lock`

---

### V6: `Run` records version + digest into the manifest

**What**: Replace lock-writing with manifest enrichment on save.
**Where**: `internal/app/app.go`, `internal/app/upgrade.go`
**Depends on**: V5
**Requirement**: SIV-06, SIV-07

**Done when**:

- [ ] On save, vendored remotes get a `digest`; all selections get `source`+`version`; no `harness.lock` written.
- [ ] `Upgrade` reads the root manifest (not the lock), re-resolves remotes, updates version+digest, reports changes.
- [ ] `internal/lock` reduced to `ContentHash` (Lockfile removed) or re-homed.
- [ ] Tests updated (app).

**Verify**: `go test ./internal/app/...`; end-to-end smoke
**Commit**: `feat(app): record version and digest in the root manifest`

---

### V7: `harness apply` — reconcile from the manifest + verify digests

**What**: Materialize a project from its committed `harness.yaml` without the TUI.
**Where**: `internal/app/apply.go`, `main.go`
**Depends on**: V5, V6
**Requirement**: SIV-08

**Done when**:

- [ ] `apply` resolves every selection, re-vendors remotes, regenerates `AGENTS.md`.
- [ ] Matching digest → untouched; drift → reported; missing source/artifact → reported, continues.
- [ ] Restores a deleted vendored artifact to match its digest.
- [ ] Test against a `file://` fixture; `main` dispatch + help.

**Verify**: `go test ./internal/app/... -run Apply`
**Commit**: `feat(app): add apply to reconcile a project from its manifest`

---

## Coverage Check

SIV-01→V2/V3/V4 · SIV-02→V1/V3 · SIV-03→V1/V3 · SIV-04→V3 · SIV-05→V3 · SIV-06→V5/V6 · SIV-07→V5/V6 · SIV-08→V7. **0 unmapped.**
