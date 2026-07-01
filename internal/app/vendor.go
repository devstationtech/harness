package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/catalog"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/vendor"
	"github.com/devstationtech/harness/internal/workspace"
)

// Vendor copies a shared or remote artifact into the project's .agents directory
// so it is committed and overrides the shared one — useful for an open-source
// project that wants specific skills available to contributors without sharing
// the maintainer's whole library. It takes a "<kind>/<name>" argument.
//
// A composition is localized as a whole: targeting the abstract or any of its
// capabilities pulls the entire tree (the abstract plus every bound capability),
// so the project is self-contained for anyone who clones it. Once the files are
// copied, the manifest and AGENTS.md are reconciled in place — there is no stale
// half-localized state to clean up afterwards.
func Vendor(out io.Writer, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: harness vendor <kind>/<name> (e.g. skill/api-designer)")
	}
	kindText, name, ok := strings.Cut(args[0], "/")
	if !ok || kindText == "" || name == "" {
		return fmt.Errorf("usage: harness vendor <kind>/<name> (e.g. skill/api-designer)")
	}
	kind, ok := artifact.ParseKind(kindText)
	if !ok {
		return fmt.Errorf("unknown kind %q", kindText)
	}

	cat, projectRoot, _, err := loadCatalog()
	if err != nil {
		return err
	}
	resolved, ok := cat.Find(artifact.Identity{Kind: kind, Name: name})
	if !ok {
		return fmt.Errorf("no %s named %q in the catalog", kindText, name)
	}

	manifest, err := workspace.LoadManifest(config.ManifestPath(projectRoot))
	if err != nil {
		return err
	}

	vendored := 0
	for _, a := range compositionTree(cat, manifest, resolved) {
		if a.Source == artifact.SourceLocal {
			continue // already under .agents
		}
		if err := vendorInto(out, projectRoot, a); err != nil {
			return err
		}
		vendored++
	}
	if vendored == 0 {
		fmt.Fprintf(out, "%s/%s is already local.\n", kindText, name)
		return nil
	}

	// Persist the localization: re-resolve from the manifest (the vendored copies
	// now win precedence), record them as local and regenerate AGENTS.md.
	return Apply(out)
}

// compositionTree returns every artifact that should be localized together with
// resolved. A composition is an atomic unit: targeting the abstract or any of its
// capabilities pulls the abstract plus every capability its bindings point to,
// deduplicated. A plain artifact yields just itself.
func compositionTree(cat catalog.Catalog, manifest workspace.Manifest, resolved artifact.Artifact) []artifact.Artifact {
	abstractID := resolved.Identity()
	if resolved.IsCapability() {
		// A capability implements an abstract of the same kind by name.
		abstractID = artifact.Identity{Kind: resolved.Kind, Name: resolved.Implements}
	}
	abstract, ok := cat.Find(abstractID)
	if !ok || !abstract.IsAbstract() {
		return []artifact.Artifact{resolved}
	}

	tree := []artifact.Artifact{abstract}
	seen := map[artifact.Identity]bool{abstract.Identity(): true}
	for _, capabilities := range manifestBindings(manifest)[abstract.Identity()] {
		for _, capabilityName := range capabilities {
			id := artifact.Identity{Kind: abstract.Kind, Name: capabilityName}
			if seen[id] {
				continue
			}
			seen[id] = true
			if capability, ok := cat.Find(id); ok {
				tree = append(tree, capability)
			}
		}
	}
	return tree
}

// vendorInto copies one artifact into the project's .agents and reports it.
func vendorInto(out io.Writer, projectRoot string, a artifact.Artifact) error {
	if _, _, err := vendor.Vendor(a, projectRoot); err != nil {
		return err
	}
	fmt.Fprintf(out, "Vendored %s/%s into .agents.\n", a.Kind, a.Name)
	return nil
}
