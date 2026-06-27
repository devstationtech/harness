package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/vendor"
	"github.com/devstationtech/harness/internal/workspace"
)

// Vendor copies a shared or remote artifact into the project's .agents directory
// so it is committed and overrides the shared one — useful for an open-source
// project that wants specific skills available to contributors without sharing
// the maintainer's whole library. It takes a "<kind>/<name>" argument.
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
	if resolved.Source == artifact.SourceLocal {
		fmt.Fprintf(out, "%s/%s is already local.\n", kindText, name)
		return nil
	}

	if err := vendorInto(out, projectRoot, resolved); err != nil {
		return err
	}
	// Localizing an abstract pulls its bound capabilities along, so the
	// composition is complete for anyone who clones the project. The bindings
	// come from the project manifest.
	if resolved.IsAbstract() {
		manifest, err := workspace.LoadManifest(config.ManifestPath(projectRoot))
		if err != nil {
			return err
		}
		for _, capabilityName := range manifestBindings(manifest)[resolved.Identity()] {
			capability, ok := cat.Find(artifact.Identity{Kind: artifact.KindSkill, Name: capabilityName})
			if !ok || capability.Source == artifact.SourceLocal {
				continue
			}
			if err := vendorInto(out, projectRoot, capability); err != nil {
				return err
			}
		}
	}
	fmt.Fprintln(out, "It now overrides the shared one. Run `harness` (or `harness apply`) to update AGENTS.md.")
	return nil
}

// vendorInto copies one artifact into the project's .agents and reports it.
func vendorInto(out io.Writer, projectRoot string, a artifact.Artifact) error {
	if _, _, err := vendor.Vendor(a, projectRoot); err != nil {
		return err
	}
	fmt.Fprintf(out, "Vendored %s/%s into .agents.\n", a.Kind, a.Name)
	return nil
}
