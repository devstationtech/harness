package app_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devstationtech/harness/internal/app"
	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/workspace"
)

// writeHomeArtifact writes a minimal artifact (entry document) into a shared
// library at home, with extra frontmatter lines (contracts, implements, …).
func writeHomeArtifact(t *testing.T, home string, kind artifact.Kind, name, extraFrontmatter string) {
	t.Helper()
	dir := filepath.Join(home, kind.Container(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "---\nname: " + name + "\ndescription: " + name + " artifact\n" + extraFrontmatter + "---\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, kind.EntryFile()), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeManifest(t *testing.T, project, yaml string) {
	t.Helper()
	if err := os.WriteFile(config.ManifestPath(project), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestApplyGeneratesMCPSection is an end-to-end integration test: it seeds a
// shared library with a composed MCP (abstract + two agent capabilities),
// applies a manifest that enables both, and asserts the generated AGENTS.md on
// disk carries the dedicated MCP section — not the skill "Composed designs" one.
func TestApplyGeneratesMCPSection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HARNESS_HOME", home)
	writeHomeArtifact(t, home, artifact.KindMCP, "gh", "contracts:\n  - target\nmultiple: true\n")
	writeHomeArtifact(t, home, artifact.KindMCP, "gh-claude", "implements: gh\nprovides:\n  - target\n")
	writeHomeArtifact(t, home, artifact.KindMCP, "gh-codex", "implements: gh\nprovides:\n  - target\n")

	project := t.TempDir()
	t.Chdir(project)
	writeManifest(t, project, `version: 3
selections:
    - kind: mcp
      name: gh
      source: home
      bindings:
        target:
            - gh-claude
            - gh-codex
    - kind: mcp
      name: gh-claude
      source: home
    - kind: mcp
      name: gh-codex
      source: home
`)

	var out bytes.Buffer
	if err := app.Apply(&out); err != nil {
		t.Fatalf("apply: %v", err)
	}

	agentsBytes, err := os.ReadFile(config.AgentsFilePath(project))
	if err != nil {
		t.Fatalf("AGENTS.md not generated: %v", err)
	}
	agents := string(agentsBytes)
	for _, want := range []string{"## MCP servers — set up on NEED", "### `gh`", "gh-claude", "gh-codex"} {
		if !strings.Contains(agents, want) {
			t.Errorf("AGENTS.md missing %q:\n%s", want, agents)
		}
	}
	// An MCP is not a skill composition: it must not land under "Composed designs".
	if strings.Contains(agents, "Composed designs") {
		t.Errorf("MCP-only project should have no skill composition section:\n%s", agents)
	}
	// Both capabilities are targets of the composition, hidden from any flat table.
	if strings.Contains(agents, "| `gh-claude` |") {
		t.Errorf("bound target should not appear in a flat table:\n%s", agents)
	}
}

// TestVendorLocalizesWholeCompositionTree is an end-to-end integration test for
// `harness vendor` on a composition: targeting the abstract must localize the
// whole tree (abstract + bound capability) exactly once each, flip both to local
// in the manifest, regenerate AGENTS.md with in-project paths, and be idempotent.
func TestVendorLocalizesWholeCompositionTree(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HARNESS_HOME", home)
	writeHomeArtifact(t, home, artifact.KindSkill, "lld", "contracts:\n  - domain\n  - hexagonal\n")
	writeHomeArtifact(t, home, artifact.KindSkill, "lld-impl", "implements: lld\nprovides:\n  - domain\n  - hexagonal\n")

	project := t.TempDir()
	t.Chdir(project)
	writeManifest(t, project, `version: 3
selections:
    - kind: skill
      name: lld
      source: home
      bindings:
        domain:
            - lld-impl
        hexagonal:
            - lld-impl
    - kind: skill
      name: lld-impl
      source: home
`)

	// @When the abstract is vendored
	var out bytes.Buffer
	if err := app.Vendor(&out, []string{"skill/lld"}); err != nil {
		t.Fatalf("vendor: %v", err)
	}

	// @Then the abstract and its capability are copied in, once each (deduped
	// despite the capability being bound to two contracts).
	for _, name := range []string{"lld", "lld-impl"} {
		if _, err := os.Stat(filepath.Join(project, ".agents", "skills", name, "SKILL.md")); err != nil {
			t.Errorf("expected %s vendored into .agents: %v", name, err)
		}
		if got := strings.Count(out.String(), "Vendored skill/"+name+" into"); got != 1 {
			t.Errorf("expected %s vendored exactly once, got %d:\n%s", name, got, out.String())
		}
	}

	// @And both are recorded as local, with AGENTS.md pointing inside the project.
	manifest, err := workspace.LoadManifest(config.ManifestPath(project))
	if err != nil {
		t.Fatal(err)
	}
	for _, sel := range manifest.Selections {
		if sel.Source != "local" {
			t.Errorf("%s should be local after localizing the tree, got %q", sel.Name, sel.Source)
		}
	}
	agents, err := os.ReadFile(config.AgentsFilePath(project))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(agents), "~/.harness") || strings.Contains(string(agents), home) {
		t.Errorf("AGENTS.md must not reference the shared library after localizing:\n%s", agents)
	}

	// @And a second vendor is a clean no-op (already local).
	out.Reset()
	if err := app.Vendor(&out, []string{"skill/lld"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "already local") {
		t.Errorf("second vendor should report already-local, got: %q", out.String())
	}
}
