// Package catalog discovers artifacts on disk and merges the shared library with
// a project's local artifacts into a single, deduplicated view for selection.
package catalog

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/devstationtech/harness/internal/artifact"
)

// Catalog is the merged set of artifacts available to a project: the union of
// the shared library and the project-local artifacts, with local entries
// shadowing shared ones of the same kind and name.
type Catalog struct {
	artifacts []artifact.Artifact
}

// Load scans both bases and merges them. sharedBase is the root of the shared
// library (~/.harness); localBase is the project's .agents directory. Either may
// be empty or non-existent, in which case it simply contributes nothing.
func Load(sharedBase, localBase string) (Catalog, error) {
	shared, err := scanBase(sharedBase, artifact.SourceShared)
	if err != nil {
		return Catalog{}, err
	}
	local, err := scanBase(localBase, artifact.SourceLocal)
	if err != nil {
		return Catalog{}, err
	}
	return merge(shared, local), nil
}

// All returns every merged artifact in a stable order: by kind (rules, skills,
// agents) then by name.
func (c Catalog) All() []artifact.Artifact {
	return c.artifacts
}

// ByKind returns the merged artifacts of a single kind, in name order.
func (c Catalog) ByKind(kind artifact.Kind) []artifact.Artifact {
	var out []artifact.Artifact
	for _, a := range c.artifacts {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out
}

// Find returns the artifact matching the identity, if present.
func (c Catalog) Find(id artifact.Identity) (artifact.Artifact, bool) {
	for _, a := range c.artifacts {
		if a.Identity() == id {
			return a, true
		}
	}
	return artifact.Artifact{}, false
}

// merge combines shared and local artifacts; a local artifact overrides the
// shared one with the same identity and is flagged as an override.
func merge(shared, local []artifact.Artifact) Catalog {
	byIdentity := make(map[artifact.Identity]artifact.Artifact)
	for _, a := range shared {
		byIdentity[a.Identity()] = a
	}
	for _, a := range local {
		if _, shadows := byIdentity[a.Identity()]; shadows {
			a.OverridesShared = true
		}
		byIdentity[a.Identity()] = a
	}

	merged := make([]artifact.Artifact, 0, len(byIdentity))
	for _, a := range byIdentity {
		merged = append(merged, a)
	}
	sortArtifacts(merged)
	return Catalog{artifacts: merged}
}

func sortArtifacts(items []artifact.Artifact) {
	order := map[artifact.Kind]int{}
	for index, kind := range artifact.Kinds() {
		order[kind] = index
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return order[items[i].Kind] < order[items[j].Kind]
		}
		return items[i].Name < items[j].Name
	})
}

// scanBase reads every kind container under base and parses the artifacts found.
// Directories that do not exist are skipped silently; malformed artifacts are
// skipped so one bad entry never blocks the whole catalog.
func scanBase(base string, source artifact.Source) ([]artifact.Artifact, error) {
	if base == "" {
		return nil, nil
	}
	var found []artifact.Artifact
	for _, kind := range artifact.Kinds() {
		container := filepath.Join(base, kind.Container())
		entries, err := os.ReadDir(container)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			directory := filepath.Join(container, entry.Name())
			a, err := readArtifact(kind, entry.Name(), directory, source)
			if err != nil {
				// Skip malformed artifacts rather than failing the scan.
				continue
			}
			found = append(found, a)
		}
	}
	return found, nil
}

func readArtifact(kind artifact.Kind, dirName, directory string, source artifact.Source) (artifact.Artifact, error) {
	entryPath := artifact.EntryFileFor(kind, directory)
	content, err := os.ReadFile(entryPath)
	if err != nil {
		return artifact.Artifact{}, err
	}
	front, _, err := artifact.ParseDocument(content)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if err := front.Validate(dirName); err != nil {
		return artifact.Artifact{}, err
	}
	return artifact.Artifact{
		Kind:        kind,
		Name:        front.Name,
		Description: front.Description,
		Source:      source,
		Directory:   directory,
		EntryPath:   entryPath,
		Metadata:    front.Metadata,
	}, nil
}
