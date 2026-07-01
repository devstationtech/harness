package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/catalog"
	"github.com/devstationtech/harness/internal/source"
	"github.com/devstationtech/harness/internal/workspace"
)

// lddLibrary writes an abstract skill and one capability into a temp shared
// library and returns a catalog loaded from it.
func lddLibrary(t *testing.T) catalog.Catalog {
	t.Helper()
	home := t.TempDir()
	write := func(name, frontmatter string) {
		dir := filepath.Join(home, "skills", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		doc := "---\nname: " + name + "\ndescription: d\n" + frontmatter + "---\nbody\n"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("low-level-design", "contracts:\n  - domain\n  - hexagonal\n")
	write("lld-go", "implements: low-level-design\nprovides:\n  - domain\n  - hexagonal\n")

	cat, err := catalog.Load(source.NewLocalDirectory(source.HomeName, home, artifact.SourceShared))
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// lddManifest binds both contracts of the abstract to the single capability.
func lddManifest() workspace.Manifest {
	return workspace.Manifest{Selections: []workspace.Selection{{
		Kind:   artifact.KindSkill,
		Name:   "low-level-design",
		Source: "home",
		Bindings: map[string]workspace.CapabilityList{
			"domain":    {"lld-go"},
			"hexagonal": {"lld-go"},
		},
	}}}
}

func names(tree []artifact.Artifact) map[string]int {
	out := map[string]int{}
	for _, a := range tree {
		out[a.Name]++
	}
	return out
}

func TestCompositionTreeFromAbstractDedupesCapabilities(t *testing.T) {
	// @Given an abstract whose two contracts both bind the same capability
	cat := lddLibrary(t)
	abstract, _ := cat.Find(artifact.Identity{Kind: artifact.KindSkill, Name: "low-level-design"})

	// @When the composition tree is computed from the abstract
	got := names(compositionTree(cat, lddManifest(), abstract))

	// @Then it contains the abstract and the capability exactly once each
	if got["low-level-design"] != 1 || got["lld-go"] != 1 || len(got) != 2 {
		t.Errorf("tree = %v, want one abstract + one capability (deduped)", got)
	}
}

func TestCompositionTreeFromCapabilityPullsAbstract(t *testing.T) {
	// @Given the capability of a composition (abstract not targeted directly)
	cat := lddLibrary(t)
	capability, _ := cat.Find(artifact.Identity{Kind: artifact.KindSkill, Name: "lld-go"})

	// @When the tree is computed from the capability
	got := names(compositionTree(cat, lddManifest(), capability))

	// @Then it still pulls in the abstract (a composition localizes as a whole)
	if got["low-level-design"] != 1 || got["lld-go"] != 1 {
		t.Errorf("tree = %v, want the abstract pulled in alongside the capability", got)
	}
}

func TestCompositionTreePlainArtifactIsItself(t *testing.T) {
	// @Given a plain (non-composed) artifact
	a := sharedSkill(t, "spec-kit")
	cat := lddLibrary(t)

	// @When the tree is computed
	got := compositionTree(cat, lddManifest(), a)

	// @Then it is just that artifact
	if len(got) != 1 || got[0].Name != "spec-kit" {
		t.Errorf("tree = %v, want only spec-kit", names(got))
	}
}
