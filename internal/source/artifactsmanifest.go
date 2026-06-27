package source

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

// ArtifactsManifestFile is the package manifest a source places at its root to
// version and locate the artifacts it ships. It plays the role package.json
// plays for an npm package.
const ArtifactsManifestFile = "harness.artifacts.yaml"

// ArtifactsManifest is a source's package manifest.
type ArtifactsManifest struct {
	Artifacts []ArtifactEntry `yaml:"artifacts"`
}

// ArtifactEntry locates and versions one artifact within a source. Path is
// relative to the source root, in forward-slash form.
type ArtifactEntry struct {
	Kind    string `yaml:"kind"`
	Name    string `yaml:"name"`
	Version string `yaml:"version,omitempty"`
	Path    string `yaml:"path"`
}

// LoadArtifactsManifest reads the package manifest at path. The boolean reports
// whether the file exists; a missing file is not an error — the source then
// resolves by directory convention instead.
func LoadArtifactsManifest(path string) (ArtifactsManifest, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ArtifactsManifest{}, false, nil
		}
		return ArtifactsManifest{}, false, err
	}
	var manifest ArtifactsManifest
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		return ArtifactsManifest{}, true, fmt.Errorf("invalid artifacts manifest %s: %w", path, err)
	}
	return manifest, true, nil
}
