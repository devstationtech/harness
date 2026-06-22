# Spec document templates

Starting templates for each document under `.agents/specs/<spec-id>/`. Adapt
sections as needed; remove what does not apply.

## spec.md

```markdown
# <spec-id>: <title>

## Problem
<What is wrong or missing, and why it matters.>

## Requirements
- <Requirement or user story>

## Acceptance criteria
- [ ] <Checkable outcome>

## Out of scope
- <Explicitly excluded items>
```

## plan.md

```markdown
# Plan — <spec-id>

## Approach
<The technical approach in prose.>

## Affected components
- <Module / package / file and the change>

## Decisions
- <Decision and rationale>

## Rules & skills in play
- <Rules that constrain this work; skills that guide it>
```

## tasks.md

```markdown
# Tasks — <spec-id>

- [ ] T1 — <task> (verify: <how to confirm it is done>)
- [ ] T2 — <task> (verify: ...)
```
