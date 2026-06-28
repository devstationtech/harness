# Releasing harness

Releases are created **in the GitHub UI**. Publishing a release triggers the
[`Release`](../.github/workflows/release.yml) workflow, which runs
[GoReleaser](https://goreleaser.com) to build every binary, archive them,
generate checksums, and upload them to that release. GoReleaser runs in
`keep-existing` mode, so it never touches the release notes you wrote — it only
attaches the build artifacts.

## 1. Pre-flight

- Make sure `main` is green — the [`CI`](../.github/workflows/ci.yml) workflow
  must pass (`make check` locally mirrors it exactly).
- Pick the next version following [Semantic Versioning](https://semver.org):
  - `v0.Y.Z` while the CLI is still unstable (current phase).
  - patch (`Z`) for fixes, minor (`Y`) for features, major (`X`) for breaking
    changes.
- (Optional) dry-run the build locally to catch config errors first:

  ```sh
  goreleaser release --snapshot --clean   # builds into ./dist, publishes nothing
  ```

## 2. Create the release on GitHub

1. Go to **Releases → Draft a new release**.
2. **Choose a tag**: type the new version (e.g. `v0.1.0`) and "Create new tag on
   publish" — target `main`.
3. Write the title and release notes.
4. Click **Publish release** (mark it a pre-release for `-rc`/`-beta` tags).

Publishing fires the workflow. Within a couple of minutes it:

1. checks out the repo at the release tag,
2. builds `harness` for `linux`, `darwin`, `windows` × `amd64`, `arm64`,
3. packages each as `harness_<os>_<arch>.tar.gz` (`.zip` on Windows),
4. writes `checksums.txt` (SHA-256),
5. uploads them all to the release you just published.

> Triggering from the UI keeps you in control of *when* a release goes out and
> what its notes say; the pipeline only produces the binaries.

## 3. Verify

- Open the release and confirm the archives + `checksums.txt` are attached
  (watch progress under the repo's **Actions** tab).
- Smoke-test the installer (once the repo is public):

  ```sh
  curl -fsSL https://raw.githubusercontent.com/devstationtech/harness/main/install.sh | sh
  harness version       # prints the tag you released
  ```

- Existing installs will see an **update available** prompt in the selection TUI,
  or can run `harness self-update`.

## How users install / update

| Action | Command |
| --- | --- |
| Install (Linux/macOS) | `curl -fsSL https://raw.githubusercontent.com/devstationtech/harness/main/install.sh \| sh` |
| Install (Windows PS) | `irm https://raw.githubusercontent.com/devstationtech/harness/main/install.ps1 \| iex` |
| Install (Go) | `go install github.com/devstationtech/harness@latest` |
| Update | press `u` in the TUI when prompted, or `harness self-update` |

Pin a version with `HARNESS_VERSION=v0.1.0` (or `$env:HARNESS_VERSION`), or
`…/harness@v0.1.0` for `go install`.

> **While the repository is private**, the public download URLs (used by the
> installers and by self-update) are not reachable anonymously. Install with a
> token — `GITHUB_TOKEN=… curl -fsSL …/install.sh | sh` — or `gh release
> download <tag> -R devstationtech/harness`. Self-update starts working for
> everyone once the repo is public. (Set `HARNESS_NO_UPDATE_CHECK=1` to silence
> the TUI's update check entirely.)

## Re-running / fixing a release

- Re-run the build from the **Actions** tab (`workflow_dispatch`) or by
  re-publishing — GoReleaser replaces the artifacts, not the notes.
- To pull a release: delete it and its tag (`git push --delete origin vX.Y.Z`),
  then cut a new, higher patch version. Never re-point an existing tag.
