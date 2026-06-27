package source

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/source/gitcli"
)

// GitRepository resolves artifacts from a git repository checked out into a
// local working copy. Network access happens only in Sync; Resolve reads the
// already-checked-out tree, so the catalog never reaches the network.
type GitRepository struct {
	name     string
	url      string
	ref      string // branch or tag; empty means the repository default branch
	cloneDir string
	tag      artifact.Source
}

// NewGitRepository returns a source for url, checked out under cloneDir. The
// artifacts it resolves are tagged with tag.
func NewGitRepository(name, url, ref, cloneDir string, tag artifact.Source) GitRepository {
	return GitRepository{name: name, url: url, ref: ref, cloneDir: cloneDir, tag: tag}
}

// Name reports the source identifier.
func (g GitRepository) Name() string { return g.name }

// Resolve scans the checked-out tree as a local directory. If the repository
// has not been synced yet, the working copy is absent and Resolve yields
// nothing; callers sync first (on `source add` and `update`).
func (g GitRepository) Resolve() ([]artifact.Artifact, []Issue, error) {
	return NewLocalDirectory(g.name, g.cloneDir, g.tag).Resolve()
}

// Sync brings the working copy up to date: it clones the repository if absent,
// or fetches and checks out the configured ref if it already exists.
func (g GitRepository) Sync(ctx context.Context) error {
	if err := gitcli.Available(); err != nil {
		return err
	}
	_, err := os.Stat(filepath.Join(g.cloneDir, ".git"))
	switch {
	case err == nil:
		return g.update(ctx)
	case errors.Is(err, fs.ErrNotExist):
		return g.clone(ctx)
	default:
		return err
	}
}

// Commit returns the resolved commit SHA of the current checkout.
func (g GitRepository) Commit(ctx context.Context) (string, error) {
	return gitcli.Run(ctx, g.cloneDir, "rev-parse", "HEAD")
}

// clone makes a fresh shallow checkout into a staging directory on the same
// volume, then renames it into place so an interrupted clone never leaves a
// corrupt source active.
func (g GitRepository) clone(ctx context.Context) error {
	parent := filepath.Dir(g.cloneDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".clone-*")
	if err != nil {
		return err
	}
	defer func() {
		// Best-effort cleanup of the staging dir; a successful rename below
		// moves it away, so this becomes a no-op.
		_ = os.RemoveAll(staging)
	}()

	args := []string{"clone", "--depth", "1"}
	if g.ref != "" {
		args = append(args, "--branch", g.ref)
	}
	args = append(args, g.url, staging)
	if _, err := gitcli.Run(ctx, "", args...); err != nil {
		return err
	}

	if err := os.RemoveAll(g.cloneDir); err != nil {
		return err
	}
	return os.Rename(staging, g.cloneDir)
}

// update fetches the configured ref and resets the working copy to it.
func (g GitRepository) update(ctx context.Context) error {
	ref := g.ref
	if ref == "" {
		ref = "HEAD"
	}
	if _, err := gitcli.Run(ctx, g.cloneDir, "fetch", "--depth", "1", "origin", ref); err != nil {
		return err
	}
	_, err := gitcli.Run(ctx, g.cloneDir, "checkout", "--force", "FETCH_HEAD")
	return err
}
