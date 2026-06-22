---
name: spec-kit
description: Run spec-driven development inside a project — turn an idea into a structured spec, plan and tasks before implementing. Use when starting a non-trivial feature, when the user asks for a spec/plan/tasks, or when planning work that touches multiple files. Specs live in .agents/specs/.
metadata:
  author: harness
  version: "0.1"
  status: scaffold
---

# Spec-driven development

> Scaffold (v0.1). The exact phases and templates will be refined; this skill
> establishes the convention so it can evolve without moving anything.

Specs formalize a change before code is written: they reduce ambiguity, break
work into tasks, and create traceability between requirement, design and
implementation. Specs are **local to the project** — they version with the code
they describe.

## Location

Every spec is a directory under the project:

```
.agents/specs/<spec-id>/
├── spec.md     # what & why: problem, requirements, acceptance criteria
├── plan.md     # how: technical design, affected components, decisions
└── tasks.md    # executable breakdown: ordered, verifiable tasks
```

`<spec-id>` is a short, ordered, kebab-case identifier, e.g.
`001-create-order` or `2026-06-payments-obfuscation`.

## Flow

1. **Specify** — write `spec.md`: the problem, requirements/user stories and
   acceptance criteria. Do not describe implementation here.
2. **Plan** — write `plan.md`: the technical approach, components touched, and
   the decisions taken. Reference any rules and skills that apply.
3. **Tasks** — write `tasks.md`: an ordered, independently verifiable task list.
4. **Implement** — work task by task, keeping the spec and code in sync.

## Conventions

- One concern per spec. Cross-cutting initiatives get their own spec that links
  to the affected feature specs.
- `spec.md` answers *what/why*; `plan.md` answers *how*. Keep them separate.
- Acceptance criteria must be checkable.

See `references/templates.md` for starting templates for each document.
