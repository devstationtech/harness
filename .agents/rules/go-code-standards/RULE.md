---
name: go-code-standards
description: Go coding standard for this repository — idiomatic Go grounded in Effective Go, the Google and Uber style guides, Go Proverbs, and Kubernetes coding conventions. Expresses SOLID the Go-native way (small consumer-defined interfaces, composition over inheritance, constructor injection), with hexagonal modularization, minimal exported surface, error wrapping, and context-first I/O. Always load and apply when writing, refactoring, or reviewing Go code in this project.
metadata:
  category: engineering
  language: go
---

# Go Code Standard

This is a non-negotiable invariant for all Go code in this repository. When writing or reviewing code, follow these rules. They are the distillation of how large Go projects (Kubernetes, Docker/Moby) actually write code — right-sized for a CLI. Prefer clarity over cleverness: *"Clear is better than clever."*

**Authority order** (when in doubt, defer in this order): Effective Go → Google Go Style Guide → Uber Go Style Guide → Go Proverbs → Kubernetes coding conventions.

> SOLID is valid here, but it is OO in origin. In Go it is expressed through small interfaces, composition, and dependency injection — never through deep inheritance hierarchies or Java-style ceremony.

## 1. Tooling is a gate, not an opinion

- All code MUST pass `gofmt`/`gofumpt`, `go vet`, and `golangci-lint` before commit. Formatting and lint findings are never "style preference" — fix them.
- No code is "done" until `make check` is green (build + vet + lint + tests).

## 2. Modularization (hexagonal, inward-only dependencies)

- Domain at the center, adapters at the edge, all under `internal/`. **Dependencies point inward only.** An adapter imports the domain port; the domain never imports an adapter. No import cycles.
- One package = one cohesive responsibility. Package names are short, lowercase, no underscores, singular: `source`, not `sources` or `sourceutil`.
- **Forbidden:** `util`, `common`, `helpers`, `misc` packages. If a function has no cohesive home, the design is wrong — find the right package.
- **No stutter:** name types so the package does not repeat in the type. `source.Manifest`, not `source.SourceManifest`. The single central type may share the package name (`source.Source`, like `context.Context`).
- Do not import the scale of Kubernetes: no code generation, no `pkg/apis` machinery, no deep abstraction layers. This is a CLI.

## 3. SOLID, expressed in Go

| Principle | Rule in this codebase |
| --------- | --------------------- |
| Single Responsibility | A package/type/function has one reason to change. If a type does 3 things, split it. |
| Open/Closed | Extend by adding a new type that satisfies an existing interface — never by editing the consumer. A new source kind is a new adapter, not a change to `catalog`. |
| Liskov | Every implementation honors the interface's documented contract (return errors, do not panic; respect nil/empty semantics). Document the contract on the interface. |
| Interface Segregation | Interfaces are small (1–3 methods). *"The bigger the interface, the weaker the abstraction."* |
| Dependency Inversion | Depend on interfaces (ports); inject concrete implementations via constructors. No global state, no `init()` side effects, no service locators. |

Two mother-rules that capture most of the above:

- **Accept interfaces, return structs.**
- **Define an interface where it is consumed, not where it is implemented.**

```go
// GOOD — small interface, defined at the consumer; constructor injection.
type Source interface {
    Name() string
    Resolve() ([]Manifest, []Issue, error)
}

func Load(sources ...Source) (Catalog, error) { /* ... */ }

// BAD — fat interface the implementer is forced to satisfy in full.
type Source interface {
    Name() string
    Resolve() ([]Manifest, []Issue, error)
    Fetch(Identity) (Payload, error)
    Update() error
    Search(string) []Manifest
    Validate() error
}
```

## 4. Encapsulation

- **Minimal exported surface.** Everything is unexported by default. Export a name only when a caller outside the package needs it.
- Fields stay unexported unless there is a concrete reason to export them; expose behavior through methods, not raw state.
- **Constructors `New…` validate and return a ready value.** Do not return a half-initialized struct for the caller to finish.
- For optional configuration, use **functional options** (`opts ...Option`) — the Docker/gRPC/Kubernetes pattern. Do not use options for a trivial constructor with no optional fields.
- Do not expose mutable internal state. Return copies or read-only views (the `catalog` package uses value receivers and returns copies — keep that discipline).

