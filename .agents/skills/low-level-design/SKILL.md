---
name: low-level-design
description: Agnostic low-level design contract — hexagonal architecture, tactical DDD and CQS. Defines the layers (domain, application, inbound, outbound), the concepts (aggregates, value objects, commands, handlers, queries, ports, adapters), directory layout and naming, independent of language. Use when structuring a feature or a bounded context, deciding where code lives, or reviewing architecture. It is abstract — each contract is implemented by a stack capability.
metadata:
  category: architecture
  author: andrespineli
  version: "0.1.0"
contracts:
  - hexagonal
  - domain
  - command
  - query
  - persistence
  - naming
---

# Low-Level Design (abstract)

The standard shape of a software feature across projects, independent of stack.
It is **hexagonal architecture + tactical DDD + CQS**: the domain is pure, the
application orchestrates it through commands and queries, and the outside world
reaches it only through ports and adapters.

This skill is **abstract** — it defines *contracts* (concepts and rules). A
stack **capability** (e.g. `lld-typescript`) implements each contract with
concrete code. harness composes the two: load this contract, then the chosen
implementation per concern.

## The four layers

```
inbound  →  application  →  domain  ←  outbound
(drivers)   (commands,       (pure       (adapters
            queries,         model)       behind ports)
            handlers)
```

Dependency rule — they point inward only:

- **Domain** depends on its own domain + shared only. No I/O, no frameworks.
- **Application** depends on its domain + shared. Orchestrates; no transport.
- **Inbound** depends on application/domain + shared; **never** on outbound.
- **Outbound** depends on domain + shared; implements domain ports.

## Contracts (load the matching capability for your stack)

| Contract | What it governs |
| --- | --- |
| [hexagonal](references/hexagonal.md) | Layers, ports, adapters, inbound/outbound boundaries, composition root |
| [domain](references/domain.md) | Aggregates, entities, value objects, domain events |
| [command](references/command.md) | Commands, handlers, the write path (CQS write) |
| [query](references/query.md) | Read models, the read path (CQS read), isolation from the write side |
| [persistence](references/persistence.md) | Outbound persistence behind a domain port |
| [naming](references/naming.md) | File, type, directory and method naming conventions |

## Bounded contexts

Group a feature's layers under one bounded context (`<bc>/`). Contexts share
only a common `shared/` module. Cross-context communication happens through
events and policies, never by importing another context's internals.
