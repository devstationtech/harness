# CLI reference

Every harness command. Concepts referenced here (precedence, localization,
composition, sources) are explained in [concepts.md](concepts.md).

```
harness            Select artifacts for the current project (interactive)
harness init       Create and seed the shared library (~/.harness)
harness list       List the merged catalog as plain text
harness source …   Manage artifact sources (add | list | remove)
harness update     Refresh all sources and rebuild the search index
harness search Q   Search artifacts across all sources (offline)
harness upgrade    Re-resolve this project's selections to the latest
harness apply      Reconcile this project from its committed harness.yaml
harness vendor K/N Copy a shared/remote artifact into .agents (local override)
harness self-update Update harness to the latest GitHub release
harness version    Print the version
harness help       Show this help
```

## `harness` (or `harness select`)

The default command. Opens the selection TUI for the current project.

- **Reads**: the merged catalog (`.agents/` + `~/.harness/` + git sources) and
  the current `harness.yaml` (to pre-select).
- **Writes**: vendored copies under `.agents/<container>/<name>/` (for remote and
  localized selections), `harness.yaml`, and `AGENTS.md`.

**Flow**: pick artifacts → for each selected *abstract* artifact, a compose step
binds its contracts to capabilities (a radio choice per contract, or independent
checkboxes when the abstract is `multiple: true`) → a confirm step saves.

The kind tabs include **mcp** when MCP artifacts are present. Selecting an MCP and
choosing its target agents writes nothing to those agents directly — it records
the choice and links the MCP's `MCP.md` in `AGENTS.md`; you (or an agent) then run
the setup script it documents to configure each tool.

| Key | Action |
| --- | ------ |
| `↑`/`↓` (`k`/`j`) | move within a tab |
| `←`/`→` | switch kind tab (rules · skills · agents) |
| `space` / `x` | toggle selection |
| `a` | toggle the whole section |
| `v` | localize (vendor) the highlighted shared/remote artifact |
| `u` | update harness (shown only when a newer release is available) |
| `i` | show artifact detail |
| `enter` | continue (compose, then save) |
| `q` / `esc` | quit without saving |

When a newer release is available, the footer's bottom-left shows an *update
available* hint; pressing `u` downloads it, replaces the binary and relaunches
in place. The check runs once in the background and never blocks startup; set
`HARNESS_NO_UPDATE_CHECK=1` to disable it.

## `harness init`

Seeds `~/.harness` with starter artifacts (`skill-creator`, `spec-kit`, an
example rule and agent) and creates the `skills/`, `rules/`, `agents/`
containers. Existing files are skipped; reports created vs skipped counts.

## `harness list` (`ls`)

Prints the merged catalog grouped by kind, each line showing a checkbox,
`name@version`, source, and description. A local artifact shadowing a shared one
shows `local (override)`. Any load issues (malformed frontmatter, name/dir
mismatch) are listed at the end. Read-only.

## `harness source`

Manage git sources of artifacts (stored in `~/.harness/sources.yaml`).

### `harness source add <git-url> [--name NAME] [--ref REF]`

Clones the repo into `~/.harness/sources/<name>/` and registers it.
`--name` defaults to the repo name; `--ref` pins a branch/tag (empty = the
repo's default branch). Fails if the name already exists. Credentials are never
stored — auth is delegated to your system `git`.

### `harness source list` (`ls`)

Prints each configured source: name, type, URL, ref (`(default)` when empty).

### `harness source remove <name>`

Removes the source from `sources.yaml`, deletes its clone and its index entry.
Artifacts already vendored into projects are left untouched.

## `harness update`

Fetches every configured git source (network) and rebuilds the offline search
index at `~/.harness/index/<source>.yaml`, including the shared library. A source
that fails to refresh keeps its previously indexed records.

## `harness search [query]`

Searches the offline index by name or description (case-insensitive; empty query
lists everything), sorted by source, kind, name. Never hits the network — builds
the index on demand from available sources if it is missing.

## `harness upgrade`

Re-resolves this project's **remote** selections against the current source refs
(syncs each repo first), re-vendors any changed content, and updates versions
and digests in `harness.yaml`. Local and shared selections pass through
unchanged; composition bindings are preserved. An artifact that no longer exists
in its source is left as-is.

## `harness apply`

Reconciles the project **from its committed `harness.yaml`** without the TUI —
the command to run on a fresh clone or in CI. It resolves every selection,
restores any missing vendored copy, verifies on-disk digests, and regenerates
`AGENTS.md`. Offline: it uses existing source clones, so run `harness update`
first if a source has never been fetched on this machine.

## `harness vendor <kind>/<name>`

Copies a shared or remote artifact into `.agents/<container>/<name>/`, overriding
the shared one for this project (see [localization](concepts.md#localization-vendoring)).
Localizing an abstract skill also localizes its bound capabilities, so the
composition is complete for anyone who clones. Run `harness` or `harness apply`
afterwards to regenerate `AGENTS.md`.

Example: `harness vendor skill/spec-kit`

## `harness self-update`

Downloads the latest GitHub release for this OS/arch, verifies its SHA-256
against `checksums.txt`, and replaces the running binary in place. Reports
"already the latest version" when current. Needs write permission to the install
directory (re-run with `sudo`, or reinstall, if harness lives in a system path).
The same machinery powers the selection TUI's `u` shortcut, which additionally
relaunches into the new version.

## `harness version` · `harness help`

Print the version (stamped at build time) or the usage summary. Neither touches
the network.

## Environment

- `HARNESS_HOME` — override the shared-library location (default `~/.harness`).
- `HARNESS_NO_UPDATE_CHECK` — when set, disables the TUI's background check for a
  newer release.
- `HARNESS_VERSION` / `HARNESS_INSTALL_DIR` — used by the install scripts to pin
  a release and choose the install directory.
