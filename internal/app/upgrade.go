package app

import (
	"context"
	"fmt"
	"io"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/source"
	"github.com/devstationtech/harness/internal/vendor"
	"github.com/devstationtech/harness/internal/workspace"
)

// Upgrade re-resolves a project's remote selections against the current ref of
// each source, re-vendoring those whose content changed and rewriting the root
// manifest with new versions and digests. It reports the version transitions. A
// selection whose source is no longer configured, or that has disappeared from
// its source, is left as-is.
func Upgrade(out io.Writer) error {
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
		fmt.Fprintln(out, "Nothing to upgrade.")
		return nil
	}
	configured, err := config.LoadSources(config.SourcesPath(home))
	if err != nil {
		return err
	}

	ctx := context.Background()
	synced := make(map[string]bool)
	updated := make([]workspace.Selection, 0, len(manifest.Selections))
	changed := 0

	for _, sel := range manifest.Selections {
		s, ok := configured.Find(sel.Source)
		if !ok {
			// Local, shared, or a removed source: nothing to re-resolve.
			updated = append(updated, sel)
			continue
		}
		repo := source.NewGitRepository(s.Name, s.URL, s.Ref, config.SourceCloneDir(home, s.Name), artifact.SourceShared)
		if !synced[s.Name] {
			if err := repo.Sync(ctx); err != nil {
				return fmt.Errorf("refresh %s: %w", s.Name, err)
			}
			synced[s.Name] = true
		}

		resolved, _, err := repo.Resolve()
		if err != nil {
			return err
		}
		found, ok := findArtifact(resolved, string(sel.Kind), sel.Name)
		if !ok {
			fmt.Fprintf(out, "  %s/%s: no longer in source; left unchanged\n", sel.Source, sel.Name)
			updated = append(updated, sel)
			continue
		}

		_, digest, err := vendor.Vendor(found, projectRoot)
		if err != nil {
			return err
		}
		next := workspace.SelectionOf(found, digest)
		next.Bindings = sel.Bindings // preserve composition bindings across upgrade
		if next.Digest != sel.Digest {
			changed++
			fmt.Fprintf(out, "  %s/%s: %s → %s\n", sel.Source, sel.Name, versionLabel(sel.Version), versionLabel(next.Version))
		}
		updated = append(updated, next)
	}

	if err := workspace.NewManifest(updated).Save(config.ManifestPath(projectRoot)); err != nil {
		return err
	}
	fmt.Fprintf(out, "Upgrade complete: %d changed of %d.\n", changed, len(updated))
	return nil
}

// findArtifact returns the resolved artifact matching kind and name, if present.
func findArtifact(artifacts []artifact.Artifact, kind, name string) (artifact.Artifact, bool) {
	for _, a := range artifacts {
		if string(a.Kind) == kind && a.Name == name {
			return a, true
		}
	}
	return artifact.Artifact{}, false
}

// versionLabel renders a version for reporting, naming the unversioned case.
func versionLabel(version string) string {
	if version == "" {
		return "unversioned"
	}
	return version
}
