# Specs index

One directory per spec. **Read this index first, then open a spec only when your
task touches it** — don't read every spec each time. Each feature has `spec.md`
(requirements), `design.md` (technical design) and `tasks.md` (breakdown).
Project-level context lives under `project/`.

Status: ✅ shipped · 🚧 in progress · 📋 planned

## Features

| Spec | Status | Summary |
| --- | --- | --- |
| [artifact-composition](features/artifact-composition/) | ✅ shipped | Abstract artifacts declare `contracts`; capabilities `implement`/`provide` them; the user binds each contract in the compose wizard. Shipped **beyond** the original spec — see its *Implementation status* note: **multi-select** (`multiple: true` binds several capabilities per contract), **any kind** (incl. `mcp`, with a dedicated AGENTS.md section), user-driven bindings (no `compose` package). Deferred: stack-aware filtering. |
| [multi-source-artifacts](features/multi-source-artifacts/) | ✅ shipped | Multiple git/local sources merged into one catalog with precedence; `source add/list/remove`, vendor, offline index, `update`, `search`, `upgrade`. **Superseded in part by M2**: the `harness.lock` / `internal/lock` design was retired — provenance + digest now live in the root `harness.yaml`. |
| [source-index-and-versioning](features/source-index-and-versioning/) | ✅ shipped | SemVer versions, `harness.artifacts.yaml` package manifest, offline per-source index, root `harness.yaml` (**schema v3**) recording source/version/digest, `harness apply` with digest verification. |

## Project

| Doc | Purpose |
| --- | --- |
| [project/PROJECT.md](project/PROJECT.md) | What harness is, scope, principles. |
| [project/ROADMAP.md](project/ROADMAP.md) | Milestones: shipped / now / next. |
| [project/STATE.md](project/STATE.md) | Decision log and current state. |

> Kept in sync by hand. If code and spec disagree, **the code wins** — update the
> spec (see each feature's *Implementation status* note). Current kinds:
> `rule`, `skill`, `agent`, `mcp`. Manifest schema: **v3** (per-contract capability
> lists; v2 scalar bindings still load).
