# Contract: hexagonal

The application core is isolated from the outside by **ports** (interfaces the
core owns) and **adapters** (implementations on the edge).

## Boundaries

- **Inbound** (driving): the ways the world calls the core — RPC/HTTP endpoints,
  CLI commands, message consumers. An inbound adapter translates an external
  request into a **command** or **query** and dispatches it. It must never reach
  for an outbound adapter directly.
- **Outbound** (driven): the ways the core calls the world — persistence,
  external services, processes. The core depends on an **outbound port**
  (interface); the adapter implements it.
- **Ports** are owned by the domain/application and expressed in domain
  vocabulary. **Adapters** depend on ports, never the reverse.

## Composition root

There is exactly one place where concrete adapters are wired to ports and
handlers — the **composition root**. No other code constructs its own
dependencies; everything is injected. This keeps the core unaware of which
adapter it runs against.

## Rules

- Dependencies point inward (inbound/outbound → application → domain).
- Inbound never imports outbound; they meet only at the composition root.
- A port belongs to the layer that needs it, not to the adapter that fulfils it.
- Architecture tests enforce these boundaries; sanctioned exceptions are
  explicit and justified.
