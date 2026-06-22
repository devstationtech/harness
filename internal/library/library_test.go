package library

import (
	"path/filepath"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/catalog"
)

func TestInitSeedsValidArtifacts(t *testing.T) {
	// @Given a fresh, empty home for the shared library
	home := t.TempDir()

	// @When the library is initialized
	result, err := Init(home)
	if err != nil {
		t.Fatal(err)
	}

	// @Then files are created and every seeded artifact is discoverable & valid
	if len(result.Created) == 0 {
		t.Fatal("expected seeded files to be created")
	}
	cat, err := catalog.Load(home, "")
	if err != nil {
		t.Fatal(err)
	}
	wanted := []artifact.Identity{
		{Kind: artifact.KindSkill, Name: "skill-creator"},
		{Kind: artifact.KindSkill, Name: "spec-kit"},
		{Kind: artifact.KindRule, Name: "hexagonal-architecture"},
		{Kind: artifact.KindAgent, Name: "code-reviewer"},
	}
	for _, id := range wanted {
		if _, ok := cat.Find(id); !ok {
			t.Errorf("seeded artifact %+v not discovered", id)
		}
	}
}

func TestInitIsIdempotent(t *testing.T) {
	// @Given a library that has already been initialized
	home := t.TempDir()
	if _, err := Init(home); err != nil {
		t.Fatal(err)
	}

	// @When init runs again
	second, err := Init(home)
	if err != nil {
		t.Fatal(err)
	}

	// @Then existing files are kept untouched, none recreated
	if len(second.Created) != 0 {
		t.Errorf("expected no files created on second init, got %d", len(second.Created))
	}
	if len(second.Skipped) == 0 {
		t.Error("expected existing files to be reported as skipped")
	}
}

func TestSeedNamesMatchDirectories(t *testing.T) {
	// @Given the seeded library
	home := t.TempDir()
	if _, err := Init(home); err != nil {
		t.Fatal(err)
	}

	// @When loading the catalog
	cat, err := catalog.Load(home, "")
	if err != nil {
		t.Fatal(err)
	}

	// @Then each artifact's name matches its directory (Agent Skills rule)
	for _, a := range cat.All() {
		if a.Name != filepath.Base(a.Directory) {
			t.Errorf("artifact name %q != directory %q", a.Name, filepath.Base(a.Directory))
		}
	}
}
