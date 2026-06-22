package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
)

func TestManifestRoundTrip(t *testing.T) {
	// @Given a manifest built from selected artifacts
	selected := []artifact.Artifact{
		{Kind: artifact.KindRule, Name: "hexagonal", Source: artifact.SourceShared},
		{Kind: artifact.KindSkill, Name: "cqrs", Source: artifact.SourceLocal},
	}
	manifest := NewManifest(selected)

	// @When it is marshalled and written, then loaded back
	path := filepath.Join(t.TempDir(), "harness.yaml")
	data, err := manifest.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the selections survive the round trip
	if len(loaded.Selections) != 2 {
		t.Fatalf("selections = %d, want 2", len(loaded.Selections))
	}
	ids := loaded.Identities()
	if ids[0] != (artifact.Identity{Kind: artifact.KindRule, Name: "hexagonal"}) {
		t.Errorf("first identity = %+v", ids[0])
	}
}

func TestLoadManifestMissingFileIsEmpty(t *testing.T) {
	// @Given a path with no manifest file
	// @When the manifest is loaded
	manifest, err := LoadManifest(filepath.Join(t.TempDir(), "absent.yaml"))

	// @Then an empty manifest is returned without error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifest.Selections) != 0 {
		t.Errorf("expected no selections, got %d", len(manifest.Selections))
	}
}

func TestApplyWritesStructureManifestAndAgentsFile(t *testing.T) {
	// @Given a project root and a shared artifact referenced in place
	projectRoot := t.TempDir()
	sharedEntry := filepath.Join(t.TempDir(), "skills", "cqrs", "SKILL.md")
	selected := []artifact.Artifact{
		{
			Kind:        artifact.KindSkill,
			Name:        "cqrs",
			Description: "Create a CQRS command | handler.",
			Source:      artifact.SourceShared,
			EntryPath:   sharedEntry,
		},
	}

	// @When the selection is applied
	if err := Apply(projectRoot, selected); err != nil {
		t.Fatal(err)
	}

	// @Then the .agents structure, manifest and AGENTS.md all exist
	for _, kind := range artifact.Kinds() {
		dir := filepath.Join(config.AgentsDir(projectRoot), kind.Container())
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("missing container %s: %v", kind.Container(), err)
		}
	}
	if _, err := os.Stat(config.SpecsDir(projectRoot)); err != nil {
		t.Errorf("missing specs dir: %v", err)
	}
	if _, err := os.Stat(config.ManifestPath(projectRoot)); err != nil {
		t.Errorf("missing manifest: %v", err)
	}

	// @Then AGENTS.md references the shared artifact by absolute path with a
	// table cell that escapes the pipe character
	agentsBytes, err := os.ReadFile(config.AgentsFilePath(projectRoot))
	if err != nil {
		t.Fatal(err)
	}
	agents := string(agentsBytes)
	if !strings.Contains(agents, sharedEntry) {
		t.Errorf("AGENTS.md does not reference shared entry path %q", sharedEntry)
	}
	if !strings.Contains(agents, "Create a CQRS command \\| handler.") {
		t.Errorf("AGENTS.md did not escape pipe in description:\n%s", agents)
	}
	if !strings.Contains(agents, "Skills — load on NEED") {
		t.Errorf("AGENTS.md missing skills section")
	}
}

func TestRenderAgentsFileUsesRelativePathForLocal(t *testing.T) {
	// @Given a local artifact whose entry lives inside the project
	projectRoot := t.TempDir()
	localEntry := filepath.Join(config.AgentsDir(projectRoot), "rules", "x", "RULE.md")
	selected := []artifact.Artifact{
		{Kind: artifact.KindRule, Name: "x", Description: "inv", Source: artifact.SourceLocal, EntryPath: localEntry},
	}

	// @When AGENTS.md is rendered
	out, err := RenderAgentsFile(projectRoot, selected)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the path is project-relative, not absolute
	rendered := string(out)
	if !strings.Contains(rendered, "`.agents/rules/x/RULE.md`") {
		t.Errorf("expected relative path in AGENTS.md:\n%s", rendered)
	}
}
