# harness — Roadmap

Milestones are ordered by dependency, not date. Each feature gets a folder under `.agents/specs/features/`.

## Shipped

- **M0 — Artifact catalog (local)** ✅ Shared (`~/.harness`) + project (`.agents/`) merge with local override; `AGENTS.md` + `harness.yaml` generation; full-screen selection TUI; invalid-artifact diagnostics.

## Now

- **M1 — Multi-source artifact management** ← `features/multi-source-artifacts/`
  Generalize the two-source merge into an N-source model with a `Source` port. Add git-repository sources (public/private via the system git client), reproducible vendor + `harness.lock`, an offline manifest index, and `search`. This is the `apt`/`tap`/`krew` foundation.

## Next (not yet specified)

- **M2 — Multi-target emitters** — emit `CLAUDE.md`, `.github/copilot-instructions.md`, `.cursor/rules` alongside `AGENTS.md`, driven by a `targets:` field in the manifest. (Required because Claude Code does not read `AGENTS.md`.)
- **M3 — MCP as a first-class artifact kind** — describe/curate MCP servers with usage guidelines; write native client config (reference, not runtime — delegate running to Docker/Smithery).
- **M4 — Artifact composition graph** — `requires:` / `produces:` frontmatter; topological resolution; "skills that build skills".
- **M5 — Cumulative knowledge base** — token-efficient, indexed memory subsystem (atomic-fact markdown vault + progressive-disclosure recall).
- **M6 — npm and OCI source adapters** — consume the existing npm-for-skills ecosystem as one backend among many.

## Out of current scope

Hosting a public registry; a web marketplace; running MCP servers; team/multi-user sync.
