# Source Index and Versioning Specification

## Problem Statement

Artifacts have no versioning today: `version` is, at best, free-form text in `metadata` that harness ignores, and the agentskills.io standard does not require it. Reproducibility rests only on a git commit/content hash recorded in a separate `harness.lock`. We want package-level SemVer that works across heterogeneous sources (git and npm) without forcing a repository topology, and a single declarative project manifest at the project root — folding the lock's provenance and integrity into it.

## Goals

- [ ] A git source can declare a package manifest, `harness.artifacts.yaml`, that versions each artifact and points to its directory — making monorepo vs repo-per-artifact a free choice for the source author.
- [ ] Every resolved artifact carries a validated SemVer `Version` (or is explicitly "unversioned"), surfaced in `list`/`search` and used by `upgrade` to report transitions.
- [ ] The project manifest moves to the project root (`harness.yaml`) and records `source + version + digest` per selection; the separate `harness.lock` is retired.
- [ ] A project can be reconciled from its committed root manifest without the TUI (`harness apply`), which also verifies vendored content against the recorded digest.

## Out of Scope

| Feature | Reason |
| ------- | ------ |
| Version range/constraint resolution (`^1.2.0`) | Needs a registry; deferred to the npm/registry milestone. Resolution stays by git ref. |
| Full SemVer ordering/comparison | Only validation and equality are needed now; ordering arrives with ranges. |
| npm source adapter implementation | Deferred; this feature defines how versioning maps to it (read `package.json`) but does not build it. |
| Hosted registry service | A git index repo is the registry; no service is built. |
| Per-path git tags for versions | The index carries versions; tags are not required. |

---

## User Stories

### P1: Versioned artifacts via a source package manifest ⭐ MVP

**User Story**: As a source author, I want to declare `harness.artifacts.yaml` at my repository root that versions and locates each artifact, so that I can ship a monorepo or a repo-per-artifact freely and consumers get real SemVer.

**Why P1**: Versioning is the core ask and the index is what makes it topology-agnostic; everything else records or surfaces it.

**Acceptance Criteria**:

1. WHEN a git source's clone contains `harness.artifacts.yaml` THEN the system SHALL resolve exactly the artifacts it lists, reading each from its declared `path` and stamping it with the declared `version`.
2. WHEN an index entry's `version` is present THEN the system SHALL validate it as SemVer (`MAJOR.MINOR.PATCH`, optional pre-release/build) and SHALL report an invalid version as an issue (the entry is skipped, not silently accepted).
3. WHEN an index entry's `version` is absent THEN the system SHALL load the artifact as "unversioned" rather than failing.
4. WHEN a source has no `harness.artifacts.yaml` THEN the system SHALL fall back to scanning the conventional `skills/`/`rules/`/`agents/` containers (today's behavior), resolving artifacts as unversioned.
5. WHEN an index entry points to a missing path, an entry document that fails to load, or a name that does not match the entry document THEN the system SHALL report it as an issue and continue with the rest.

**Independent Test**: Build a git fixture with a `harness.artifacts.yaml` listing two artifacts in non-conventional paths with versions; `source add` it and `list`/`search` show both with their versions. A second fixture without an index resolves by convention as unversioned.

---

### P2: Root project manifest, lock retired

**User Story**: As a developer, I want one declarative manifest at my project root that records what I selected, from which source, at which version and digest, so that the project's harness state is visible, committable, and self-contained.

**Why P2**: Consolidates the project's record; removes the manifest/lock split that is unjustified without version ranges.

**Acceptance Criteria**:

1. WHEN a selection is saved THEN the system SHALL write `harness.yaml` at the **project root** with, per selection: kind, name, source, version, and (for vendored remote artifacts) a content digest.
2. WHEN the manifest is written THEN the system SHALL NOT write a separate `harness.lock`, and SHALL remove a stale one if present.
3. WHEN harness re-reads a configured project THEN it SHALL pre-select from the root manifest exactly as before.
4. WHEN a selection is local or shared (referenced in place) THEN its manifest entry SHALL omit the digest (only vendored remotes are digested).

**Independent Test**: Select a remote and a local artifact, save; `harness.yaml` exists at the project root (not under `.agents/`), carries versions and a digest only for the remote one, and no `harness.lock` exists.

---

### P3: Reconcile from the committed manifest (`harness apply`)

**User Story**: As a developer (or a teammate cloning the repo), I want `harness apply` to materialize the project from its committed `harness.yaml` without the TUI, so that the manifest is the source of truth and content is verifiable.

**Why P3**: Completes the package-manager model (`npm install` equivalent) and gives the integrity check the lock used to promise.

**Acceptance Criteria**:

1. WHEN `harness apply` runs THEN the system SHALL resolve every manifest selection from its source, re-vendor remote ones, and regenerate `AGENTS.md`.
2. WHEN a vendored artifact already on disk matches its recorded digest THEN apply SHALL leave it untouched; WHEN it differs THEN apply SHALL report the drift.
3. WHEN a selection's source is not configured or the artifact is gone THEN apply SHALL report it and continue.

**Independent Test**: Save a project, delete `.agents/<vendored>`, run `harness apply`, and the artifact is restored matching its digest.

---

## Edge Cases

- WHEN `harness.artifacts.yaml` is malformed YAML THEN the system SHALL report a clear error naming the file, not crash.
- WHEN two index entries share a kind and name THEN the later one SHALL be reported as a duplicate issue.
- WHEN an index `path` escapes the repository (e.g. `../`) THEN the system SHALL reject it as an issue.
- WHEN both an index and conventional directories exist THEN the index is authoritative (no double counting).
- WHEN a version is present in an artifact's frontmatter but the index omits it THEN the artifact is unversioned (the index, not the frontmatter, is authoritative for version).

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| SIV-01 | P1: index-driven resolution with version | Design | Pending |
| SIV-02 | P1: SemVer validation + issue on invalid | Design | Pending |
| SIV-03 | P1: unversioned when version absent | Design | Pending |
| SIV-04 | P1: convention fallback when no index | Design | Pending |
| SIV-05 | P1: invalid entry (path/name/load) → issue | Design | Pending |
| SIV-06 | P2: root manifest with source/version/digest | Design | Pending |
| SIV-07 | P2: retire harness.lock | Design | Pending |
| SIV-08 | P3: `harness apply` reconcile + digest verify | Design | Pending |

**ID format:** `SIV-[NUMBER]`
**Status values:** Pending → In Design → In Tasks → Implementing → Verified

---

## Success Criteria

- [ ] A source author ships a monorepo with `harness.artifacts.yaml`; a consumer sees per-artifact SemVer in `search` and `upgrade` reports `name 1.2.0 → 1.3.0`.
- [ ] A plain agentskills repo with no index still works (unversioned), unchanged.
- [ ] The project root holds `harness.yaml`; `.agents/` holds only content (artifacts + specs); no `harness.lock` anywhere.
- [ ] `harness apply` restores a deleted vendored artifact to match its recorded digest.
