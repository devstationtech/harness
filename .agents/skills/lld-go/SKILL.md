---
name: lld-go
description: Go implementation of the low-level-design contracts — the concepts translated to idiomatic Go, not transliterated. Package-by-feature layout instead of layer directories, consumer-defined interfaces as ports, functions as command handlers, structs with invariants only where invariants exist. Use alongside the low-level-design abstract skill when building or reviewing a Go bounded context.
metadata:
  category: architecture
  author: andrespineli
  version: "0.2.0"
implements: low-level-design
stack: go
provides:
  - hexagonal
  - domain
  - command
  - query
  - persistence
  - naming
---

# Low-Level Design — Go capability

Go implements the `low-level-design` contracts by **translation, not
transliteration**. The concepts survive as *roles*; the OO structures that
usually carry them (layer directories, handler classes, value-object wrappers
around every string) mostly do not. Where a contract's structural default and a
Go idiom collide, the idiom wins — and each section below says what honours the
underlying rule instead.

Authority order: Effective Go → Google Go Style Guide → Uber Go Style Guide →
the project's go-code-standards rule. A good worked example of this whole
translation is Ben Johnson's "Standard Package Layout" (the WTF Dial project):
hexagonal architecture in Go with no ceremony — a root package defines domain
types and ports, one package per dependency implements them, `main` wires.

## Translation table

| Contract concept | In Go it becomes |
| --- | --- |
| Bounded context | A feature package (or small group) under `internal/`; a separate module only at real scale |
| Layer (`domain/application/inbound/outbound`) | Import direction between packages, enforced by the compiler — **not** directories |
| Port | Small interface (1–3 methods) defined in the **consuming** package |
| Adapter | A package (or type) implementing that interface, with a compile-time assertion |
| Command + Handler | An exported function or method on an application type |
| Aggregate | Struct with unexported fields + behaviour methods **only where an invariant exists** |
| Value object | Named type with validating constructor **only where validation earns it**; otherwise a plain type |
| Domain event | A return value; a past-tense event struct only when multiple consumers react |
| Repository port | Consumer-defined interface with **exported** methods (`Cluster`, `ClusterByName`, `Save`) |
| Read model | Plain struct with exported fields, shaped for the caller |
| Composition root | `main` (or the single `app`/`cli` package it calls) |
| Architecture tests | The compiler (no import cycles) + `internal/` + `depguard`/`gomodguard` in golangci-lint when boundaries need teeth |

## hexagonal — layers as import direction, not directories

- Layout is **package-by-feature, flat** under `internal/`. Do not create
  `domain/`, `application/`, `inbound/`, `outbound/` directories: they multiply
  micro-packages, invite stutter and import cycles, and hide what the program
  does. A feature package holds its domain types, its ports and its
  application functions together.
- The dependency rule still binds, expressed as imports: a feature package
  imports nothing that implements its ports; adapter packages import the
  feature package, never the reverse. The compiler rejects cycles; `internal/`
  caps the exported surface; add `depguard` rules when a boundary deserves
  explicit enforcement.
- A **port** is a small interface declared in the package that needs it
  (`source.Source` is declared in `source`, consumed by `catalog`). Each
  adapter carries a compile-time check:
  `var _ source.Source = (*GitRepository)(nil)`.
- The **inbound adapter** is `main` plus whatever thin transport package parses
  input (flags, HTTP, MCP) and calls application functions. It never touches an
  outbound adapter directly.
- The **composition root** is `main` or the one package it delegates wiring to.
  It is the only place concrete adapters meet ports; everything else receives
  its dependencies through constructors. No DI container, no globals, no
  `init()` side effects.

## domain — invariants earn structure; absence of invariants earns plainness

- Model a concept as a struct with unexported fields and behaviour methods
  **when there is an invariant to guard**. When there is none, a plain struct
  with exported fields is the correct Go model — that is not an anaemic-model
  smell, it is the zero-cost honest shape (`artifact.Artifact`).
- **Value objects**: a named type with a constructor `NewName(s string) (Name,
  error)` that validates and returns a ready value — but only where the
  validation or semantics justify a type. Do not wrap every string; idiomatic
  Go validates at the boundary and passes plain values inward.
