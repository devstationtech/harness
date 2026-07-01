# Contract: query (read path)

The read side of CQS, deliberately separate from the write side. Reads do not go
through aggregates.

## Building blocks

- **Query** — reads data and returns a **read model** (a plain record shaped for
  the caller), reading storage or a provider directly. It does not load or
  mutate aggregates.
- **Read Model / Record** — a flat, serialisable structure built for display or
  for the API response, decoupled from the domain model's shape.

## Rules

- Never instantiate aggregates to read. The write model's invariants are
  irrelevant to a read; rebuilding them wastes work and couples the sides.
- A query may read the same storage the write side persists to, or a dedicated
  read store/provider — but it stays on the read path.
- Reads degrade gracefully: a missing optional source yields a partial record,
  not a failure, when that is the sensible behaviour.
- Keep queries free of write-side imports; architecture tests enforce it.
