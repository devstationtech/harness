# Contract: persistence

Persistence is an **outbound** concern hidden behind a domain port. The domain
says *what* it needs to store and retrieve; the adapter decides *how*.

## Building blocks

- **Repository port** — a domain-owned interface named for the collection it
  guards (`Clusters`, `Vaults`), with methods in domain vocabulary: `of(id)`,
  `byName(name)`, `exists(name)`, `save(aggregate)`, `remove(id)`.
- **Persistence adapter** — the outbound implementation (filesystem, SQL, a
  document store...) that maps aggregates to and from storage.

## Rules

- The domain depends on the port, never on the storage technology. Swapping the
  backend is swapping an adapter, nothing else.
- Mapping (aggregate ↔ row/document/file) lives entirely in the adapter.
- The adapter implements the domain port and may import the domain; it must not
  import the application layer.
- Transactions/consistency are an adapter concern, scoped to the aggregate
  boundary the domain defines.
