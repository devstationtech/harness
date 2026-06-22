package workspace

import (
	"os"
	"path/filepath"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
)

const dirPermission = 0o755
const filePermission = 0o644

// Apply persists a project's selection: it ensures the .agents structure exists,
// writes the manifest and (re)generates AGENTS.md. Shared artifacts are not
// copied — they are referenced in place by AGENTS.md.
func Apply(projectRoot string, selected []artifact.Artifact) error {
	if err := EnsureStructure(projectRoot); err != nil {
		return err
	}

	manifest := NewManifest(selected)
	manifestBytes, err := manifest.Marshal()
	if err != nil {
		return err
	}
	if err := os.WriteFile(config.ManifestPath(projectRoot), manifestBytes, filePermission); err != nil {
		return err
	}

	agentsBytes, err := RenderAgentsFile(projectRoot, selected)
	if err != nil {
		return err
	}
	return os.WriteFile(config.AgentsFilePath(projectRoot), agentsBytes, filePermission)
}

// EnsureStructure creates the project's .agents directory tree (the kind
// containers plus specs) so the user can author local artifacts immediately.
func EnsureStructure(projectRoot string) error {
	agentsDir := config.AgentsDir(projectRoot)
	dirs := []string{agentsDir, config.SpecsDir(projectRoot)}
	for _, kind := range artifact.Kinds() {
		dirs = append(dirs, filepath.Join(agentsDir, kind.Container()))
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirPermission); err != nil {
			return err
		}
	}
	return nil
}
