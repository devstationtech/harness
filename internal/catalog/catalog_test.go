package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
)

// writeArtifact lays down a minimal valid artifact under base for the given kind.
func writeArtifact(t *testing.T, base string, kind artifact.Kind, name, description string) {
	t.Helper()
	dir := filepath.Join(base, kind.Container(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, kind.EntryFile()), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMergesSharedAndLocal(t *testing.T) {
	// @Given a shared library and a project with distinct artifacts
	shared := t.TempDir()
	local := t.TempDir()
	writeArtifact(t, shared, artifact.KindSkill, "cqrs", "shared cqrs")
	writeArtifact(t, shared, artifact.KindRule, "hexagonal", "shared rule")
	writeArtifact(t, local, artifact.KindSkill, "legacy-import", "local only")

	// @When the catalog is loaded
	cat, err := Load(shared, local)
	if err != nil {
		t.Fatal(err)
	}

	// @Then every artifact from both bases is present
	if got := len(cat.All()); got != 3 {
		t.Fatalf("merged count = %d, want 3", got)
	}
	if _, ok := cat.Find(artifact.Identity{Kind: artifact.KindSkill, Name: "legacy-import"}); !ok {
		t.Error("expected local-only artifact to be present")
	}
}

func TestLoadLocalOverridesShared(t *testing.T) {
	// @Given the same skill name in both the shared library and the project
	shared := t.TempDir()
	local := t.TempDir()
	writeArtifact(t, shared, artifact.KindSkill, "cqrs", "shared description")
	writeArtifact(t, local, artifact.KindSkill, "cqrs", "local description")

	// @When the catalog is loaded
	cat, err := Load(shared, local)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the local artifact wins and is flagged as an override
	resolved, ok := cat.Find(artifact.Identity{Kind: artifact.KindSkill, Name: "cqrs"})
	if !ok {
		t.Fatal("cqrs not found")
	}
	if resolved.Source != artifact.SourceLocal {
		t.Errorf("source = %q, want local", resolved.Source)
	}
	if resolved.Description != "local description" {
		t.Errorf("description = %q, want local description", resolved.Description)
	}
	if !resolved.OverridesShared {
		t.Error("expected OverridesShared to be true")
	}
}

func TestLoadIsOrderedByKindThenName(t *testing.T) {
	// @Given artifacts of several kinds out of order
	shared := t.TempDir()
	writeArtifact(t, shared, artifact.KindSkill, "zeta", "s")
	writeArtifact(t, shared, artifact.KindSkill, "alpha", "s")
	writeArtifact(t, shared, artifact.KindRule, "rule-one", "r")
	writeArtifact(t, shared, artifact.KindAgent, "agent-one", "a")

	// @When the catalog is loaded
	cat, err := Load(shared, "")
	if err != nil {
		t.Fatal(err)
	}

	// @Then results are grouped rules, skills, agents and sorted by name within
	all := cat.All()
	want := []string{"rule-one", "alpha", "zeta", "agent-one"}
	if len(all) != len(want) {
		t.Fatalf("count = %d, want %d", len(all), len(want))
	}
	for i, name := range want {
		if all[i].Name != name {
			t.Errorf("position %d = %q, want %q", i, all[i].Name, name)
		}
	}
}

func TestLoadSkipsMissingBases(t *testing.T) {
	// @Given a non-existent shared base and an empty local base
	// @When the catalog is loaded
	cat, err := Load(filepath.Join(t.TempDir(), "does-not-exist"), "")

	// @Then loading succeeds with an empty catalog
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cat.All()) != 0 {
		t.Errorf("expected empty catalog, got %d", len(cat.All()))
	}
}
