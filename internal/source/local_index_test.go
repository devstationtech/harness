package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/source"
)

func writeEntryDoc(t *testing.T, dir string, kind artifact.Kind, name, description string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, kind.EntryFile()), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeManifest(t *testing.T, base, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(base, source.ArtifactsManifestFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLocalDirectoryResolvesFromManifestWithVersions(t *testing.T) {
	// @Given a source whose package manifest lists artifacts at free paths
	base := t.TempDir()
	writeEntryDoc(t, filepath.Join(base, "library", "api-designer"), artifact.KindSkill, "api-designer", "d")
	writeEntryDoc(t, filepath.Join(base, "policies", "twelve-factor"), artifact.KindRule, "twelve-factor", "d")
	writeManifest(t, base, "artifacts:\n"+
		"  - kind: skill\n    name: api-designer\n    version: 1.3.0\n    path: library/api-designer\n"+
		"  - kind: rule\n    name: twelve-factor\n    version: 1.0.0\n    path: policies/twelve-factor\n")

	// @When the source is resolved
	got, issues, err := source.NewLocalDirectory("acme", base, artifact.SourceShared).Resolve()
	if err != nil {
		t.Fatal(err)
	}

	// @Then both artifacts resolve with their declared versions and no issues
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	if len(got) != 2 {
		t.Fatalf("got %d artifacts, want 2", len(got))
	}
	byName := map[string]artifact.Artifact{}
	for _, a := range got {
		byName[a.Name] = a
	}
	if byName["api-designer"].Version != "1.3.0" {
		t.Errorf("api-designer version = %q, want 1.3.0", byName["api-designer"].Version)
	}
	if byName["twelve-factor"].Version != "1.0.0" {
		t.Errorf("twelve-factor version = %q, want 1.0.0", byName["twelve-factor"].Version)
	}
}

func TestLocalDirectoryManifestReportsBadEntriesAsIssues(t *testing.T) {
	// @Given a manifest with one good entry and five broken ones
	base := t.TempDir()
	writeEntryDoc(t, filepath.Join(base, "skills", "good"), artifact.KindSkill, "good", "d")
	writeEntryDoc(t, filepath.Join(base, "skills", "mism"), artifact.KindSkill, "actual-name", "d")
	writeManifest(t, base, "artifacts:\n"+
		"  - kind: skill\n    name: good\n    version: 1.0.0\n    path: skills/good\n"+ // ok
		"  - kind: skill\n    name: badver\n    version: 1.0\n    path: skills/good\n"+ // invalid semver
		"  - kind: skill\n    name: escape\n    path: ../outside\n"+ // escapes the source
		"  - kind: skill\n    name: declared\n    path: skills/mism\n"+ // name mismatch
		"  - kind: gizmo\n    name: weird\n    path: skills/good\n"+ // unknown kind
		"  - kind: skill\n    name: good\n    version: 2.0.0\n    path: skills/good\n") // duplicate

	// @When the source is resolved
	got, issues, err := source.NewLocalDirectory("acme", base, artifact.SourceShared).Resolve()
	if err != nil {
		t.Fatal(err)
	}

	// @Then only the good artifact loads and each bad entry is an issue
	if len(got) != 1 || got[0].Name != "good" {
		t.Fatalf("expected only 'good', got %+v", got)
	}
	if len(issues) != 5 {
		t.Fatalf("expected 5 issues, got %d: %v", len(issues), issues)
	}
}
