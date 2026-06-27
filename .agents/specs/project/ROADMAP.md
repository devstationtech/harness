# harness — Roadmap

Milestones are ordered by dependency, not date. Each feature gets a folder under `.agents/specs/features/`.

## Shipped

- **M0 — Artifact catalog (local)** ✅ Shared (`~/.harness`) + project (`.agents/`) merge with local override; `AGENTS.md` + `harness.yaml` generation; full-screen selection TUI; invalid-artifact diagnostics.
- **M1 — Multi-source artifact management** ✅ (`features/multi-source-artifacts/`) `Source` port (the N-source generalization); git sources via the system git client (public/private); `source add/list/remove`; reproducible vendor; offline manifest index with `update`/`search`; `upgrade`. The `apt`/`tap`/`krew` foundation.
- **M2 — Source index & versioning** ✅ (`features/source-index-and-versioning/`) Package-level SemVer via a source `harness.artifacts.yaml` (monorepo or repo-per-artifact, author's choice; convention fallback when absent); version surfaced in list/search/upgrade; project manifest moved to the root carrying source+version+digest; `harness.lock` retired; `harness apply` reconciles + verifies from the committed manifest. (Subsumes the old D9 verify idea.)

## Now

- **M3 — Artifact composition** ← `features/artifact-composition/` (specified). Abstract skills declare language-agnostic *contracts*; *capabilities* implement them per stack (`implements`/`provides`/`stack`); harness composes a project's selection, records bindings in `harness.yaml`, and renders `AGENTS.md` as the contract plus linked implementations. Shared capabilities reused across projects; local overrides win by precedence. Validated by extracting `low-level-design` + `lld-typescript` from `../cli`. (Was the old M4/M5 composition idea.) Phase 1 (model + render) lands first; the interactive composition screen and stack filtering follow.

## Next (not yet specified)

- **M4 — Multi-target emitters** — emit `CLAUDE.md`, `.github/copilot-instructions.md`, `.cursor/rules` alongside `AGENTS.md`, driven by a `targets:` field in the manifest. Required because Claude Code does not read `AGENTS.md`. Open decision: build emitters in-house vs. delegate distribution to Ruler.
- **M4 — MCP as a first-class artifact kind** — describe/curate MCP servers with usage guidelines; write native client config (reference, not runtime — delegate running to Docker/Smithery).
- **M5 — Artifact composition graph** — `requires:` / `produces:` frontmatter; topological resolution; "skills that build skills".
- **M6 — Cumulative knowledge base** — token-efficient, indexed memory subsystem (atomic-fact markdown vault + progressive-disclosure recall).
- **M7 — npm and OCI source adapters** — consume the existing npm-for-skills ecosystem as one backend among many; the npm adapter reads `package.json` for versions. Version *range* resolution and a git index-source ("registry without a service", Krew model) land here.

## Out of current scope

Hosting a public registry; a web marketplace; running MCP servers; team/multi-user sync.
