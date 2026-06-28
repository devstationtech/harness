//go:build windows

package selfupdate

import (
	"os"
	"os/exec"
)

// Relaunch starts the updated binary in a new process (Windows has no exec that
// replaces the image) and returns; the caller should exit so the fresh process
// takes over.
func Relaunch() error {
	exe, err := resolveExecutable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Start()
}
