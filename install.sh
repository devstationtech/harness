#!/usr/bin/env sh
# harness installer — downloads a prebuilt release binary and installs it.
#
#   curl -fsSL https://raw.githubusercontent.com/devstationtech/harness/main/install.sh | sh
#
# Environment overrides:
#   HARNESS_VERSION       release tag to install (default: latest), e.g. v0.1.0
#   HARNESS_INSTALL_DIR   install directory (default: /usr/local/bin, falling
#                         back to ~/.local/bin when not writable and sudo is
#                         unavailable)
#   GITHUB_TOKEN          token for a PRIVATE repository (uses the GitHub API);
#                         not needed once the repo is public
#
# Windows: use install.ps1 instead.
set -eu

REPO="devstationtech/harness"
BINARY="harness"
VERSION="${HARNESS_VERSION:-latest}"
INSTALL_DIR="${HARNESS_INSTALL_DIR:-${INSTALL_DIR:-/usr/local/bin}}"

info() { printf '%s\n' "$*"; }
err()  { printf 'error: %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

# --- detect platform (must match .goreleaser.yaml archive naming) ---
os=$(uname -s)
case "$os" in
	Linux)  os=linux ;;
	Darwin) os=darwin ;;
	*) err "unsupported OS '$os' — on Windows run install.ps1 instead" ;;
esac

arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*) err "unsupported architecture '$arch'" ;;
esac

asset="${BINARY}_${os}_${arch}.tar.gz"

# --- pick a downloader ---
if have curl; then
	dl_to()   { curl -fsSL "$1" -o "$2"; }
	dl_auth() { curl -fsSL -H "Authorization: Bearer $GITHUB_TOKEN" -H "$2" "$1" -o "$3"; }
elif have wget; then
	dl_to()   { wget -qO "$2" "$1"; }
	dl_auth() { wget -q --header="Authorization: Bearer $GITHUB_TOKEN" --header="$2" -O "$3" "$1"; }
else
	err "need curl or wget to download"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# --- resolve download URLs and fetch ---
if [ -n "${GITHUB_TOKEN:-}" ]; then
	# Private repo: resolve asset IDs through the API and download with auth.
	if [ "$VERSION" = latest ]; then rel="latest"; else rel="tags/$VERSION"; fi
	api="https://api.github.com/repos/$REPO/releases/$rel"
	info "Resolving $REPO release ($VERSION) via API ..."
	json=$(curl -fsSL -H "Authorization: Bearer $GITHUB_TOKEN" \
		-H "Accept: application/vnd.github+json" "$api") || err "could not query the release API"
	au=$(printf '%s' "$json" | tr '{}' '\n\n' \
		| awk -v want="\"name\":\"$asset\"" '
			index($0, want) { if (match($0, /"url":"[^"]+"/)) { print substr($0, RSTART+7, RLENGTH-8); exit } }')
	[ -n "$au" ] || err "asset '$asset' not found in the release"
	info "Downloading $asset ..."
	dl_auth "$au" "Accept: application/octet-stream" "$tmp/$asset"
else
	# Public repo: download release assets directly.
	if [ "$VERSION" = latest ]; then
		base="https://github.com/$REPO/releases/latest/download"
	else
		base="https://github.com/$REPO/releases/download/$VERSION"
	fi
	info "Downloading $asset ($VERSION) ..."
	dl_to "$base/$asset" "$tmp/$asset" || err "download failed — is the repo public and does the release exist?"

	# Best-effort checksum verification.
	if dl_to "$base/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
		if have sha256sum; then sum=$(sha256sum "$tmp/$asset" | awk '{print $1}')
		elif have shasum; then sum=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
		else sum=""; fi
		if [ -n "$sum" ]; then
			if grep -q "$sum  $asset" "$tmp/checksums.txt"; then
				info "Checksum OK."
			else
				err "checksum mismatch for $asset — aborting"
			fi
		fi
	fi
fi

# --- extract ---
tar -xzf "$tmp/$asset" -C "$tmp" || err "could not extract $asset"
[ -f "$tmp/$BINARY" ] || err "archive did not contain '$BINARY'"
chmod +x "$tmp/$BINARY"

# --- install, falling back to ~/.local/bin when the default needs sudo ---
install_into() { # dir
	if [ -d "$1" ] && [ -w "$1" ]; then
		install -m 0755 "$tmp/$BINARY" "$1/$BINARY"
	elif have sudo; then
		info "Elevated permissions required to write to $1 (using sudo)."
		sudo install -d -m 0755 "$1"
		sudo install -m 0755 "$tmp/$BINARY" "$1/$BINARY"
	else
		return 1
	fi
}

target_dir="$INSTALL_DIR"
if ! install_into "$target_dir"; then
	target_dir="$HOME/.local/bin"
	info "Cannot write to $INSTALL_DIR and sudo is unavailable — installing to $target_dir."
	mkdir -p "$target_dir"
	install -m 0755 "$tmp/$BINARY" "$target_dir/$BINARY"
fi

info "Installed: $("$target_dir/$BINARY" version)"
case ":$PATH:" in
	*":$target_dir:"*) ;;
	*) info "Note: $target_dir is not on your PATH — add it to use '$BINARY' directly." ;;
esac
info "Run '$BINARY help' to get started."
