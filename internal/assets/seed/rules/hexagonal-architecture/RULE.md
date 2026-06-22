---
name: hexagonal-architecture
description: Example invariant — the domain layer must not depend on frameworks, transport, persistence or I/O; dependencies point inward through ports and adapters. Replace or remove this with your project's real invariants.
metadata:
  author: harness
  version: "1.0"
  example: "true"
---

# Hexagonal architecture (example rule)

> This is a seeded example so you can see how a rule looks and renders in
> AGENTS.md. Edit it to match your project, or remove it.

## Invariant

The domain (core business logic) must not import or depend on:

- web/transport frameworks (HTTP, gRPC, messaging clients);
- persistence implementations (ORM, database drivers);
- concrete I/O (filesystem, network, clock, randomness).

The domain defines **ports** (interfaces); the outside world provides
**adapters** that implement them. Dependencies always point inward: adapters
depend on the domain, never the reverse.

## How to verify

- Inspect imports in the domain layer; flag any that reach outward packages.
- New side effects must enter through a port, not a direct dependency.
- Where the language allows, enforce import direction with a linter or an
  architecture test instead of relying on review alone.
