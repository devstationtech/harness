package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SourcesConfig is the list of artifact sources a user has registered,
// persisted as sources.yaml under the shared home — the "sources.list" of
// harness. The local library and project are implicit; remote sources (git
// today) are added here. It never holds credentials: authentication is
// delegated to the system git client.
type SourcesConfig struct {
	Sources []SourceConfig `yaml:"sources"`
}

// SourceConfig describes one registered source.
type SourceConfig struct {
	Name string `yaml:"name"`          // stable identifier; namespaces artifacts
	Type string `yaml:"type"`          // "git" today; npm/oci later
	URL  string `yaml:"url"`           // clone URL (ssh or https)
	Ref  string `yaml:"ref,omitempty"` // branch or tag; empty means the source default
}

// LoadSources reads the sources list from path. A missing file yields an empty
// configuration, not an error.
func LoadSources(path string) (SourcesConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return SourcesConfig{}, nil
		}
		return SourcesConfig{}, err
	}
	var config SourcesConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return SourcesConfig{}, fmt.Errorf("invalid sources file %s: %w", path, err)
	}
	return config, nil
}

// Save writes the sources list to path as YAML, creating the parent directory.
func (c SourcesConfig) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

// Find returns the source registered under name, if any.
func (c SourcesConfig) Find(name string) (SourceConfig, bool) {
	for _, s := range c.Sources {
		if s.Name == name {
			return s, true
		}
	}
	return SourceConfig{}, false
}