```go
// GOOD — validates, fails fast, fields stay private.
func NewGitRepository(name, url, ref string) (*GitRepository, error) {
    if _, err := exec.LookPath("git"); err != nil {
        return nil, fmt.Errorf("git is required but was not found on PATH: %w", err)
    }
    return &GitRepository{name: name, url: url, ref: ref}, nil
}
```

## 5. Errors

- Return errors; do not `panic`. `panic` is only for programmer bugs / truly unrecoverable state, never for expected failures (missing file, failed clone).
- Add context by wrapping: `fmt.Errorf("clone %s: %w", url, err)`. Use `errors.Is` / `errors.As` to inspect.
- Error strings are lowercase, no trailing punctuation, no newline.
- Define a typed/sentinel error only when a caller must branch on it (e.g. a "git not found" condition the CLI reports specially).
- Never ignore an error silently. If an error is genuinely ignorable, assign to `_` with a comment saying why. The one blanket exception is writing to an `io.Writer` such as `os.Stdout` in CLI output (`fmt.Fprint*`), where a failure is not actionable — this is excluded centrally in `.golangci.yml`, not with scattered `_ =`.

## 6. Context, naming, concurrency

- `context.Context` is the **first parameter** of any function doing I/O, network, or subprocess work (the git adapter, index refresh, etc.). Name it `ctx`. Never store a context in a struct.
- Naming: full words (no invented abbreviations), **but** short names in short scopes are idiomatic and expected (`i`, `r`, `err`, `ctx`, single-letter receivers). Initialisms keep their case: `URL`, `ID`, `HTTP` → `userID`, `parseURL`, not `userId`/`parseUrl`. Receivers are short and consistent per type (`func (g *GitRepository) ...`).
- Concurrency only when it earns its keep. *"Don't communicate by sharing memory; share memory by communicating."* For bounded parallel work (e.g. fetching sources) use `errgroup` + `context`, never naked goroutines without lifecycle control.

## 7. Testing

Follow the **test pyramid**: many fast unit tests, fewer integration tests, a
handful of true end-to-end ones. Push logic down so it can be unit-tested against
small interfaces; reserve integration tests for the seams that only real I/O
exercises.

- **Unit** — table-driven with `t.Run` subtests, Given/When/Then comments. Standard
  library `testing` + `google/go-cmp` (`cmp.Diff`) for deep equality. No assertion
  or mock framework. Test doubles are **hand-written fakes** over the small
  interfaces (the Kubernetes approach), never generated mocks. Prefer black-box
  tests (`package x_test`) so tests exercise the exported contract. Mark
  independent tests `t.Parallel()`.
- **Integration** — exercise **real I/O** against a temp sandbox (`t.TempDir()`,
  `t.Setenv("HARNESS_HOME", …)`, `t.Chdir(project)`) and drive the actual command
  surface (e.g. `app.Apply`, `app.Vendor`), not internals. Assert on what a user
  would see: files written to disk, `harness.yaml` contents, the generated
  `AGENTS.md`. Read them back and check them — never assert only in memory when the
  behavior *is* a file on disk. A test that needs the network or `git` guards with
  a capability check and `t.Skip` when unavailable.
- **Regression-first coverage** — every exported behavior and every error path has
  a test, and a bug fix starts with a **failing** test that the fix turns green.
  When a change alters an on-disk format (a manifest schema bump, an `AGENTS.md`
  section), pin the new shape with a test *and* keep one that loads the old shape,
  so regressions surface precisely.

## 8. Documentation

- Doc comment on every exported identifier, written as a full sentence starting with the identifier's name (`// Load merges an ordered list of sources …`). This is enforced by lint and by convention.
- Comment the *why*, not the *what*. The code already says what it does.

## Quick review checklist

- [ ] `make check` green (fmt, vet, lint, tests)
- [ ] Dependencies point inward; no new `util`/`common` package; no import cycle
- [ ] Interfaces small and defined at the consumer; accept interfaces, return structs
- [ ] Exported surface is the minimum a caller needs
- [ ] Errors wrapped with context; no swallowed errors; no panic on expected failure
- [ ] `ctx` first param on I/O; initialisms cased correctly
- [ ] Table-driven tests with Given/When/Then; error paths covered
