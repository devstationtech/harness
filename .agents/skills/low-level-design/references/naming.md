# Contract: naming

Consistent names make the architecture legible. The concrete casing is a stack
concern (see the capability); the *structure* of names is universal.

## Directory layout (per bounded context)

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

## Name structures

- **Command**: `<Action><Subject>` (RegisterCluster).
- **Handler**: the command name + `Handler` (RegisterClusterHandler); one per
  command.
- **Aggregate**: the concept, optionally provider-qualified (`Cluster`,
  `ProxmoxCluster`).
- **Value Object**: the concept (`Name`, `Hostname`).
- **Outbound port**: plural noun for the collection (`Clusters`, `Vaults`).
- **Query**: named for context by its location; method describes the read.
- **Port methods**: domain verbs — `of`, `byName`, `exists`, `add`, `update`,
  `remove`.
- **Event**: past tense (`ServiceDeployed`).
- **Files**: one artifact per file, named after the type it contains.

## Rules

- Names carry domain meaning, never the layer or the technology.
- No abbreviations; spell concepts out.
- A file holds one primary type; the file name mirrors that type.
