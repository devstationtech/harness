//go:build !windows

package selfupdate

import (
	"os"
	"syscall"
)

// Relaunch replaces the current process with a fresh exec of the (updated)
// binary, preserving args and environment — the user lands directly in the new
// version. On success it does not return.
func Relaunch() error {
	exe, err := resolveExecutable()
	if err != nil {
		return err
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}
