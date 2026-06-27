package workspace

import (
	"os"
	"path/filepath"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
)

const filePermission = 0o644

// Apply persists a project's selection: it writes the manifest at the project
// root and (re)generates AGENTS.md. digests maps a selection's identity to the
// content digest of its vendored copy (empty for artifacts referenced in place).
// bindings maps an abstract skill's identity to its chosen capability per
// contract, recorded verbatim — so an explicit "no implementation" is preserved,
// not re-derived.
//
// Shared and local artifacts are referenced in place by AGENTS.md; only remote
// artifacts are vendored (by the caller) before Apply runs. Per-kind directories
// come into existence on demand when a project-local artifact is authored or
// vendored, so an empty .agents/skills/ never clutters a project.
func Apply(projectRoot string, selected []artifact.Artifact, digests map[artifact.Identity]string, bindings map[artifact.Identity]map[string]string) error {
	selections := make([]Selection, 0, len(selected))
	for _, a := range selected {
		selection := SelectionOf(a, digests[a.Identity()])
		if bound := bindings[a.Identity()]; len(bound) > 0 {
			selection.Bindings = bound
		}
		selections = append(selections, selection)
	}
	if err := NewManifest(selections).Save(config.ManifestPath(projectRoot)); err != nil {
		return err
	}
	removeStale(projectRoot)

	agentsBytes, err := RenderAgentsFile(projectRoot, selected, bindings)
	if err != nil {
		return err
	}
	return os.WriteFile(config.AgentsFilePath(projectRoot), agentsBytes, filePermission)
}

// removeStale best-effort removes manifest and lock files from the pre-v2
// location under .agents, so a project that predates the root manifest does not
// keep two copies.
func removeStale(projectRoot string) {
	agents := config.AgentsDir(projectRoot)
	for _, name := range []string{config.ManifestFileName, "harness.lock"} {
		_ = os.Remove(filepath.Join(agents, name))
	}
}
