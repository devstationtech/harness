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

// mustGit runs a git command in dir, failing the test on error.
func mustGit(t *testing.T, ctx context.Context, dir string, args ...string) {
	t.Helper()
	if _, err := gitcli.Run(ctx, dir, args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

// writeIndexedSkill writes a skill and a package manifest versioning it.
func writeIndexedSkill(t *testing.T, repo, name, description, version string) {
	t.Helper()
	dir := filepath.Join(repo, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "---\nname: " + name + "\ndescription: " + description + "\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	index := "artifacts:\n  - kind: skill\n    name: " + name + "\n    version: " + version + "\n    path: skills/" + name + "\n"
	if err := os.WriteFile(filepath.Join(repo, source.ArtifactsManifestFile), []byte(index), 0o644); err != nil {
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

func TestUpgradeReVendorsAndReportsVersionTransition(t *testing.T) {
	if err := gitcli.Available(); err != nil {
		t.Skip("git not available")
	}
	ctx := context.Background()

	// @Given a versioned git source and a project that vendored it at 1.0.0
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
	if !ok || found.Version != "1.0.0" {
		t.Fatalf("fixture skill missing or wrong version: %+v", found)
	}
	_, digest, err := vendor.Vendor(found, project)
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.NewManifest([]workspace.Selection{workspace.SelectionOf(found, digest)}).Save(config.ManifestPath(project)); err != nil {
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

	// @When the source bumps the version and content
	writeIndexedSkill(t, origin, "reviewer", "second edition", "1.1.0")
	mustGit(t, ctx, origin, "add", ".")
	mustGit(t, ctx, origin, "commit", "-m", "bump")

	// @Then upgrade re-vendors, reports the transition, and updates the manifest
	out.Reset()
	if err := app.Upgrade(&out); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if !strings.Contains(out.String(), "mine/reviewer: 1.0.0 → 1.1.0") {
		t.Errorf("expected a version transition, got: %q", out.String())
	}
	updated, err := workspace.LoadManifest(config.ManifestPath(project))
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Selections) != 1 || updated.Selections[0].Version != "1.1.0" {
		t.Errorf("manifest not updated to 1.1.0: %+v", updated.Selections)
	}
	data, err := os.ReadFile(filepath.Join(project, ".agents", "skills", "reviewer", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "second edition") {
		t.Error("vendored content was not updated")
	}
}
