---
name: skill-creator
description: Create a new harness artifact (skill, rule or agent) in the adopted on-disk convention. Use when the user wants to author, scaffold or edit a skill, rule or agent, asks how harness artifacts are structured, or needs a valid SKILL.md / RULE.md / AGENT.md.
metadata:
  author: harness
  version: "1.0"
---

# Creating harness artifacts

harness shares three kinds of artifact across projects. All three use the same
folder convention, adapted from the Agent Skills specification
(https://agentskills.io):

| Kind  | Container | Entry file | Role |
| ----- | --------- | ---------- | ---- |
| skill | `skills/` | `SKILL.md` | On-demand capability, loaded only when a task matches its description. |
| rule  | `rules/`  | `RULE.md`  | Project invariant, always loaded. |
| agent | `agents/` | `AGENT.md` | Specialized executor that work is delegated to. |

Authored artifacts live in either:

- the **shared library** `~/.harness/<container>/<name>/` (reused across projects), or
- a **project** `.agents/<container>/<name>/` (local to one repository).

## Folder layout

```
<container>/<name>/
├── <ENTRY>.md        # required: frontmatter + instructions
├── scripts/          # optional: executable helpers
├── references/       # optional: docs loaded on demand
└── assets/           # optional: templates, schemas, data
```

`<name>` must match the `name` field in the frontmatter.

## Frontmatter

```yaml
---
name: my-artifact          # required; lowercase, digits, single hyphens; == folder name
description: What it does and WHEN to use it.  # required; <= 1024 chars
metadata:                  # optional
  author: your-name
  version: "1.0"
---
```

Naming rules for `name`: 1–64 characters, lowercase `a-z0-9` and single hyphens,
no leading/trailing/consecutive hyphens.

## How to write the description

The description is the only text harness shows in the selection screen and the
only signal an agent uses to decide whether to load the artifact. Make it earn
its place:

- State **what** the artifact does and **when** to use it.
- Include concrete trigger keywords ("migration", "aggregate", "adapter").
- Avoid vague phrasing like "helps with X".

Good: `Create a CQRS command and its handler following the project convention.
Use when adding a write operation, command or use case.`

Poor: `Helps with commands.`

## How to write the body

Keep `SKILL.md` under ~500 lines / 5000 tokens. Add what the agent would get
wrong without you (project conventions, non-obvious edge cases, exact tools);
omit what it already knows. Prefer reusable procedures over one-off answers, give
a clear default instead of a menu of options, and move long reference material
into `references/`, telling the agent exactly when to load each file.

For rules, the body states the invariant and how to verify compliance. For
agents, the body defines the executor's responsibility, scope and boundaries.

## Procedure

1. Decide the kind. Invariant the project must always respect → rule. Repeatable
   capability → skill. Delegatable executor → agent.
2. Create `~/.harness/<container>/<name>/` (shared) or
   `.agents/<container>/<name>/` (project-local).
3. Write `<ENTRY>.md` with valid frontmatter and a focused body.
4. Run `harness list` to confirm it is discovered, then `harness` to select it
   into a project.