- **Zero value useful**: options, accumulators and other invariant-free types
  should work without a constructor. Require `New…` only to enforce an
  invariant.
- **Factories**: `New…` by default; an intent-revealing name (`Register…`,
  `Parse…`) when the construction itself carries domain meaning.
- **Domain events**: in a synchronous program the return value *is* the event.
  Introduce an event struct (past tense: `ServiceDeployed`) only when more
  than one consumer must react; collect on the aggregate and let the caller
  dispatch after persisting.
- Purity holds untranslated: a domain-bearing package imports the standard
  library and shared code only — no I/O, no transport, no adapter imports.

## command — the write path is a function

- A command + its handler collapse into **one exported function or method** on
  an application type:

  ```go
  func (s *Service) RegisterCluster(ctx context.Context, name, host string) (ID, error)
  ```

  The parameter list is the command; the body is the handler. Do not create a
  `RegisterClusterHandler` struct with a `Handle` method — that pattern reifies
  functions for DI containers Go does not have.
- Reify a params struct (`RegisterClusterParams`) only when the input has many
  fields or must be serialised, queued or logged as a unit.
- The function stays thin: build domain values from primitives → invoke
  domain behaviour → persist through a port. Business rules live in the domain
  type, not in the function.
- Dependencies are port interfaces held as fields on the service struct,
  injected via its constructor at the composition root. `ctx context.Context`
  is the first parameter of anything that does I/O.
- Return the minimum the caller needs (usually an id and an error), not the
  domain value.

## query — the read path is a function returning a plain struct

- A query is a function or method that reads storage/providers directly and
  returns a **read struct**: plain, exported fields, shaped for the caller,
  free to differ from any domain type.
- Never build write-side domain values just to read — their invariants are
  irrelevant on the read path.
- Read/write separation is **discipline, not directories**: no mutation on the
  read path, return copies or read-only views. Reads and writes live in the
  same package until the read side grows dependencies of its own; only then
  split.
- Missing optional sources degrade to partial results where that is the
  sensible behaviour, not to errors.

## persistence — exported port methods, adapter as a package

- The repository port is an interface in the **consuming** package. Its methods
  MUST be exported — a lowercase interface method is unexported in Go, and an
  adapter in another package literally cannot implement it. The contract's
  `of`/`byName` vocabulary translates as:

  | Contract role | Go method |
  | --- | --- |
  | lookup by id | `Cluster(ctx, id)` — getters drop `Get` |
  | lookup by name | `ClusterByName(ctx, name)` |
  | existence | `Exists(ctx, name)` |
  | add / update | `Save(ctx, c)` |
  | remove | `Delete(ctx, id)` |

- The adapter is its own package named for the technology (`sqlite`, `gitcli`,
  `fsstore`), implementing the port with a compile-time assertion. All mapping
  between storage shape and domain shape lives inside it.
- Swapping backends is adding a package and rewiring the composition root —
  nothing else changes.

## naming — replaces the contract's default layout wholesale

The contract's per-layer directory tree does **not** apply to Go. The layout is:

```
main.go                  # inbound adapter + composition root
internal/
  <feature>/             # one feature: domain types, ports, application funcs
    <adapter>/           # optional subpackage for a heavy adapter (source/gitcli)
  <transport>/           # optional: tui, httpapi — inbound edges as features
```

- Package names: short, lowercase, singular, no underscores. Forbidden:
  `util`, `common`, `helpers`, `misc`.
- No stutter: `source.Manifest`, not `source.SourceManifest`. The one central
  type may share the package name (`source.Source`).
- **Files group a cohesive concern — not one type per file.** Name the file
  after the concern (`writer.go`, `manifest.go`), not after a type.
- `MixedCaps`; initialisms keep their case (`ID`, `URL` → `userID`,
  `parseURL`); getters without `Get`; receivers short and consistent per type.
- Command functions keep the contract's `<Action><Subject>` structure —
  `RegisterCluster` — as the function name.
- Events, when they exist, are past tense.

## When the full contract shape is justified

A large Go service with several rich subdomains may group packages as
`internal/<context>/...` with a handful of packages per context. The dependency
rule then applies between those packages exactly as above — but the default is
flat, and structure is added only when scale forces it. *"Clear is better than
clever."*
