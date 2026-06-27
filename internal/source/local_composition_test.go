package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/source"
)

func writeDoc(t *testing.T, dir, doc string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLocalDirectoryCarriesCompositionFields(t *testing.T) {
	// @Given an abstract skill and a capability resolved by convention
	base := t.TempDir()
	writeDoc(t, filepath.Join(base, "skills", "lld"),
		"---\nname: lld\ndescription: d\ncontracts: [domain, command]\n---\nbody\n")
	writeDoc(t, filepath.Join(base, "skills", "lld-ts"),
		"---\nname: lld-ts\ndescription: d\nimplements: lld\nprovides: [domain, command]\nstack: typescript\n---\nbody\n")

	// @When resolved
	got, issues, err := source.NewLocalDirectory("home", base, artifact.SourceShared).Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}

	// @Then the composition fields survive
	byName := map[string]artifact.Artifact{}
	for _, a := range got {
		byName[a.Name] = a
	}
	if !byName["lld"].IsAbstract() || len(byName["lld"].Contracts) != 2 {
		t.Errorf("abstract not carried: %+v", byName["lld"])
	}
	capability := byName["lld-ts"]
	if !capability.IsCapability() || capability.Implements != "lld" || capability.Stack != "typescript" || len(capability.Provides) != 2 {
		t.Errorf("capability fields not carried: %+v", capability)
	}
}
