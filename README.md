# harness

Configure AI-agent harness artifacts — **rules**, **skills** and **agents** —
across your projects from a single shared library.

`harness` merges a personal library in your home (`~/.harness`) with the
project-local artifacts in `.agents/`, lets you pick what each project needs in a
small TUI, and generates an `AGENTS.md` that tells agents *what to always load*
and *what to load only when needed*.

> Status: early pilot. Single-user / local sharing between your own projects.
> A future release will add remote artifact repositories and centralized
> install.

## Concepts

Three artifact kinds, one on-disk convention (adapted from
[Agent Skills](https://agentskills.io)):

| Kind  | Container | Entry file | Role in `AGENTS.md` |
| ----- | --------- | ---------- | ------------------- |
| rule  | `rules/`  | `RULE.md`  | Invariant — **load ALWAYS** |
| skill | `skills/` | `SKILL.md` | Capability — **load on NEED** |
| agent | `agents/` | `AGENT.md` | Executor — **delegate on NEED** |

```
<container>/<name>/
├── <ENTRY>.md        # frontmatter (name, description, …) + instructions
├── scripts/          # optional
├── references/       # optional
└── assets/           # optional
```

Artifacts live in two places with the **same structure**:

- `~/.harness/` — your shared library, reused across projects (`shared`).
- `<project>/.agents/` — local to one repository (`local`). A local artifact
  with the same name overrides the shared one.

## Install

```sh
go install github.com/devstationtech/harness@latest
```

Or build from source:

```sh
go build -o harness .
```

## Usage

```sh
harness init     # create & seed your shared library (~/.harness)
harness          # pick artifacts for the current project (interactive)
harness list     # print the merged catalog as text
harness help     # show all commands
```

Running `harness` inside a project shows the merged catalog grouped by kind:

```
Rules · load ALWAYS  (1/1)
  › [x] hexagonal-architecture  shared   Example invariant — the domain layer …

Skills · load on NEED  (1/2)
    [x] skill-creator           shared   Create a new harness artifact …
    [ ] spec-kit                shared   Run spec-driven development …
```

Saving generates, at the project root and under `.agents/`:

- `AGENTS.md` — the agent entry point, with one table per kind and the loading
  protocol.
- `.agents/harness.yaml` — the manifest of active artifacts (source of truth).
- `.agents/{rules,skills,agents,specs}/` — ready for project-local artifacts.

Shared artifacts are **referenced in place**, not copied.

### Keys

| Key | Action |
| --- | ------ |
| `↑`/`↓` (`k`/`j`) | move |
| `space` (`x`) | toggle |
| `a` | toggle whole section |
| `enter` | save |
| `q` / `esc` | quit without saving |

## Seeded artifacts

`harness init` seeds your library with:

- `skill-creator` — how to author harness artifacts in the adopted convention.
- `spec-kit` — spec-driven development; specs live in `.agents/specs/`.
- `hexagonal-architecture` (example rule) and `code-reviewer` (example agent).

Configuration:

- `HARNESS_HOME` overrides the shared library location (default `~/.harness`).

## License

TBD.
