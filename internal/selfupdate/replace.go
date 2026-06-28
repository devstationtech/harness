package selfupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// targetExecutable resolves the path of the running binary to replace. It is a
// variable so tests can point the swap at a temporary file.
var targetExecutable = os.Executable

// resolveExecutable returns the running binary's path with symlinks resolved, so
// an install behind a symlink replaces the real file.
func resolveExecutable() (string, error) {
	exe, err := targetExecutable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// replaceExecutable atomically swaps the running binary for newBinary. It writes
// the bytes to a temporary file in the same directory (so the final rename is
// atomic and on the same filesystem), then moves it into place.
func replaceExecutable(newBinary []byte) error {
	exe, err := resolveExecutable()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)

	tmp, err := os.CreateTemp(dir, ".harness-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s — re-run with elevated permissions or reinstall: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op once the rename below consumes it

	if _, err := tmp.Write(newBinary); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	return swap(exe, tmpName)
}

// swap moves newFile onto exe. On Unix this is a single atomic rename. On Windows
// a running executable cannot be overwritten, so the current file is moved aside
// to <exe>.old first (and cleaned up on the next launch).
func swap(exe, newFile string) error {
	if runtime.GOOS == "windows" {
		old := exe + ".old"
		_ = os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			return err
		}
		if err := os.Rename(newFile, exe); err != nil {
			_ = os.Rename(old, exe) // roll back
			return err
		}
		_ = os.Remove(old) // usually still locked; cleaned by CleanupPrevious next run
		return nil
	}
	return os.Rename(newFile, exe)
}

// CleanupPrevious removes the <exe>.old file left by a previous Windows update.
// It is a no-op on other platforms and best-effort everywhere.
func CleanupPrevious() {
	if runtime.GOOS != "windows" {
		return
	}
	if exe, err := resolveExecutable(); err == nil {
		_ = os.Remove(exe + ".old")
	}
}
