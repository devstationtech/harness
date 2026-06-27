// Package workspace owns the project side of harness: the manifest that records
// which artifacts are active and the generated AGENTS.md entry point.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/devstationtech/harness/internal/artifact"
	"gopkg.in/yaml.v3"
)

// manifestVersion is the current schema version of the project manifest. v2
// records the source, version and content digest per selection at the project
// root, folding in what the retired harness.lock used to hold.
const manifestVersion = 2

// Manifest records the artifacts a project has activated. It is the single
// declarative record of the project's harness state, committed at the project
// root: what is selected, from which source, at which version, and (for
// vendored remote artifacts) a content digest for integrity.
type Manifest struct {
	Version    int         `yaml:"version"`
	Selections []Selection `yaml:"selections"`
}

// Selection is one activated artifact.
type Selection struct {
	Kind    artifact.Kind `yaml:"kind"`
	Name    string        `yaml:"name"`
	Source  string        `yaml:"source"`            // origin name: local | home | <remote>
	Version string        `yaml:"version,omitempty"` // SemVer; empty = unversioned
	Digest  string        `yaml:"digest,omitempty"`  // sha256 of vendored content; empty if referenced
}

// SelectionOf builds a selection for a resolved artifact and an optional content
// digest (empty for artifacts referenced in place).
func SelectionOf(a artifact.Artifact, digest string) Selection {
	return Selection{
		Kind:    a.Kind,
		Name:    a.Name,
		Source:  a.Origin,
		Version: a.Version,
		Digest:  digest,
	}
}

// NewManifest builds a manifest from a set of selections, in canonical order.
func NewManifest(selections []Selection) Manifest {
	ordered := append([]Selection(nil), selections...)
	sortSelections(ordered)
	return Manifest{Version: manifestVersion, Selections: ordered}
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

// Save writes the manifest to path as YAML, creating the parent directory.
func (m Manifest) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
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
