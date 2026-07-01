# Concepts & patterns

How harness models artifacts, resolves them across locations, and composes them
into a project's `AGENTS.md`. For the command-by-command reference see
[cli.md](cli.md).

## Artifact kinds

An **artifact** is a directory with a frontmatter entry document. There are
four kinds, sharing one on-disk convention (adapted from
[Agent Skills](https://agentskills.io)):

| Kind  | Container | Entry file | Role in `AGENTS.md` |
| ----- | --------- | ---------- | ------------------- |
| rule  | `rules/`  | `RULE.md`  | Invariant — **load ALWAYS** |
| skill | `skills/` | `SKILL.md` | Capability — **load on NEED** |
| agent | `agents/` | `AGENT.md` | Executor — **delegate on NEED** |
| mcp   | `mcps/`   | `MCP.md`   | Tool-server integration — **set up on NEED** |

```
<container>/<name>/
├── <ENTRY>.md        # YAML frontmatter (name, description, …) + instructions
├── scripts/          # optional
├── references/       # optional
└── assets/           # optional
```

Frontmatter requires `name` (1–64 chars, lowercase alphanumerics + single
hyphens, matching the directory name) and `description`. Optional fields:
`metadata`, `license`, `compatibility`, `allowed-tools`, and the composition
fields (`contracts`, `implements`, `provides`, `stack`, `multiple`).

An **MCP** artifact documents how to wire an external [Model Context
Protocol](https://modelcontextprotocol.io) server into the coding agents that
support it. The repetitive part — writing each agent's config file — belongs in
deterministic `scripts/`; the `MCP.md` instructs the agent to run the right
script and walk the user through any interactive step (OAuth, token creation).

## Locations & precedence

The same convention is used in three places. When the same `kind` + `name`
exists in more than one, **the highest-precedence one wins** and the others are
marked *shadowed*:

| Precedence | Location | Source tag | Notes |
| ---------- | -------- | ---------- | ----- |
| 1 (highest) | `<project>/.agents/` | `local` | project-local artifacts |
| 2 | `~/.harness/` | `shared` | your personal library, reused across projects |
| 3 | configured git sources | `<source-name>` | in the order listed in `sources.yaml` |

So a project-local `skills/cqrs` overrides a `shared` one of the same name, and
a `shared` one overrides a git source's. In `harness list`, an overriding local
artifact is shown as `local (override)`. The merged, de-duplicated catalog is
sorted by kind then name.

`HARNESS_HOME` overrides the shared-library location (default `~/.harness`) —
useful for tests or keeping the library outside `$HOME`.

## The manifest — `harness.yaml`

The project's source of truth, written at the project root. It records exactly
what was selected, where it came from, its content digest, and any composition
bindings:

```yaml
version: 3
selections:
  - kind: rule
    name: go-code-standards
    source: local
  - kind: skill
    name: low-level-design          # an abstract skill (declares contracts)
    source: home
    bindings:                       # contract -> capabilities (a list)
      domain:
        - lld-go
      persistence:
        - lld-go
  - kind: skill
    name: lld-go                    # a vendored capability
    source: local
    digest: sha256:f4b5…            # set when the artifact is copied in (vendored)
```

- `source` is the origin (`local`, `home`, or a remote source name).
- `version` is a SemVer string or empty (unversioned).
- `digest` is the SHA-256 of the vendored copy; empty for artifacts *referenced
  in place* (not copied).
- `bindings` is recorded only for composed abstract artifacts: each contract maps
  to a list of capabilities (one for a single-select abstract, several for a
  `multiple: true` one). Manifests written before v3 stored a bare scalar per
  contract and still load.

Selections are written in canonical order (by kind, then name) so the file is
diff-stable.

## `AGENTS.md` generation

Saving regenerates `AGENTS.md` at the project root. It opens with a **loading
protocol** (rules always, skills/agents on need, specs under `.agents/specs/`),
then one table per kind. Paths are:

- **project-relative** for local artifacts (`.agents/skills/x/SKILL.md`);
- **`~/`-relative** for shared artifacts (`~/.harness/skills/x/SKILL.md`) — so a
  committed `AGENTS.md` never bakes in an absolute home path or username;
- absolute only as a last resort (a library kept outside `$HOME`).

Descriptions are escaped for Markdown table cells. `AGENTS.md` carries a
"do not edit by hand" banner — it is derived state; edit artifacts and re-run
`harness` (or `harness apply`).

## Composition — abstract artifacts & capabilities

Composition lets a single, technology-agnostic artifact be fulfilled by
stack-specific implementations. It works for any kind; skills use it for
language stacks, MCPs use it for target agents.

- An **abstract artifact** declares the `contracts` it needs (e.g.
  `low-level-design` with `domain`, `persistence`, `naming`, …).
- A **capability** declares `implements: <abstract>` and the `provides:
  [contract, …]` it covers, optionally tagged with a `stack` (e.g. `lld-go`,
  `stack: go`). An artifact is *either* abstract *or* a capability, never both,
  and a capability shares the abstract's kind.
- A **binding** maps each contract to a chosen capability. Bindings are
  **explicit** — made in the selection TUI's compose wizard, recorded in
  `harness.yaml`, and rendered verbatim into the `AGENTS.md` "Composed designs"
  section. They are never re-derived behind your back; an unbound contract
  stays unimplemented.
- **Single vs. multiple.** By default a contract binds **one** capability (a
  radio choice — pick the Go *or* the TypeScript implementation). An abstract
  that sets `multiple: true` lets each contract bind **several** capabilities at
  once (checkboxes), for cases like an MCP enabled for Claude Code *and* Codex
  simultaneously. In the manifest, bindings are recorded as a list per contract.

In `AGENTS.md`, the abstract and its chosen capabilities are pulled out of the
flat tables and shown together: load the contract first, then the bound
implementation(s) per concern.

## Localization (vendoring)

By default, shared and remote artifacts are **referenced in place** — not
copied — so an empty `.agents/` never clutters a project. **Localizing** an
artifact copies it into the project so it is self-contained and committable:

- In the TUI, press `v` on a shared/remote artifact; or run
  `harness vendor <kind>/<name>`.
- The artifact's directory is copied to `.agents/<container>/<name>/`, its source
  becomes `local`, and a content **digest** (SHA-256 over the copied tree, with
  CRLF normalized for cross-platform stability) is recorded in the manifest.
- Localizing an **abstract** also localizes every capability bound to it, so the
  whole composition is complete for anyone who clones the repo
  (`expandLocalized`).

Remote (git-source) selections are always vendored on save; shared and local
selections are vendored only when you localize them.

## Sources

Beyond the shared library, you can register **git sources** of artifacts:

- `harness source add <git-url> [--name NAME] [--ref REF]` clones into
  `~/.harness/sources/<name>/` and appends to `~/.harness/sources.yaml`.
  Credentials are never stored — git auth is delegated to your system `git`.
- A source resolves artifacts by the same convention scan (`skills/`, `rules/`,
  `agents/`), **or**, if the repo ships a `harness.artifacts.yaml` index at its
  root, from that index (authoritative, with versions and explicit paths).
- `harness update` fetches every source and rebuilds the offline **search
  index** under `~/.harness/index/<source>.yaml`.
- `harness search <query>` matches name/description across the index, fully
  offline (it builds the index on demand if missing).

## Lifecycle commands

- `harness` / `select` — interactive selection → vendor → write `harness.yaml`
  + `AGENTS.md`.
- `harness apply` — reconcile a project from its committed `harness.yaml`
  without the TUI (restore missing vendored copies, verify digests, regenerate
  `AGENTS.md`). Offline; run `update` first if a source was never fetched.
- `harness upgrade` — re-resolve remote selections against current source refs,
  re-vendor changed content, bump versions/digests (bindings preserved).

See [cli.md](cli.md) for full details on every command.
