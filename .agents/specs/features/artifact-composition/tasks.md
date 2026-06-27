# Artifact Composition Tasks

**Design**: `.agents/specs/features/artifact-composition/design.md`
**Status**: Draft

> Self-contained Go CLI, no MCPs. Execution skill: `tlc-spec-driven`. Each task ends `make check` green.
> The interactive composition screen (C7) touches `internal/tui`, which has uncommitted work — it is deferred until that is committed.

---

## Execution Plan

```
Phase 1 (model + render, no TUI):
  C1 (frontmatter fields + validation) → C2 (artifact carries them; source populates)
  → C3 (compose.Bind) → C4 (manifest bindings) → C5 (AGENTS.md rendering)
  C6 (extract low-level-design + lld-typescript) runs alongside as validating content

Phase 2 (TUI, deferred):
  C7 (composition screen)

Phase 3 (deferred):
  C8 (stack-aware filtering)
```

---

## Task Breakdown

### C1: Composition frontmatter fields + validation

**What**: Add `contracts`/`implements`/`provides`/`stack` to frontmatter; reject abstract-and-capability-at-once.
**Where**: `internal/artifact/frontmatter.go`
**Depends on**: None
**Requirement**: CMP-01

**Done when**:

- [ ] Fields parse from YAML; unknown-to-others, agentskills-compatible.
- [ ] `Validate` errors when both `contracts` and `implements` are set.
- [ ] Tests cover abstract, capability, and the conflict.

**Verify**: `go test ./internal/artifact/...`
**Commit**: `feat(artifact): add composition frontmatter (contracts/implements/provides/stack)`

---

### C2: Carry composition fields on the artifact

**What**: Surface the fields on `artifact.Artifact`; populate them when resolving.
**Where**: `internal/artifact/artifact.go`, `internal/source/local.go`
**Depends on**: C1
**Requirement**: CMP-01

**Done when**:

- [ ] `Artifact` has `Contracts`, `Implements`, `Provides`, `Stack`.
- [ ] `LocalDirectory.read` (and thus git) populates them.
- [ ] Source test asserts an abstract and a capability resolve with their fields.

**Verify**: `go test ./internal/source/...`
**Commit**: `feat(artifact): carry composition fields through resolution`

---

### C3: `compose.Bind`

**What**: Derive compositions (bindings + unbound + shadowed) from a selected set.
**Where**: `internal/compose/compose.go`
**Depends on**: C2
**Requirement**: CMP-02, CMP-04, CMP-05

**Done when**:

- [ ] `Bind(selected)` binds each abstract contract to a providing capability.
- [ ] Unbound contracts and shadowed alternates are reported; deterministic by precedence then name.
- [ ] Tests: full binding, an unbound contract, a two-provider conflict, subset-providing capabilities (trait mixing).

**Verify**: `go test ./internal/compose/...`
**Commit**: `feat(compose): bind abstract contracts to selected capabilities`

---

### C4: Record bindings in the manifest

**What**: `Selection.Bindings` for abstract skills, set during `Apply`.
**Where**: `internal/workspace/manifest.go`, `internal/workspace/writer.go`, `internal/app/app.go`
**Depends on**: C3
**Requirement**: CMP-03

**Done when**:

- [ ] Abstract-skill selections carry `bindings: {contract: capability}` in `harness.yaml`.
- [ ] Round-trip test; ordinary selections have no bindings.

**Verify**: `go test ./internal/workspace/...`
**Commit**: `feat(workspace): record composition bindings in the manifest`

---

### C5: AGENTS.md renders contract + implementations

**What**: Render an abstract skill as its contract followed by linked implementations; flag unbound contracts.
**Where**: `internal/workspace/agentsmd.go`, `internal/assets/templates`
**Depends on**: C3
**Requirement**: CMP-06

**Done when**:

- [ ] Each selected abstract renders an Implementations block, one line per contract → capability entry link.
- [ ] Unbound contracts render a visible warning.
- [ ] Test asserts the rendered structure for a complete and an incomplete composition.

**Verify**: `go test ./internal/workspace/...`
**Commit**: `feat(workspace): render composed implementations in AGENTS.md`

---

### C6: Extract `low-level-design` + `lld-typescript` from ../cli

**What**: Author the abstract skill (agnostic contracts) and the TypeScript capability in the shared library.
**Where**: `~/.harness/skills/low-level-design`, `~/.harness/skills/lld-typescript` (content, not repo code)
**Depends on**: C1 (so the frontmatter is valid)
**Requirement**: CMP-07

**Done when**:

- [ ] `low-level-design/SKILL.md` declares contracts `[hexagonal, domain, command, query, persistence, naming]` with one agnostic reference per contract.
- [ ] `lld-typescript/SKILL.md` `implements: low-level-design`, `provides` all contracts, `stack: typescript`, with TS references adapted from `../cli` `devstation-arch`/`code-standards`.
- [ ] `harness list` shows both; selecting both composes with no unbound contract.

**Verify**: `harness list` and a save smoke; both resolve and compose.
**Commit**: (content lives under ~/.harness; documented, not committed to the repo)

---

### C7: Interactive composition screen (DEFERRED — needs TUI committed)

**What**: On selecting an abstract skill, a screen to pick a capability per contract.
**Where**: `internal/tui`
**Depends on**: C3; the in-flight TUI work committed
**Requirement**: CMP-08

---

### C8: Stack-aware filtering (DEFERRED)

**What**: Project stack declaration/detection; filter and auto-bind by stack.
**Requirement**: CMP-09

---

## Coverage Check

CMP-01→C1/C2 · CMP-02→C3 · CMP-03→C4 · CMP-04→C3/C5 · CMP-05→C3 · CMP-06→C5 · CMP-07→C6 · CMP-08→C7 (deferred) · CMP-09→C8 (deferred). Phase 1 = C1–C6.
