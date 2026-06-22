// Package workspace owns the project side of harness: the manifest that records
// which artifacts are active and the generated AGENTS.md entry point.
package workspace

import (
	"fmt"
	"os"
	"sort"

	"github.com/devstationtech/harness/internal/artifact"
	"gopkg.in/yaml.v3"
)

// manifestVersion is the current schema version of the project manifest.
const manifestVersion = 1

// Manifest records the artifacts a project has activated. Under the current
// reference model the manifest is the source of truth for what is active; shared
// artifacts are referenced in place rather than copied, and the structure leaves
// room for a future install/vendor model (origin, version, checksum).
type Manifest struct {
	Version    int         `yaml:"version"`
	Selections []Selection `yaml:"selections"`
}

// Selection is one activated artifact, identified by kind, name and the source
// it resolved from.
type Selection struct {
	Kind   artifact.Kind   `yaml:"kind"`
	Name   string          `yaml:"name"`
	Source artifact.Source `yaml:"source"`
}

// NewManifest builds a manifest from a set of resolved artifacts.
func NewManifest(artifacts []artifact.Artifact) Manifest {
	selections := make([]Selection, 0, len(artifacts))
	for _, a := range artifacts {
		selections = append(selections, Selection{Kind: a.Kind, Name: a.Name, Source: a.Source})
	}
	sortSelections(selections)
	return Manifest{Version: manifestVersion, Selections: selections}
}

// Identities returns the identity keys of every selection, for pre-selecting the
// catalog when re-running harness in a configured project.
func (m Manifest) Identities() []artifact.Identity {
	ids := make([]artifact.Identity, 0, len(m.Selections))
	for _, s := range m.Selections {
		ids = append(ids, artifact.Identity{Kind: s.Kind, Name: s.Name})
	}
	return ids
}

// LoadManifest reads a manifest from path. A missing file yields an empty
// manifest (no selections yet), not an error.
func LoadManifest(path string) (Manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Manifest{Version: manifestVersion}, nil
		}
		return Manifest{}, err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("invalid manifest %s: %w", path, err)
	}
	return manifest, nil
}

// Marshal renders the manifest as YAML.
func (m Manifest) Marshal() ([]byte, error) {
	return yaml.Marshal(m)
}

func sortSelections(selections []Selection) {
	order := map[artifact.Kind]int{}
	for index, kind := range artifact.Kinds() {
		order[kind] = index
	}
	sort.SliceStable(selections, func(i, j int) bool {
		if selections[i].Kind != selections[j].Kind {
			return order[selections[i].Kind] < order[selections[j].Kind]
		}
		return selections[i].Name < selections[j].Name
	})
}
