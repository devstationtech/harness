# Contributing to harness

Thanks for helping improve harness. This guide covers the dev setup, the quality
gate, and the conventions the codebase follows.

## Prerequisites

- **Go** — the version in [`go.mod`](go.mod) (or newer).
- **make** — the developer entry point; run `make` to list targets.
- **git** — harness delegates all source auth to your system `git`.

## Getting started

```sh
git clone git@github.com:devstationtech/harness.git
cd harness
make tools     # installs gofumpt + golangci-lint into $(go env GOPATH)/bin
make check     # the full gate — should pass on a clean checkout
make run       # run the selection TUI from source
```

## Dev loop

| Target | What it does |
| ------ | ------------ |
| `make run` | run the TUI from source |
| `make build` | compile a version-stamped binary into `./dist` |
| `make test` | run all tests |
| `make fmt` | format in place (gofumpt) |
| `make vet` / `make lint` | static analysis / golangci-lint |
| `make check` | **the gate**: gofmt check · vet · lint · test |
| `make install` / `make uninstall` | build from source & install / remove locally |

`make check` is exactly what CI runs. Run it before pushing — a green local gate
means a green PR.

## Conventions

- **Go style is non-negotiable.** The [`go-code-standards`](.agents/rules/go-code-standards/RULE.md)
  rule is the binding standard: idiomatic Go (Effective Go, Google/Uber guides,
  Go Proverbs), SOLID expressed the Go-native way — small **consumer-defined**
  interfaces, composition over inheritance, constructor injection — with a
  hexagonal `internal/` layout, a minimal exported surface, wrapped errors, and
  context-first I/O.
- **Architecture.** Domain and adapters live under `internal/` in small,
  single-responsibility packages (`artifact`, `source`, `catalog`, `config`,
  `vendor`, `workspace`, `index`, `tui`, `app`). New behavior goes behind a small
  interface where it crosses a boundary. See [docs/concepts.md](docs/concepts.md)
  for the domain model.
- **Tests.** Every behavioral change ships with tests (table-driven, `@Given /
  @When / @Then` comments as in the existing suites). Keep tests
  deterministic — no network, no real `$HOME` writes (use `t.TempDir()` and
  `HARNESS_HOME`).
- **Commits.** Conventional Commits (`feat:`, `fix:`, `docs:`, `refactor:`,
  `test:`, `chore:`, `ci:`). One logical change per commit; imperative subject.
- **Docs.** Update `README.md` / `docs/` and the relevant spec when you change
  behavior or add a command.

## Dogfooding & ignored files

harness manages its own artifacts with harness. The repo commits **only what is
native to its development**: the `go-code-standards` rule and the specs under
`.agents/specs/`. Everything else under `.agents/` — vendored or localized copies
of shared/third-party skills — is gitignored, as are the generated `harness.yaml`
and `AGENTS.md` (they reference *your* personal `~/.harness` library and are not
portable). Run `harness` locally to regenerate them; don't commit them. If you
add an artifact that genuinely belongs to harness's own development, un-ignore it
explicitly in [`.gitignore`](.gitignore).

## Specs

Non-trivial features are specified before they're built. Specs live in
[`.agents/specs/`](.agents/specs/) (one directory per feature, plus
`project/` for the project-level PROJECT/ROADMAP/STATE). Consult the relevant
spec before implementing, and update it as the design evolves.

## Pull requests

1. Branch from `main` (`git switch -c feat/short-name`).
2. Make the change with tests; keep the diff focused.
3. Run `make check` — it must pass.
4. Open the PR against `main`. CI (format · vet · lint · test) must be green
   before review.
5. Describe **what** changed and **why**; link the spec or issue if there is one.

## Releases

Maintainers cut releases by pushing a `vX.Y.Z` tag — see
[docs/RELEASING.md](docs/RELEASING.md).

## License

By contributing, you agree your contributions are licensed under the project's
[MIT License](LICENSE).
