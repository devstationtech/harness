# Contract: command (write path)

The write side of CQS. A change to state flows: **inbound → command → handler →
aggregate → outbound port → events**.

## Building blocks

- **Command** — a data structure carrying *primitive* input describing an intent
  (`RegisterCluster{name, host}`). No behaviour, no domain types — just the
  request.
- **Handler** — orchestrates one command: builds value objects from the
  command's primitives, loads/creates the aggregate through a port, calls
  aggregate methods, persists through the port, returns a minimal result. One
  handler per command.
- **Factory** (optional) — when constructing an aggregate is non-trivial or
  specialised (e.g. per provider), encapsulate it in a factory the handler uses.
- **Application Service** (optional) — orchestration spanning multiple
  aggregates when a single handler is not enough.

## Rules

- The handler is thin: translate → invoke domain → persist. Business rules stay
  in the aggregate.
- Commands carry primitives; value objects are built inside the handler, not
  passed in.
- A handler depends on **ports** (interfaces), injected at the composition root.
- Return the minimum the caller needs (often just an id), not the aggregate.
