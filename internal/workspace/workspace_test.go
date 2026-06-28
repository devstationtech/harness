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
	// @Given a manifest with a referenced and a vendored selection
	selections := []Selection{
		SelectionOf(artifact.Artifact{Kind: artifact.KindRule, Name: "hexagonal", Origin: "home", Version: "1.0.0"}, ""),
		SelectionOf(artifact.Artifact{Kind: artifact.KindSkill, Name: "cqrs", Origin: "acme", Version: "2.1.0"}, "sha256:abc"),
	}
	manifest := NewManifest(selections)

	// @When it is saved and loaded back
	path := filepath.Join(t.TempDir(), "harness.yaml")
	if err := manifest.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the selections survive, source/version/digest included, rule first
	if len(loaded.Selections) != 2 {
		t.Fatalf("selections = %d, want 2", len(loaded.Selections))
	}
	first := loaded.Selections[0]
	if first.Kind != artifact.KindRule || first.Name != "hexagonal" || first.Source != "home" || first.Version != "1.0.0" {
		t.Errorf("first selection = %+v", first)
	}
	second := loaded.Selections[1]
	if second.Version != "2.1.0" || second.Digest != "sha256:abc" {
		t.Errorf("second selection lost version/digest: %+v", second)
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

func TestApplyWritesManifestAndAgentsFile(t *testing.T) {
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
	if err := Apply(projectRoot, selected, nil, nil); err != nil {
		t.Fatal(err)
	}

	// @Then the manifest sits at the project root (not under .agents)
	if _, err := os.Stat(config.ManifestPath(projectRoot)); err != nil {
		t.Errorf("missing root manifest: %v", err)
	}
	if config.ManifestPath(projectRoot) != filepath.Join(projectRoot, "harness.yaml") {
		t.Errorf("manifest path = %q, want project root", config.ManifestPath(projectRoot))
	}

	// @Then per-kind directories and specs/ are NOT created eagerly; they are
	// materialized on demand only when a local artifact or spec is authored
	for _, kind := range artifact.Kinds() {
		dir := filepath.Join(config.AgentsDir(projectRoot), kind.Container())
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("expected %s not to be created eagerly", kind.Container())
		}
	}
	if _, err := os.Stat(config.SpecsDir(projectRoot)); !os.IsNotExist(err) {
		t.Errorf("expected specs/ not to be created eagerly")
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
	out, err := RenderAgentsFile(projectRoot, selected, nil)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the path is project-relative, not absolute
	rendered := string(out)
	if !strings.Contains(rendered, "`.agents/rules/x/RULE.md`") {
		t.Errorf("expected relative path in AGENTS.md:\n%s", rendered)
	}
}

func TestRenderAgentsFileUsesTildeForSharedHomePath(t *testing.T) {
	// @Given a shared artifact whose entry lives under the user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	sharedEntry := filepath.Join(home, ".harness", "skills", "cqrs", "SKILL.md")
	selected := []artifact.Artifact{
		{Kind: artifact.KindSkill, Name: "cqrs", Description: "shared", Source: artifact.SourceShared, EntryPath: sharedEntry},
	}

	// @When AGENTS.md is rendered for some project
	out, err := RenderAgentsFile(t.TempDir(), selected, nil)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the path is shown under ~/, never leaking the absolute home path
	rendered := string(out)
	if !strings.Contains(rendered, "`~/.harness/skills/cqrs/SKILL.md`") {
		t.Errorf("expected ~-relative path in AGENTS.md:\n%s", rendered)
	}
	if strings.Contains(rendered, home) {
		t.Errorf("AGENTS.md leaked the absolute home path %q:\n%s", home, rendered)
	}
}

func TestRenderAgentsFileComposesAbstractAndCapability(t *testing.T) {
	// @Given an abstract skill (two contracts) and a capability providing one
	projectRoot := t.TempDir()
	abstractEntry := filepath.Join(t.TempDir(), "skills", "lld", "SKILL.md")
	capabilityEntry := filepath.Join(t.TempDir(), "skills", "lld-ts", "SKILL.md")
	selected := []artifact.Artifact{
		{Kind: artifact.KindSkill, Name: "lld", Description: "agnostic", Source: artifact.SourceShared, EntryPath: abstractEntry, Contracts: []string{"domain", "persistence"}},
		{Kind: artifact.KindSkill, Name: "lld-ts", Description: "ts", Source: artifact.SourceShared, EntryPath: capabilityEntry, Implements: "lld", Provides: []string{"domain"}, Stack: "typescript"},
	}

	// @When AGENTS.md is rendered with domain bound to lld-ts and persistence left unset
	bindings := map[artifact.Identity]map[string]string{
		{Kind: artifact.KindSkill, Name: "lld"}: {"domain": "lld-ts"},
	}
	out, err := RenderAgentsFile(projectRoot, selected, bindings)
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(out)

	// @Then a composition section binds domain and flags persistence as unbound
	if !strings.Contains(rendered, "Composed designs") {
		t.Errorf("missing compositions section:\n%s", rendered)
	}
	if !strings.Contains(rendered, "**domain** → `lld-ts`") {
		t.Errorf("domain not bound to lld-ts:\n%s", rendered)
	}
	if !strings.Contains(rendered, "**persistence** → ⚠ no implementation selected") {
		t.Errorf("persistence not flagged as unbound:\n%s", rendered)
	}
	if !strings.Contains(rendered, "incomplete composition") {
		t.Errorf("missing incomplete marker:\n%s", rendered)
	}

	// @And the abstract and bound capability are hidden from the flat skills table
	if strings.Contains(rendered, "| `lld` |") || strings.Contains(rendered, "| `lld-ts` |") {
		t.Errorf("abstract/capability should not appear in the skills table:\n%s", rendered)
	}
}
