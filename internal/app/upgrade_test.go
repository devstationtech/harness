package app_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devstationtech/harness/internal/app"
	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/lock"
	"github.com/devstationtech/harness/internal/source"
	"github.com/devstationtech/harness/internal/source/gitcli"
	"github.com/devstationtech/harness/internal/vendor"
)

// commitSkillDescription rewrites a skill's description in the fixture repo and
// commits it, producing a new upstream revision.
func commitSkillDescription(t *testing.T, repo, name, description string) {
	t.Helper()
	ctx := context.Background()
	body := "---\nname: " + name + "\ndescription: " + description + "\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(repo, "skills", name, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := gitcli.Run(ctx, repo, "add", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := gitcli.Run(ctx, repo, "commit", "-m", "update "+name); err != nil {
		t.Fatal(err)
	}
}

func findResolved(artifacts []artifact.Artifact, name string) (artifact.Artifact, bool) {
	for _, a := range artifacts {
		if a.Name == name {
			return a, true
		}
	}
	return artifact.Artifact{}, false
}

func TestUpgradeReVendorsChangedArtifact(t *testing.T) {
	// @Given a project that vendored a skill from a git source at some revision
	origin := gitFixture(t)
	home := t.TempDir()
	t.Setenv("HARNESS_HOME", home)
	if err := app.Source(io.Discard, []string{"add", "file://" + origin, "--name", "mine"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	project := t.TempDir()
	t.Chdir(project)

	repo := source.NewGitRepository("mine", "file://"+origin, "main", config.SourceCloneDir(home, "mine"), artifact.SourceShared)
	resolved, _, err := repo.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	skill, ok := findResolved(resolved, "reviewer")
	if !ok {
		t.Fatal("fixture skill not found")
	}
	commit, err := repo.Commit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, entry, err := vendor.Vendor(skill, project, commit)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.New([]lock.Entry{entry}).Save(config.LockPath(project)); err != nil {
		t.Fatal(err)
	}

	// @When nothing changed upstream
	var out bytes.Buffer
	if err := app.Upgrade(&out); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	// @Then upgrade reports no changes
	if !strings.Contains(out.String(), "0 changed") {
		t.Errorf("expected no changes, got: %q", out.String())
	}

	// @When the artifact changes upstream
	commitSkillDescription(t, origin, "reviewer", "a substantially new description")

	// @Then upgrade re-vendors it, reports the change, and rewrites the lock
	out.Reset()
	if err := app.Upgrade(&out); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if !strings.Contains(out.String(), "mine/reviewer: updated") {
		t.Errorf("expected an update report, got: %q", out.String())
	}
	data, err := os.ReadFile(filepath.Join(project, ".agents", "skills", "reviewer", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "substantially new description") {
		t.Error("vendored content was not updated")
	}
	relocked, err := lock.Load(config.LockPath(project))
	if err != nil {
		t.Fatal(err)
	}
	if len(relocked.Artifacts) != 1 || relocked.Artifacts[0].ContentHash == entry.ContentHash {
		t.Errorf("expected the lock hash to change, got %+v", relocked.Artifacts)
	}
}
