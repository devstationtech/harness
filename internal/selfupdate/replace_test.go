package selfupdate

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceWritableDirNoSudo(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "harness")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	withExecutable(t, exe)
	// sudo must never be consulted when the directory is writable.
	withSudo(t, func() (string, error) { return "", errors.New("sudo must not be called") })

	if err := replaceExecutable(io.Discard, []byte("new")); err != nil {
		t.Fatalf("writable dir should not need elevation: %v", err)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "new" {
		t.Errorf("binary not replaced: got %q", got)
	}
}

func TestReplaceElevatesWhenDirReadOnly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: the directory would be writable regardless of mode")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "harness")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Restore write permission before t.TempDir cleanup runs (LIFO: this first).
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	withExecutable(t, exe)
	withSudo(t, func() (string, error) { return "", errors.New("not found") })

	err := replaceExecutable(io.Discard, []byte("new"))
	if err == nil {
		t.Fatal("expected an error when the dir is read-only and sudo is missing")
	}
	if !strings.Contains(err.Error(), "elevated permissions") && !strings.Contains(err.Error(), "sudo") {
		t.Errorf("error should explain the permission problem, got: %v", err)
	}
}

func withExecutable(t *testing.T, path string) {
	t.Helper()
	prev := targetExecutable
	targetExecutable = func() (string, error) { return path, nil }
	t.Cleanup(func() { targetExecutable = prev })
}

func withSudo(t *testing.T, fn func() (string, error)) {
	t.Helper()
	prev := lookSudo
	lookSudo = fn
	t.Cleanup(func() { lookSudo = prev })
}
