# Multi-Source Artifact Management Specification

> **Implementation status:** ✅ shipped, but **partly superseded by M2**
> (source-index-and-versioning). The separate `harness.lock` and `internal/lock`
> package described below were **retired**: provenance + digest now live
> per-selection in the root `harness.yaml`, and `ContentHash` moved to
> `internal/vendor`. Read every "lock"/`internal/lock` reference here as "the root
> manifest". The `Source` port also returns resolved `artifact.Artifact` (no
> `Fetch`/`Payload`), per decision D8.

## Problem Statement

Today `harness` discovers artifacts from exactly two fixed locations: the shared library (`~/.harness`) and the project (`.agents/`). The developer cannot point `harness` at external or private artifact repositories, so a personal library cannot be reused across machines, shared selectively, or grown from public catalogs. We need a unified manager that consumes many sources (local directories and git repositories, public or private) with offline search and reproducible resolution — following the consolidated `apt` / `Homebrew tap` / `Krew` model rather than inventing a format.

## Goals

- [ ] A developer can register a git repository (public or private) as an artifact source and select its artifacts into any project.
- [ ] Anything resolved from a remote source is reproducible: vendored into the project and locked by content hash, identical across machines and CI without re-fetching.
- [ ] Artifacts from all sources are searchable offline, in well under one second, without spending agent tokens.
- [ ] The existing shared+local behavior is preserved exactly, re-expressed as the N=2 case of the new source model.

## Out of Scope

| Feature | Reason |
| ------- | ------ |
| npm and OCI source adapters | Deferred to M6; the `Source` port is designed to accept them later. |
| Multi-target emitters (`CLAUDE.md`, Cursor, Copilot) | Separate milestone (M2). This feature only changes how artifacts are sourced, not how they are emitted. |
| MCP artifact kind | Separate milestone (M3). |
| Running/hosting a public registry or marketplace | Product is a client-side manager, not a registry. |
| Embedding custom authentication (token storage, SSH key management) | Delegated to the system git client by design (D5). |
| Semantic-version constraint solving across sources | A source resolves at a branch/tag/commit ref; no cross-source version solver in this milestone. |

---

## User Stories

### P1: Register and consume a git source, reproducibly ⭐ MVP

**User Story**: As a developer, I want to add a git repository as an artifact source and select its skills/rules/agents into my project, so that my curated library is reusable across projects and machines with identical, reproducible content.

**Why P1**: This is the vertical slice that delivers the entire value of the milestone — a source the user controls, consumed reproducibly. Every other story is an enhancement around it.

**Acceptance Criteria**:

1. WHEN the user runs `harness source add <git-url> [--name <name>] [--ref <ref>]` THEN the system SHALL record the source in `~/.harness/sources.yaml` and clone it into `~/.harness/sources/<name>/`.
2. WHEN a registered git source clones successfully THEN the system SHALL include its valid artifacts in the catalog, addressable as `<source>/<name>`, merged with precedence `project > home > remote sources (in configured order)`.
3. WHEN the user selects a remote artifact and saves THEN the system SHALL copy (vendor) that artifact's directory into the project's `.agents/<container>/<name>/` and record an entry in `.agents/harness.lock` with source, resolved commit, and content hash.
4. WHEN `harness apply` (or save) runs on another machine with the same `harness.lock` THEN the system SHALL produce byte-identical vendored artifacts and SHALL report a hash mismatch as an error rather than silently diverging.
5. WHEN the source is a private repository AND the user can already `git clone` it in their shell THEN the system SHALL clone it with no additional configuration (credentials handled by the system git client).
6. WHEN the `git` binary is not found on `PATH` THEN the system SHALL fail with a clear, actionable message naming `git` as the missing dependency.
7. WHEN artifacts are vendored or hashed on Windows THEN the system SHALL produce the same content hashes as on macOS/Linux (line-ending-stable).

**Independent Test**: Create a throwaway git repo containing one skill; `harness source add` it; run the selection TUI and confirm the skill appears as `<source>/skill`; select and save; verify the skill directory is copied under `.agents/` and a `harness.lock` entry exists; delete the local clone and re-apply from the lock to confirm reproducibility.

---

### P2: Refresh sources and search offline

**User Story**: As a developer, I want to refresh my sources and search all available artifacts by keyword offline, so that I can find the right artifact across many repositories without scanning each or spending tokens.

**Why P2**: Search makes a multi-source library usable at scale, but the MVP is demonstrable with the selection TUI alone.

**Acceptance Criteria**:

