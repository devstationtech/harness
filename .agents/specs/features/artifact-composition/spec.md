# Artifact Composition Specification

## Problem Statement

A developer's low-level design is the same shape across many projects (hexagonal, CQS, domain/command/query/handler/persistence) but the concrete code differs by stack (TypeScript, Go, PHP). Today each skill is monolithic and stack-bound, so the knowledge is duplicated and cannot be reused or recombined. We want an **abstract skill** that defines language-agnostic *contracts* (concepts, layer rules, directory and naming conventions) and **capabilities** that *implement* those contracts per stack, composed per project — like an interface that extends segregated interfaces, each fulfilled by a trait, mixed and matched. Composition must reuse shared capabilities (no repetition per project), allow local overrides via the existing source precedence, and render a project's `AGENTS.md` as the contract followed by links to the chosen implementations.

## Goals

- [ ] An abstract skill declares the `contracts` it needs implemented; a capability declares which abstract it `implements`, the `contracts` it `provides`, and its `stack`.
- [ ] harness composes a project's selection: it binds each abstract contract to a selected capability that provides it, and reports any unbound contract (incomplete composition).
- [ ] Capabilities are shared and reusable; a project's `harness.yaml` records only the bindings (and any local override wins by existing precedence).
- [ ] `AGENTS.md` renders an abstract skill as its contract plus an "Implementations" section linking the bound capabilities, with unbound contracts flagged.
- [ ] Validate the model by extracting a real `low-level-design` abstract skill and a `lld-typescript` capability from the `../cli` architecture skills.

## Out of Scope

| Feature | Reason |
| ------- | ------ |
| Interactive composition screen in the TUI | Phase 2 — depends on the in-flight TUI work being committed first; Phase 1 derives bindings from the flat selection. |
| Automatic project stack detection | Phase 3 — Phase 1 treats `stack` as a capability label only (no project-side detection). |
| Version-range resolution between abstract and capability | Uses the existing identity + precedence model; ranges remain a registry-era concern. |
| Transitive multi-level contract graphs (capabilities requiring capabilities) | One level (abstract → capability) for now; deeper graphs are a later milestone. |

---

## User Stories

### P1: Compose an abstract skill from selected capabilities ⭐ MVP

**User Story**: As a developer, I want to select an abstract skill and the capabilities that implement its contracts, so that my `AGENTS.md` presents one agnostic contract with the concrete, stack-specific implementations linked beneath it — reusing shared capabilities across projects.

**Why P1**: This is the whole value — composition and reuse — and it works with the existing flat selection UI, no TUI change.

**Acceptance Criteria**:

1. WHEN an artifact's frontmatter declares `contracts: [...]` THEN harness SHALL treat it as an abstract skill requiring those contracts.
2. WHEN an artifact declares `implements: <name>` and `provides: [...]` THEN harness SHALL treat it as a capability of that abstract for those contracts, carrying its `stack` label.
3. WHEN both an abstract skill and capabilities implementing it are selected THEN harness SHALL bind each contract to a providing capability and record the bindings in `harness.yaml`.
4. WHEN a contract has no selected provider THEN harness SHALL flag the composition as incomplete (a warning naming the unbound contract), without failing the save.
5. WHEN two selected capabilities provide the same contract THEN harness SHALL bind deterministically (highest source precedence, then name) and report the alternative as shadowed.
6. WHEN a local capability overrides a shared one of the same identity THEN the binding SHALL resolve to the local one by the existing precedence (no special handling).

**Independent Test**: Author a tiny abstract skill with two contracts and one capability providing both; select both; save; `harness.yaml` records the bindings and `AGENTS.md` shows the contract with the capability linked. Omit the capability and the save warns about both contracts being unbound.

---

### P2: Interactive composition screen

**User Story**: As a developer, I want, on selecting an abstract skill, a screen that lets me pick a capability per contract (pre-selected from my manifest, filtered to relevant ones), so that composing is guided rather than requiring me to find capabilities in the flat list.

**Why P2**: A usability layer over P1; the model already works without it. Deferred until the in-flight TUI changes are committed.

**Acceptance Criteria**:

1. WHEN a selected skill declares contracts THEN pressing enter SHALL open a composition screen listing each contract with its candidate capabilities.
2. WHEN the project already has bindings THEN they SHALL appear pre-selected.
3. WHEN a contract is left unbound THEN the screen SHALL show an incomplete-composition indicator.

**Independent Test**: Deferred to Phase 2 implementation.

---

### P3: Stack-aware composition

**User Story**: As a developer, I want capabilities filtered and pre-selected by my project's stack, so that I rarely have to choose manually.

**Why P3**: Convenience on top of P1/P2.

**Acceptance Criteria**:

1. WHEN a project declares (or harness detects) a stack THEN candidate capabilities SHALL be filtered to that stack, and a single match SHALL be auto-bound.

**Independent Test**: Deferred to Phase 3.

---

## Edge Cases

- WHEN a capability's `implements` names an abstract that is not selected (or does not exist) THEN it is treated as an ordinary skill (no composition), not an error.
- WHEN `provides` lists a contract the abstract does not declare THEN harness SHALL ignore the extra contract and may note it.
- WHEN an abstract skill is selected with no capabilities THEN all its contracts are unbound (incomplete) and `AGENTS.md` renders the contract alone with the warning.
- WHEN the same artifact declares both `contracts` and `implements` THEN it is rejected as an issue (an artifact is either an abstract or a capability, not both).

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| CMP-01 | P1: abstract/capability frontmatter model | Design | Pending |
| CMP-02 | P1: compose + bind contracts to capabilities | Design | Pending |
| CMP-03 | P1: record bindings in harness.yaml | Design | Pending |
| CMP-04 | P1: flag unbound contracts (incomplete) | Design | Pending |
| CMP-05 | P1: deterministic bind + shadow report on conflict | Design | Pending |
| CMP-06 | P1: AGENTS.md renders contract + linked implementations | Design | Pending |
| CMP-07 | P1: extract low-level-design + lld-typescript from ../cli | Design | Pending |
| CMP-08 | P2: interactive composition screen (deferred) | Design | Pending |
| CMP-09 | P3: stack-aware filtering (deferred) | Design | Pending |

**ID format:** `CMP-[NUMBER]`

---

## Success Criteria

- [ ] Selecting `low-level-design` + `lld-typescript` renders an `AGENTS.md` with the agnostic contract and the TypeScript implementation links, no contract unbound.
- [ ] The same `low-level-design` is reused across projects from the shared library with no duplication; a project-local capability overrides a shared one transparently.
- [ ] An incomplete composition (a contract with no provider) is clearly reported, not silently dropped.
