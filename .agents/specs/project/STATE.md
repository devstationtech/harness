# harness — State (persistent memory)

Decisions, blockers, lessons, and deferred ideas that survive across sessions.

## Decisions

| # | Decision | Rationale | Date |
|---|----------|-----------|------|
| D1 | Artifact format follows Agent Skills / agentskills.io (`<container>/<name>/<ENTRY>.md`, frontmatter `name`==dir, `description`). | Adopt the consolidated open standard; any source's skills stay compatible. | 2026-06-27 |
| D2 | Distribution model = git-repository sources + local index + lockfile (the `apt` / `Homebrew tap` / `Krew` pattern). | Matches the user's "add public/private repos, search, install" vision without inventing a format. | 2026-06-27 |
| D3 | Remote artifacts are **vendored + locked by content hash** (`harness.lock`), not referenced live. | Reproducible, offline, CI-safe; consumers don't need source access at consume time. | 2026-06-27 |
| D4 | First remote source type = **git** (public/private). npm/OCI are later adapters of the same `Source` port. | Most reusable; aligns with "add repo" model. npm-as-foundation rejected (couples to Node). | 2026-06-27 |
| D5 | The git adapter **shells out to the system `git` binary** (no pure-Go git, no custom auth). | Inherits ssh-agent, ssh config, credential helpers, OS keychain / Git Credential Manager for free; private repos "just work" if `git clone` works. | 2026-06-27 |
| D6 | Cross-platform is a hard requirement (macOS, Linux, Windows). | Target audience uses all three. | 2026-06-27 |
| D7 | The existing shared+local merge is the N=2 case of the `Source` port — refactor, don't add a parallel system. | Single resolution path; less code. | 2026-06-27 |
| D8 | The `Source` port returns resolved `artifact.Artifact` (not `Manifest`) and has no `Fetch` method; `Issue` lives in the `source` package. | Behavior-preserving refactor: the app already speaks `Artifact`, and git resolves over its on-disk clone so `Artifact.Directory` suffices for vendoring. `Manifest` (a serializable projection) is introduced with the index (T10); `Issue` in `source` avoids a `source`↔`catalog` import cycle. Honors the rule: small interface, no premature abstraction. | 2026-06-27 |

## Cross-platform rules (binding for the git adapter + hashing)

- Locate git with `exec.LookPath("git")`; never hardcode a path. Clear error if absent.
- Invoke via `exec.Command("git", args...)` with an args slice — never `sh -c` / `cmd /c` (no shell on Windows; avoids injection).
- All filesystem paths via `filepath` + `os.UserHomeDir()`; lockfile `path:` stored with forward slashes, converted with `filepath.FromSlash` on disk.
- Neutralize CRLF: run git with `-c core.autocrlf=false -c core.eol=lf` **and** normalize newlines (`\r\n`→`\n`) before computing content hashes, so hashes are identical across OSes.
- Set `GIT_TERMINAL_PROMPT=0` so missing credentials fail fast instead of hanging (CI).
- Vendor = copy (not symlink) — avoids Windows symlink privilege requirement.

## Lessons

- Market discovery (2026-06-27): AGENTS.md is the dominant standard (60k+ repos, Linux Foundation) but **Claude Code does not consume it** — multi-target emit is mandatory (tracked as M2). See memory `harness-market-discovery`.
- The deep-research workflow's synthesis stage returned a stub; the verified claim data was still recovered from the run logs.

## Blockers

- None.

## Deferred ideas

- **D9 — verify against an existing lock**: vendoring on save always re-copies and rewrites the lock (re-selecting legitimately changes content). Verifying vendored content against a committed `harness.lock` and erroring on hash mismatch (SRC-04 ac#4) belongs to a future `harness verify` / frozen-apply command, not the normal save. Pairs naturally with T12 (upgrade).
- Strategic question still open: for multi-agent *distribution* (M2), build emitters in-house vs. delegate to Ruler. Decide at M2.
- Source precedence could later support explicit pinning/priority (apt-style) beyond the default order.

## Preferences

- User prefers design/spec before implementation; no half-specs; hexagonal layering; no abbreviations in names; commits only at meaningful checkpoints or on request.
