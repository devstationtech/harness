// Package selfupdate checks GitHub Releases for a newer harness and replaces the
// running binary in place. It backs both `harness self-update` and the selection
// TUI's "update available" prompt. It needs no API token: the latest version is
// read from the releases/latest redirect, and assets are fetched from the public
// release download URLs (the archive naming mirrors .goreleaser.yaml).
package selfupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// DefaultRepo is the GitHub repository releases are pulled from.
const DefaultRepo = "devstationtech/harness"

// binaryName is the executable name inside a release archive (".exe" on Windows).
const binaryName = "harness"

// ErrUpToDate is returned by Update when the running build is already the latest
// release.
var ErrUpToDate = errors.New("already on the latest version")

// Updater checks for and applies harness releases from a GitHub repository.
// The zero value is not usable; construct one with New.
type Updater struct {
	repo    string
	current string
	http    *http.Client
	baseURL string // GitHub web base; the redirect + download host
	goos    string
	goarch  string
}

// New returns an Updater for the running build (current is main.version).
func New(current string) *Updater {
	return &Updater{
		repo:    DefaultRepo,
		current: current,
		http:    &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://github.com",
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
	}
}

// LatestVersion returns the tag of the latest published release (e.g. "v0.2.0").
// It reads the Location of the releases/latest redirect, so it needs no API
// token and is not rate-limited.
func (u *Updater) LatestVersion(ctx context.Context) (string, error) {
	client := *u.http
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	url := fmt.Sprintf("%s/%s/releases/latest", u.baseURL, u.repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no redirect from %s (status %s)", url, resp.Status)
	}
	tag := loc[strings.LastIndex(loc, "/")+1:]
	// With no releases yet, latest redirects to the releases page, not a tag.
	if tag == "" || tag == "releases" {
		return "", errors.New("no published release found")
	}
	return tag, nil
}

// Available reports the latest version and whether it is newer than the running
// build. It never returns an error — any failure (offline, no release, dev
// build) simply yields available=false — so a background check never disrupts
// the CLI.
func (u *Updater) Available(ctx context.Context) (latest string, available bool) {
	latest, err := u.LatestVersion(ctx)
	if err != nil {
		return "", false
	}
	return latest, Newer(u.current, latest)
}

// Update downloads the latest release for this OS/arch, verifies its checksum and
// replaces the running executable in place, returning the version installed.
// Progress is written to w. It returns ErrUpToDate when a release build is
// already current (a dev build always updates to the latest).
func (u *Updater) Update(ctx context.Context, w io.Writer) (string, error) {
	latest, err := u.LatestVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("check latest release: %w", err)
	}
	if isRelease(u.current) && !Newer(u.current, latest) {
		return latest, ErrUpToDate
	}

	binary, err := u.downloadBinary(ctx, latest, w)
	if err != nil {
		return "", err
	}
	fmt.Fprintln(w, "Installing …")
	if err := replaceExecutable(binary); err != nil {
		return "", err
	}
	return latest, nil
}

// downloadBinary fetches the release archive for this OS/arch, verifies it
// against checksums.txt, and returns the extracted binary's bytes.
func (u *Updater) downloadBinary(ctx context.Context, tag string, w io.Writer) ([]byte, error) {
	asset := u.assetName()
	base := fmt.Sprintf("%s/%s/releases/download/%s", u.baseURL, u.repo, tag)

	fmt.Fprintf(w, "Downloading %s %s (%s/%s) …\n", binaryName, tag, u.goos, u.goarch)
	archive, err := u.fetch(ctx, base+"/"+asset)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", asset, err)
	}
	checksums, err := u.fetch(ctx, base+"/checksums.txt")
	if err != nil {
		return nil, fmt.Errorf("download checksums: %w", err)
	}
	if err := verifyChecksum(asset, archive, checksums); err != nil {
		return nil, err
	}
	return extractBinary(archive, asset, u.exeName())
}

// fetch GETs url and returns the body, failing on any non-2xx status.
func (u *Updater) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := u.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// assetName is the release archive for this OS/arch, matching .goreleaser.yaml:
// harness_<os>_<arch>.tar.gz (.zip on Windows).
func (u *Updater) assetName() string {
	ext := "tar.gz"
	if u.goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s_%s_%s.%s", binaryName, u.goos, u.goarch, ext)
}

// exeName is the binary's filename inside the archive.
func (u *Updater) exeName() string {
	if u.goos == "windows" {
		return binaryName + ".exe"
	}
	return binaryName
}
