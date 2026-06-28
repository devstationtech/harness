# Releasing harness

Releases are fully automated. Pushing a SemVer tag (`vX.Y.Z`) triggers the
[`Release`](../.github/workflows/release.yml) workflow, which runs
[GoReleaser](https://goreleaser.com) to build every binary, archive them,
generate checksums, and publish a GitHub Release.

## 1. Pre-flight

- Make sure `main` is green — the [`CI`](../.github/workflows/ci.yml) workflow
  must pass (`make check` locally mirrors it exactly).
- Pick the next version following [Semantic Versioning](https://semver.org):
  - `v0.Y.Z` while the API/CLI is still unstable (current phase).
  - patch (`Z`) for fixes, minor (`Y`) for features, major (`X`) for breaking
    changes.
- (Optional) dry-run the release locally to catch config errors before tagging:

  ```sh
  goreleaser release --snapshot --clean   # builds into ./dist, publishes nothing
  ```

## 2. Tag and push

```sh
git switch main && git pull
git tag -a v0.1.0 -m "harness v0.1.0"
git push origin v0.1.0
```

Pushing the tag **is** the publish step. The workflow then:

1. checks out the repo at the tag,
2. builds `harness` for `linux`, `darwin`, `windows` × `amd64`, `arm64`,
3. packages each as `harness_<os>_<arch>.tar.gz` (`.zip` on Windows),
4. writes `checksums.txt` (SHA-256),
5. creates the GitHub Release **harness v0.1.0** with a changelog and all assets.

A pre-release tag (e.g. `v0.1.0-rc.1`) is published as a GitHub **pre-release**
automatically (`prerelease: auto` in `.goreleaser.yaml`).

## 3. Verify

- Open the Release on GitHub and confirm the archives + `checksums.txt` are
  attached.
- Smoke-test the installer (once the repo is public):

  ```sh
  curl -fsSL https://raw.githubusercontent.com/devstationtech/harness/main/install.sh | sh
  harness version    # should print the tag you released
  ```

## How users install a release

| Platform | Command |
| --- | --- |
| Linux / macOS | `curl -fsSL https://raw.githubusercontent.com/devstationtech/harness/main/install.sh \| sh` |
| Windows (PowerShell) | `irm https://raw.githubusercontent.com/devstationtech/harness/main/install.ps1 \| iex` |
| Go toolchain | `go install github.com/devstationtech/harness@latest` |

Pin a version with `HARNESS_VERSION=v0.1.0` (or `$env:HARNESS_VERSION` on
Windows), or `…/harness@v0.1.0` for `go install`.

> **While the repository is private**, the public `install.sh` / `install.ps1`
> download URLs are not reachable anonymously. Either install with a token —
> `GITHUB_TOKEN=… curl -fsSL …/install.sh | sh` (the script falls back to the
> authenticated GitHub API) — or use `gh release download v0.1.0 -R
> devstationtech/harness`. Once the repo is public, no token is needed.

## Rolling back

Releases are immutable once assets are attached. To pull one: delete the
GitHub Release and its tag (`git push --delete origin vX.Y.Z`), then cut a new,
higher patch version with the fix. Never re-point an existing tag.
