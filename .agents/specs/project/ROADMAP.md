# harness — Roadmap

Milestones are ordered by dependency, not date. Each feature gets a folder under `.agents/specs/features/`.

## Shipped

- **M0 — Artifact catalog (local)** ✅ Shared (`~/.harness`) + project (`.agents/`) merge with local override; `AGENTS.md` + `harness.yaml` generation; full-screen selection TUI; invalid-artifact diagnostics.
- **M1 — Multi-source artifact management** ✅ (`features/multi-source-artifacts/`) `Source` port (the N-source generalization); git sources via the system git client (public/private); `source add/list/remove`; reproducible vendor; offline manifest index with `update`/`search`; `upgrade`. The `apt`/`tap`/`krew` foundation.
- **M2 — Source index & versioning** ✅ (`features/source-index-and-versioning/`) Package-level SemVer via a source `harness.artifacts.yaml` (monorepo or repo-per-artifact, author's choice; convention fallback when absent); version surfaced in list/search/upgrade; project manifest moved to the root carrying source+version+digest; `harness.lock` retired; `harness apply` reconciles + verifies from the committed manifest. (Subsumes the old D9 verify idea.)
- **M3 — Artifact composition** ✅ (`features/artifact-composition/`) Abstract artifacts declare language-agnostic *contracts*; *capabilities* implement them per stack (`implements`/`provides`/`stack`); the user binds each contract in the compose wizard, recorded in `harness.yaml` (schema v3, per-contract lists), and `AGENTS.md` renders the contract plus linked implementations. Shipped **beyond spec**: composition works for **any kind**, an abstract may set `multiple: true` (a contract binds several capabilities — the MCP-per-agent case), and the interactive screen landed. Stack-aware filtering deferred.
- **M4 — MCP as a first-class artifact kind** ✅ Kind `mcp` (`mcps/`, `MCP.md`): curate MCP servers with usage/setup instructions and deterministic setup scripts; native client config written by the agent's own CLI (reference, not runtime). Composes via `multiple` (enable several agents at once); renders in a dedicated AGENTS.md "MCP servers" section.

## Now

- (Between milestones — pick the next from below.)

## Next (not yet specified)

- **M5 — Multi-target emitters** — emit `CLAUDE.md`, `.github/copilot-instructions.md`, `.cursor/rules` alongside `AGENTS.md`, driven by a `targets:` field in the manifest. Required because Claude Code does not read `AGENTS.md`. Open decision: build emitters in-house vs. delegate distribution to Ruler.
- **M6 — Artifact composition graph** — `requires:` / `produces:` frontmatter; topological resolution; "skills that build skills".
- **M7 — Cumulative knowledge base** — token-efficient, indexed memory subsystem (atomic-fact markdown vault + progressive-disclosure recall).
- **M8 — npm and OCI source adapters** — consume the existing npm-for-skills ecosystem as one backend among many; the npm adapter reads `package.json` for versions. Version *range* resolution and a git index-source ("registry without a service", Krew model) land here.

## Out of current scope

Hosting a public registry; a web marketplace; running MCP servers; team/multi-user sync.
