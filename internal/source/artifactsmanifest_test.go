package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devstationtech/harness/internal/source"
)

func TestLoadArtifactsManifestPresent(t *testing.T) {
	// @Given a package manifest on disk
	dir := t.TempDir()
	path := filepath.Join(dir, source.ArtifactsManifestFile)
	body := "artifacts:\n  - kind: skill\n    name: api-designer\n    version: 1.3.0\n    path: skills/api-designer\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	// @When it is loaded
	manifest, present, err := source.LoadArtifactsManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	// @Then it is present with the declared entry
	if !present {
		t.Fatal("expected the manifest to be present")
	}
	if len(manifest.Artifacts) != 1 {
		t.Fatalf("got %d artifacts, want 1", len(manifest.Artifacts))
	}
	got := manifest.Artifacts[0]
	if got.Kind != "skill" || got.Name != "api-designer" || got.Version != "1.3.0" || got.Path != "skills/api-designer" {
		t.Errorf("unexpected entry: %+v", got)
	}
}

func TestLoadArtifactsManifestAbsent(t *testing.T) {
	// @Given no manifest on disk
	// @When loading from a non-existent path
	_, present, err := source.LoadArtifactsManifest(filepath.Join(t.TempDir(), source.ArtifactsManifestFile))
	// @Then it reports absent without error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if present {
		t.Error("expected the manifest to be absent")
	}
}

func TestLoadArtifactsManifestMalformed(t *testing.T) {
	// @Given a malformed manifest
	dir := t.TempDir()
	path := filepath.Join(dir, source.ArtifactsManifestFile)
	if err := os.WriteFile(path, []byte("artifacts: [this is not: valid"), 0o644); err != nil {
		t.Fatal(err)
	}

	// @When it is loaded
	_, _, err := source.LoadArtifactsManifest(path)

	// @Then it errors, naming the file
	if err == nil {
		t.Fatal("expected an error for malformed YAML")
	}
}
