# Contract: domain

The domain is the **pure** write-side model: behaviour and invariants, no I/O,
no frameworks, no transport.

## Building blocks

- **Value Object** — immutable, defined by its value, self-validating on
  construction (e.g. `Name`, `Hostname`, `Email`). No identity.
- **Entity** — has identity and a lifecycle; equality is by id.
- **Aggregate** — a consistency boundary around one or more entities, with a
  root that is the only entry point. All invariants are enforced through the
  root. External code holds a reference only to the root.
- **Domain Event** — a past-tense fact the aggregate emits when something
  meaningful happens (`ServiceDeployed`). Collected on the aggregate and
  dispatched after persistence.
- **Domain Port** — an interface the domain needs from the outside (e.g. a
  repository), expressed in domain terms.

## Rules

- Construct aggregates through named factories that make intent explicit
  (`register`, `create`), not anonymous constructors scattered across the code.
- Invariants live in the model, not in handlers or services.
- The domain never imports application, inbound or outbound code.
- No anaemic models: behaviour lives with the data it guards. Avoid a separate
  "operations" layer that only shuffles primitives.
