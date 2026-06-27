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
	"github.com/devstationtech/harness/internal/source"
	"github.com/devstationtech/harness/internal/source/gitcli"
	"github.com/devstationtech/harness/internal/vendor"
	"github.com/devstationtech/harness/internal/workspace"
)

func TestApplyRestoresDeletedVendoredArtifact(t *testing.T) {
	if err := gitcli.Available(); err != nil {
		t.Skip("git not available")
	}
	ctx := context.Background()

	// @Given a project that vendored a skill from a git source
	origin := t.TempDir()
	writeIndexedSkill(t, origin, "reviewer", "first", "1.0.0")
	mustGit(t, ctx, origin, "init", "-b", "main")
	mustGit(t, ctx, origin, "config", "user.email", "test@example.com")
	mustGit(t, ctx, origin, "config", "user.name", "Harness Test")
	mustGit(t, ctx, origin, "add", ".")
	mustGit(t, ctx, origin, "commit", "-m", "seed")

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
	found, ok := findResolved(resolved, "reviewer")
	if !ok {
		t.Fatal("fixture skill not found")
	}
	_, digest, err := vendor.Vendor(found, project)
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.NewManifest([]workspace.Selection{workspace.SelectionOf(found, digest)}).Save(config.ManifestPath(project)); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(project, ".agents", "skills", "reviewer")

	// @When the vendored artifact is deleted and apply runs
	if err := os.RemoveAll(dest); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := app.Apply(&out); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// @Then it is restored and AGENTS.md is generated
	if !strings.Contains(out.String(), "mine/reviewer: restored") {
		t.Errorf("expected a restore report, got: %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(dest, "SKILL.md")); err != nil {
		t.Errorf("expected the artifact to be restored: %v", err)
	}
	if _, err := os.Stat(config.AgentsFilePath(project)); err != nil {
		t.Errorf("expected AGENTS.md to be generated: %v", err)
	}

	// @And applying again with content intact is clean (no restore or drift)
	out.Reset()
	if err := app.Apply(&out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "restored") || strings.Contains(out.String(), "differs") {
		t.Errorf("expected a clean apply, got: %q", out.String())
	}
}
