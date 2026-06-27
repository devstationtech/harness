// Package gitcli is a thin, cross-platform wrapper around the system git binary.
// harness shells out to git rather than embedding a git library so that it
// inherits every authentication mechanism the user already has configured —
// ssh-agent, ssh config, credential helpers, and OS keychains / Git Credential
// Manager. A private repository "just works" if `git clone` works in the shell.
package gitcli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrNotFound is returned when the git binary is not on PATH.
var ErrNotFound = errors.New("git is required but was not found on PATH; install git and retry")

// Available reports whether the git binary can be located on PATH.
func Available() error {
	if _, err := exec.LookPath("git"); err != nil {
		return ErrNotFound
	}
	return nil
}

// Run executes git with the given arguments inside dir (the process working
// directory when dir is empty), returning trimmed stdout. Every invocation
// neutralizes line-ending conversion so checked-out files are byte-identical
// across operating systems, and disables interactive prompts so missing
// credentials fail fast instead of hanging. git is invoked directly with an
// argument slice — never through a shell — for portability and safety.
func Run(ctx context.Context, dir string, args ...string) (string, error) {
	full := append([]string{"-c", "core.autocrlf=false", "-c", "core.eol=lf"}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, detail)
	}
	return strings.TrimSpace(stdout.String()), nil
}