1. WHEN the user runs `harness update` THEN the system SHALL fetch the latest ref of each git source and rebuild a local manifest index under `~/.harness/index/`.
2. WHEN the user runs `harness search <query>` THEN the system SHALL match the query against artifact name and description across all sources and print results as `<source>/<name> | kind | description`, reading only the local index.
3. WHEN a source is unreachable during `update` THEN the system SHALL keep the previously indexed manifests for that source and report the failure without aborting the whole command.
4. WHEN no index exists yet AND the user runs `search` THEN the system SHALL build it on demand from currently available sources.

**Independent Test**: With two sources registered, run `harness update`, then `harness search review` and confirm matching artifacts from both sources are listed with their source prefix, with networking disabled.

---

### P2: Manage configured sources

**User Story**: As a developer, I want to list and remove sources, so that I can curate where my artifacts come from.

**Why P2**: Needed for real use but not for the first demo.

**Acceptance Criteria**:

1. WHEN the user runs `harness source list` THEN the system SHALL print each source's name, type, URL, and ref.
2. WHEN the user runs `harness source remove <name>` THEN the system SHALL delete it from `sources.yaml`, remove its clone and indexed manifests, and SHALL NOT alter already-vendored artifacts or `harness.lock` entries in any project.

**Independent Test**: Add two sources, `source list` shows both; `source remove` one; `source list` shows one; a previously vendored artifact from the removed source still exists in its project.

---

### P3: Upgrade locked artifacts to the latest source ref

**User Story**: As a developer, I want to re-resolve a project's locked artifacts against the latest source refs, so that I can pull updates intentionally.

**Why P3**: Convenience; until then a user can re-select in the TUI.

**Acceptance Criteria**:

1. WHEN the user runs `harness upgrade` in a project THEN the system SHALL re-resolve each locked artifact against its source's current ref, re-vendor changed artifacts, and update `harness.lock` with new commits/hashes.
2. WHEN an upgrade changes an artifact THEN the system SHALL report which artifacts changed (old → new commit).

**Independent Test**: Vendor an artifact, push a change to the source repo, run `harness upgrade`, confirm the vendored content and lock entry update and the change is reported.

---

## Edge Cases

- WHEN two sources expose an artifact of the same kind and name THEN the system SHALL apply precedence order and SHALL surface the shadowed one as overridden (consistent with today's local-over-shared behavior).
- WHEN a git source contains a directory that is not a valid artifact (frontmatter `name` ≠ folder, missing entry document) THEN the system SHALL skip it and report it via the existing invalid-artifact diagnostics, not crash.
- WHEN `harness.lock` references a commit no longer present on the source (force-push/gc) THEN the system SHALL report a clear reproducibility error on upgrade and leave the vendored content intact.
- WHEN a source URL is added twice (same resolved name) THEN the system SHALL reject the duplicate with a clear message.
- WHEN the project has no `.agents/` yet and a remote artifact is selected THEN the system SHALL create `.agents/` (and the needed container dir) on demand, consistent with current on-demand directory creation.
- WHEN a clone is interrupted/partial THEN the system SHALL not leave a corrupt source active (clone into a temp path, then rename on success).

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| SRC-01 | P1: register git source | Design | Pending |
| SRC-02 | P1: N-source catalog + precedence (refactor) | Design | Pending |
| SRC-03 | P1: vendor remote artifact into project | Design | Pending |
| SRC-04 | P1: lockfile write + reproducible apply (hash verify) | Design | Pending |
| SRC-05 | P1: private repo via system git (shell-out) | Design | Pending |
| SRC-06 | P1: git-missing + cross-platform safety | Design | Pending |
| SRC-07 | P2: `update` refresh + index build | Design | Pending |
| SRC-08 | P2: `search` offline | Design | Pending |
| SRC-09 | P2: `source list` / `source remove` | Design | Pending |
| SRC-10 | P3: `upgrade` re-resolve + report | Design | Pending |

**ID format:** `SRC-[NUMBER]`
**Status values:** Pending → In Design → In Tasks → Implementing → Verified
**Coverage:** 10 total, 0 mapped to tasks yet, 0 unmapped.

---

## Success Criteria

- [ ] A developer adds a private git source and uses one of its skills in a fresh project in under two minutes, with no auth setup beyond what `git clone` already needs.
- [ ] Deleting `~/.harness/sources/` and re-applying a project from its `harness.lock` reproduces byte-identical vendored artifacts (hash-verified) on macOS, Linux, and Windows.
- [ ] `harness search` returns results across all sources with networking disabled.
- [ ] The existing local/shared selection flow and generated `AGENTS.md` are unchanged for users who add no sources.
