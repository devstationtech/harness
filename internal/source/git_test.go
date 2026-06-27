package source_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/source"
	"github.com/devstationtech/harness/internal/source/gitcli"
)

// gitFixtureRepo creates a throwaway git repository containing one skill and
// returns its path. It skips the test when git is unavailable.
func gitFixtureRepo(t *testing.T) string {
	t.Helper()
	if err := gitcli.Available(); err != nil {
		t.Skip("git not available")
	}
	repo := t.TempDir()
	ctx := context.Background()
	mustGit(t, ctx, repo, "init", "-b", "main")
	mustGit(t, ctx, repo, "config", "user.email", "test@example.com")
	mustGit(t, ctx, repo, "config", "user.name", "Harness Test")
	writeArtifact(t, repo, artifact.KindSkill, "reviewer", "a code reviewer skill")
	mustGit(t, ctx, repo, "add", ".")
	mustGit(t, ctx, repo, "commit", "-m", "seed")
	return repo
}

func mustGit(t *testing.T, ctx context.Context, dir string, args ...string) {
	t.Helper()
	if _, err := gitcli.Run(ctx, dir, args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

func TestGitRepositorySyncAndResolve(t *testing.T) {
	// @Given a git repository containing one skill
	origin := gitFixtureRepo(t)
	cloneDir := filepath.Join(t.TempDir(), "clone")
	repo := source.NewGitRepository("mine", "file://"+origin, "main", cloneDir, artifact.SourceShared)

	// @When the source is synced and resolved
	if err := repo.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, issues, err := repo.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	// @Then the repository's skill is resolved, tagged with the source
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	if len(got) != 1 || got[0].Name != "reviewer" {
		t.Fatalf("resolved %d artifacts, want the reviewer skill", len(got))
	}
	if got[0].Source != artifact.SourceShared {
		t.Errorf("source = %q, want shared", got[0].Source)
	}
}

func TestGitRepositorySyncIsIdempotent(t *testing.T) {
	// @Given a synced git source
	origin := gitFixtureRepo(t)
	cloneDir := filepath.Join(t.TempDir(), "clone")
	repo := source.NewGitRepository("mine", "file://"+origin, "main", cloneDir, artifact.SourceShared)
	ctx := context.Background()
	if err := repo.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	// @When it is synced again (the fetch + checkout path)
	if err := repo.Sync(ctx); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	// @Then it still resolves the skill
	got, _, err := repo.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("resolved %d artifacts after re-sync, want 1", len(got))
	}
}
