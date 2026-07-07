package selfupdate

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// targetExecutable resolves the path of the running binary to replace. It is a
// variable so tests can point the swap at a temporary file.
var targetExecutable = os.Executable

// lookSudo finds the sudo binary. It is a variable so tests can stub it.
var lookSudo = func() (string, error) { return exec.LookPath("sudo") }

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

// replaceExecutable swaps the running binary for newBinary. When the install
// directory is writable it stages a temp file alongside the binary and renames
// it into place (atomic, no privileges). When it is not — the common case for a
// system path like /usr/local/bin — it stages the binary in a private temp
// directory and elevates the final install with sudo, which prompts the user for
// their password. Progress is written to w.
func replaceExecutable(w io.Writer, newBinary []byte) error {
	exe, err := resolveExecutable()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)

	// Fast path: the install directory is writable — temp + atomic rename.
	tmp, err := os.CreateTemp(dir, ".harness-update-*")
	if err != nil {
		// Only a permission problem justifies escalating; anything else (disk
		// full, missing directory) is a real error, not a reason to sudo.
		if errors.Is(err, fs.ErrPermission) {
			return replaceElevated(w, exe, newBinary)
		}
		return err
	}
	tmpName := tmp.Name()
	if err := writeBinary(tmp, tmpName, newBinary); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := swap(exe, tmpName); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// writeBinary writes data to the open temp file at path and makes it executable.
func writeBinary(tmp *os.File, path string, data []byte) error {
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Chmod(path, 0o755)
}

// replaceElevated installs newBinary onto exe with elevated privileges, staging
// it in a private temp directory and running `sudo install`. sudo prompts for
// the password on the terminal.
func replaceElevated(w io.Writer, exe string, newBinary []byte) error {
	dir := filepath.Dir(exe)
	if runtime.GOOS == "windows" {
		return fmt.Errorf("cannot write to %s — re-run harness from an elevated prompt", dir)
	}
	sudo, err := lookSudo()
	if err != nil {
		return fmt.Errorf("cannot write to %s and sudo is not available — re-run with elevated permissions or reinstall into a user-writable directory", dir)
	}

	staging, err := os.MkdirTemp("", "harness-update-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()

	staged := filepath.Join(staging, "harness")
	if err := os.WriteFile(staged, newBinary, 0o755); err != nil {
		return err
	}

	fmt.Fprintf(w, "%s needs elevated permissions — installing with sudo (you may be prompted for your password).\n", dir)
	cmd := exec.Command(sudo, "install", "-m", "0755", staged, exe)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo install into %s failed: %w", dir, err)
	}
	return nil
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
