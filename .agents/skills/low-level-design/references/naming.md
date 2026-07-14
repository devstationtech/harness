# Contract: naming

Consistent names make the architecture legible. The concrete casing, method
spellings and file conventions are a stack concern (see the capability); the
*structure* of names is universal.

## Default directory layout (structural default — a capability may replace it)

This tree is the default for stacks whose idiom is package-by-layer (Kotlin,
TypeScript, PHP). A capability may replace it wholesale when its stack's idiom
differs (e.g. Go uses flat package-by-feature and enforces layer boundaries
through import direction instead). What must survive any replacement is the
**dependency rule**, not the directories.

```
<bc>/
  domain/
    models/        # value objects, entities, aggregate root
    events/        # domain events
    ports/         # outbound (and optional inbound) interfaces
  application/
    commands/      # command data structures
    handlers/      # one per command
    queries/       # read models
    services/      # optional orchestration
  inbound/
    <transport>/   # rpc/http/cli/mcp endpoints
    policies/      # reactions to other contexts' events
  outbound/
    persistence/   # repository adapters
    <integration>/ # external services, processes
```

## Name structures (universal)

- **Command**: `<Action><Subject>` (RegisterCluster) — whether it materialises
  as a class, a struct or a function name is the capability's choice.
- **Handler**: named after its command; one unit of orchestration per command.
- **Aggregate**: the concept, optionally provider-qualified (`Cluster`,
  `ProxmoxCluster`).
- **Value Object**: the concept (`Name`, `Hostname`).
- **Outbound port**: named for the collection it guards (`Clusters`, `Vaults`).
- **Query**: named for context by its location; the method/function name
  describes the read.
- **Port methods**: domain verbs covering lookup by id, lookup by name,
  existence, add/update and removal. Concrete spellings (`of`/`byName` vs
  `Cluster`/`ClusterByName`) belong to the capability — including casing rules
  the language imposes.
- **Event**: past tense (`ServiceDeployed`).

## Rules

- Names carry domain meaning, never the layer or the technology.
- No abbreviations; spell concepts out (short idiomatic scope names allowed
  where the stack's style guide expects them).
- **Structural default:** one primary artifact per file, file named after it.
  A capability may replace this with its stack's grouping convention (e.g. Go
  groups a cohesive concern per file).
