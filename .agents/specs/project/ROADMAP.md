# harness — Roadmap

Milestones are ordered by dependency, not date. Each feature gets a folder under `.agents/specs/features/`.

## Shipped

- **M0 — Artifact catalog (local)** ✅ Shared (`~/.harness`) + project (`.agents/`) merge with local override; `AGENTS.md` + `harness.yaml` generation; full-screen selection TUI; invalid-artifact diagnostics.
- **M1 — Multi-source artifact management** ✅ (`features/multi-source-artifacts/`) `Source` port (the N-source generalization); git sources via the system git client (public/private); `source add/list/remove`; reproducible vendor + `harness.lock`; offline manifest index with `update`/`search`; `upgrade`. The `apt`/`tap`/`krew` foundation. Deferred: a `verify`/frozen-apply hash check (D9).

## Now

- **M2 — Multi-target emitters** ← next to specify. Emit `CLAUDE.md`, `.github/copilot-instructions.md`, `.cursor/rules` alongside `AGENTS.md`, driven by a `targets:` field in the manifest. Required because Claude Code does not read `AGENTS.md`. Open decision: build emitters in-house vs. delegate distribution to Ruler.

## Next (not yet specified)
- **M3 — MCP as a first-class artifact kind** — describe/curate MCP servers with usage guidelines; write native client config (reference, not runtime — delegate running to Docker/Smithery).
- **M4 — Artifact composition graph** — `requires:` / `produces:` frontmatter; topological resolution; "skills that build skills".
- **M5 — Cumulative knowledge base** — token-efficient, indexed memory subsystem (atomic-fact markdown vault + progressive-disclosure recall).
- **M6 — npm and OCI source adapters** — consume the existing npm-for-skills ecosystem as one backend among many.

## Out of current scope

Hosting a public registry; a web marketplace; running MCP servers; team/multi-user sync.
