package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
)

// sharedSkill writes a minimal skill on disk and returns it as a shared artifact.
func sharedSkill(t *testing.T, name string) artifact.Artifact {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "---\nname: " + name + "\ndescription: d\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return artifact.Artifact{
		Kind: artifact.KindSkill, Name: name, Source: artifact.SourceShared, Origin: "home",
		Directory: dir, EntryPath: filepath.Join(dir, "SKILL.md"),
	}
}

func TestMaterializeVendorsLocalizedSharedArtifact(t *testing.T) {
	// @Given a shared artifact the user asked to localize
	a := sharedSkill(t, "foo")
	project := t.TempDir()
	home := t.TempDir() // no sources.yaml → no remotes
	localized := map[artifact.Identity]bool{a.Identity(): true}

	// @When the selection is materialized
	resolved, digests, err := materialize([]artifact.Artifact{a}, project, home, localized)
	if err != nil {
		t.Fatal(err)
	}

	// @Then it is copied into .agents as a local artifact with a digest
	if len(resolved) != 1 || resolved[0].Source != artifact.SourceLocal {
		t.Fatalf("expected foo vendored as local, got %+v", resolved)
	}
	if digests[a.Identity()] == "" {
		t.Error("expected a content digest for the vendored copy")
	}
	if _, err := os.Stat(filepath.Join(project, ".agents", "skills", "foo", "SKILL.md")); err != nil {
		t.Errorf("expected the copy under .agents: %v", err)
	}
}

func TestMaterializeLeavesNonLocalizedSharedReferenced(t *testing.T) {
	// @Given a shared artifact that is not localized
	a := sharedSkill(t, "bar")
	project := t.TempDir()
	home := t.TempDir()

	// @When materialized without a localize request
	resolved, _, err := materialize([]artifact.Artifact{a}, project, home, nil)
	if err != nil {
		t.Fatal(err)
	}

	// @Then it passes through referenced in place (not copied)
	if resolved[0].Source != artifact.SourceShared {
		t.Errorf("non-localized shared should stay referenced, got %q", resolved[0].Source)
	}
}
