// Package catalog merges an ordered list of artifact sources into a single,
// deduplicated view for selection. Precedence resolves collisions: when two
// sources expose the same kind and name, the higher-precedence source wins.
package catalog

import (
	"sort"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/source"
)

// Catalog is the merged set of artifacts available to a project across all
// sources, with higher-precedence sources shadowing lower ones.
type Catalog struct {
	artifacts []artifact.Artifact
	issues    []source.Issue
}

// Load resolves every source and merges them. Sources are given in precedence
// order, highest first (typically the project, then the shared library, then
// any remote sources). When two sources expose the same identity, the
// higher-precedence artifact wins and is flagged as overriding the lower one.
func Load(sources ...source.Source) (Catalog, error) {
	winners := make(map[artifact.Identity]artifact.Artifact)
	order := make([]artifact.Identity, 0)
	shadowed := make(map[artifact.Identity]bool)
	var issues []source.Issue

	for _, src := range sources {
		resolved, srcIssues, err := src.Resolve()
		if err != nil {
			return Catalog{}, err
		}
		issues = append(issues, srcIssues...)
		for _, a := range resolved {
			id := a.Identity()
			if _, taken := winners[id]; taken {
				shadowed[id] = true // a lower-precedence source also has it
				continue
			}
			winners[id] = a
			order = append(order, id)
		}
	}

	for id := range shadowed {
		a := winners[id]
		a.OverridesShared = true
		winners[id] = a
	}

	merged := make([]artifact.Artifact, 0, len(order))
	for _, id := range order {
		merged = append(merged, winners[id])
	}
	sortArtifacts(merged)
	return Catalog{artifacts: merged, issues: issues}, nil
}

// Issues returns the artifacts that were skipped during loading, with reasons.
func (c Catalog) Issues() []source.Issue {
	return c.issues
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
