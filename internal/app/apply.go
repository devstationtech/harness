package app

import (
	"fmt"
	"io"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/catalog"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/source"
	"github.com/devstationtech/harness/internal/vendor"
	"github.com/devstationtech/harness/internal/workspace"
)

// Apply reconciles a project from its committed manifest, without the TUI: it
// resolves every selection, restores any missing vendored artifact from its
// source, verifies on-disk vendored content against the recorded digest, and
// regenerates AGENTS.md. It does not touch the network — it uses existing
// clones; run `update` first if a source has never been fetched.
func Apply(out io.Writer) error {
	home, err := config.SharedHome()
	if err != nil {
		return err
	}
	projectRoot, err := config.ProjectRoot()
	if err != nil {
		return err
	}

	manifest, err := workspace.LoadManifest(config.ManifestPath(projectRoot))
	if err != nil {
		return err
	}
	if len(manifest.Selections) == 0 {
		fmt.Fprintln(out, "Nothing to apply (the manifest has no selections).")
		return nil
	}

	sources, err := projectSources(home, projectRoot)
	if err != nil {
		return err
	}
	cat, err := catalog.Load(sources...)
	if err != nil {
		return err
	}
	configured, err := config.LoadSources(config.SourcesPath(home))
	if err != nil {
		return err
	}

	resolved := make([]artifact.Artifact, 0, len(manifest.Selections))
	digests := make(map[artifact.Identity]string)
	for _, sel := range manifest.Selections {
		a, found := cat.Find(artifact.Identity{Kind: sel.Kind, Name: sel.Name})
		if !found {
			fmt.Fprintf(out, "  %s/%s: not found in any source; skipped\n", sel.Source, sel.Name)
			continue
		}
		if _, isRemote := configured.Find(sel.Source); !isRemote {
			// Local or shared selection: referenced in place.
			resolved = append(resolved, a)
			continue
		}
		materialized, digest, err := reconcileRemote(out, sel, a, projectRoot)
		if err != nil {
			return err
		}
		resolved = append(resolved, materialized)
		digests[materialized.Identity()] = digest
	}

	if err := workspace.Apply(projectRoot, resolved, digests, manifestBindings(manifest)); err != nil {
		return err
	}
	fmt.Fprintf(out, "Applied %d selection(s).\n", len(resolved))
	return nil
}

// reconcileRemote ensures a remote selection is materialized and verified. When
// the catalog already resolved a vendored copy on disk it is verified against
// the recorded digest; otherwise it is restored from its source.
func reconcileRemote(out io.Writer, sel workspace.Selection, a artifact.Artifact, projectRoot string) (artifact.Artifact, string, error) {
	if a.Origin == source.LocalName {
		// A vendored copy already exists on disk (it won precedence).
		digest, err := vendor.ContentHash(a.Directory)
		if err != nil {
			return artifact.Artifact{}, "", err
		}
		if sel.Digest != "" && digest != sel.Digest {
			fmt.Fprintf(out, "  %s/%s: on-disk content differs from the recorded digest\n", sel.Source, sel.Name)
		}
		return a, digest, nil
	}

	// Not on disk: restore it from the source.
	vendored, digest, err := vendor.Vendor(a, projectRoot)
	if err != nil {
		return artifact.Artifact{}, "", err
	}
	if sel.Digest != "" && digest != sel.Digest {
		fmt.Fprintf(out, "  %s/%s: restored, but differs from the recorded digest\n", sel.Source, sel.Name)
	} else {
		fmt.Fprintf(out, "  %s/%s: restored\n", sel.Source, sel.Name)
	}
	return vendored, digest, nil
}
