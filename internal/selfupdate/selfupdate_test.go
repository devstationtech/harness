package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newServer builds an httptest server emulating the GitHub release endpoints for
// repo "devstationtech/harness" at tag, serving archive as the linux/amd64 asset.
func newServer(t *testing.T, tag string, archive []byte) *httptest.Server {
	t.Helper()
	sum := sha256.Sum256(archive)
	asset := "harness_linux_amd64.tar.gz"
	checksums := hex.EncodeToString(sum[:]) + "  " + asset + "\n"

	mux := http.NewServeMux()
	mux.HandleFunc("/devstationtech/harness/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://example.test/devstationtech/harness/releases/tag/"+tag)
		w.WriteHeader(http.StatusFound)
	})
	mux.HandleFunc("/devstationtech/harness/releases/download/"+tag+"/"+asset, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	})
	mux.HandleFunc("/devstationtech/harness/releases/download/"+tag+"/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, checksums)
	})
	return httptest.NewServer(mux)
}

func testUpdater(srv *httptest.Server, current string) *Updater {
	u := New(current)
	u.baseURL = srv.URL
	u.http = srv.Client()
	u.goos = "linux"
	u.goarch = "amd64"
	return u
}

func TestLatestVersion(t *testing.T) {
	srv := newServer(t, "v0.2.0", tarGz(t, "harness", []byte("x")))
	defer srv.Close()

	got, err := testUpdater(srv, "0.1.0").LatestVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != "v0.2.0" {
		t.Errorf("LatestVersion = %q, want v0.2.0", got)
	}
}

func TestAvailable(t *testing.T) {
	srv := newServer(t, "v0.2.0", tarGz(t, "harness", []byte("x")))
	defer srv.Close()

	latest, ok := testUpdater(srv, "0.1.0").Available(context.Background())
	if !ok || latest != "v0.2.0" {
		t.Errorf("Available = (%q, %v), want (v0.2.0, true)", latest, ok)
	}
	if _, ok := testUpdater(srv, "0.2.0").Available(context.Background()); ok {
		t.Error("Available reported an update when already current")
	}
}

func TestUpdateReplacesBinary(t *testing.T) {
	newBytes := []byte("the-new-harness-binary")
	srv := newServer(t, "v0.3.0", tarGz(t, "harness", newBytes))
	defer srv.Close()

	// Point the swap at a throwaway "executable".
	exe := filepath.Join(t.TempDir(), "harness")
	if err := os.WriteFile(exe, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	restore := targetExecutable
	targetExecutable = func() (string, error) { return exe, nil }
	defer func() { targetExecutable = restore }()

	version, err := testUpdater(srv, "0.1.0").Update(context.Background(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if version != "v0.3.0" {
		t.Errorf("Update returned %q, want v0.3.0", version)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newBytes) {
		t.Errorf("binary not replaced: got %q", got)
	}
}

func TestUpdateUpToDate(t *testing.T) {
	srv := newServer(t, "v0.3.0", tarGz(t, "harness", []byte("x")))
	defer srv.Close()

	_, err := testUpdater(srv, "0.3.0").Update(context.Background(), io.Discard)
	if !errors.Is(err, ErrUpToDate) {
		t.Errorf("Update on current version: got %v, want ErrUpToDate", err)
	}
}
