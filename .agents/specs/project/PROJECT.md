# harness — Project Vision

## What

`harness` is a global, cross-project CLI that manages AI-agent harness artifacts — **rules**, **skills**, and **agents** (and, ahead, MCP configurations) — and composes them into per-project agent instructions (`AGENTS.md` and per-agent equivalents).

## Why

A single developer works across many repositories with many coding agents (Claude Code, Codex, Copilot, Cursor, ...). Their working style, standards, and reusable procedures live in their head or scattered in dotfiles. `harness` turns that into a **curated, cumulative, reusable artifact library** that can be selected per project — so new projects gain traction fast, following the same quality bar, without re-explaining context to each agent.

## Positioning (decided after market discovery)

The "sync one rule-set to many agents" lane is already owned by mature tools (Ruler, rulesync, block/ai-rules). `harness` does **not** compete there. Its defensible value is the layer above:

1. **Composition** — artifacts that reference and shape other artifacts (skills that build skills); a dependency-aware library, not a flat rule dump.
2. **Multi-source curation** — a personal library plus external/private repositories, unified under one manager with search (the `apt` / `Homebrew tap` / `Krew` model), selected per project.
3. **Cumulative knowledge** — a token-efficient, indexed knowledge base as a first-class concern (progressive disclosure, not RAG-by-default).

## Principles

- **Follow consolidated patterns; do not reinvent.** Artifact format = Agent Skills (agentskills.io). Distribution = git-repo sources + local index + lockfile (apt/brew/krew). Auth = delegate to the system git client.
- **Stack-agnostic output, multi-target.** `AGENTS.md` is canonical, but Claude Code reads `CLAUDE.md` — emit per agent.
- **Cross-platform (macOS, Linux, Windows).** No shell interpolation, no hardcoded paths, line-ending-stable hashing.
- **Reproducible.** Anything resolved from a remote source is vendored and locked by content hash.
- **OSS-distributable.** Personal knowledge base lives outside this repo.

## Tech

Go (module `github.com/devstationtech/harness`), Bubble Tea + Lipgloss TUI, hexagonal layering (ports/adapters under `internal/`).
