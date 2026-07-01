---
name: lld-go
description: Go implementation of the low-level-design contracts — hexagonal + tactical DDD + CQS in idiomatic Go (small interfaces as ports, packages per layer, constructor injection). Use alongside the low-level-design abstract skill when building a Go bounded context. Draft — refine the concrete patterns over time.
metadata:
  category: architecture
  author: andrespineli
  version: "0.1.0"
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

# Low-Level Design — Go capability (draft)

Idiomatic Go for each `low-level-design` contract. Read the agnostic contract
first, then apply these Go specifics.

## Stack baseline

- Ports are **small interfaces** defined at the consumer (the domain/application
  package), implemented by adapters in `outbound/`.
- One package per layer under `internal/<bc>/{domain,application,inbound,outbound}`.
- Value objects are small structs with constructor functions that validate and
  return `(T, error)`; aggregates expose behaviour methods, not exported fields.
- Commands are plain structs of primitives; handlers depend on port interfaces
  injected via constructors; no global state.
- Queries read storage directly and return plain record structs.
- Names: package names short and lowercase; exported types `MixedCaps`; no
  stutter; errors wrapped with `%w`. (See the go-code-standards rule.)

## Go idioms (Effective Go)

- **Receivers**: pointer receiver when the method mutates state or the struct is
  large — aggregates and handlers take pointer receivers; small immutable value
  objects take value receivers. One receiver kind per type, one short consistent
  receiver name.
- **Constructors** are named `New…` (or `New` when the type is the package's only
  export). They validate and return a ready value — never a half-built struct.
- **Getters** drop the `Get` prefix: `Owner()`, not `GetOwner()`; a setter is
  `SetOwner()`. Expose behaviour; keep fields unexported.
- **Zero value useful**: where a struct has no invariant to protect (an options
  or accumulator type), make its zero value ready to use instead of requiring a
  constructor. Require a constructor only when there is an invariant to enforce.
- **Compile-time port check** in each adapter: `var _ domain.Port = (*adapter)(nil)`.

> Draft: per-contract reference files to be added as the patterns settle.
