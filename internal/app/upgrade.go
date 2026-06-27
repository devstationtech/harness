package app

import (
	"context"
	"fmt"
	"io"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/lock"
	"github.com/devstationtech/harness/internal/source"
	"github.com/devstationtech/harness/internal/vendor"
)

// Upgrade re-resolves a project's locked artifacts against the current ref of
// each source, re-vendoring those whose content changed and rewriting the
// lockfile. It reports which artifacts changed. An artifact whose source is no
// longer configured, or that has disappeared from its source, is left as-is.
func Upgrade(out io.Writer) error {
	home, err := config.SharedHome()
	if err != nil {
		return err
	}
	projectRoot, err := config.ProjectRoot()
	if err != nil {
		return err
	}

	current, err := lock.Load(config.LockPath(projectRoot))
	if err != nil {
		return err
	}
	if len(current.Artifacts) == 0 {
		fmt.Fprintln(out, "Nothing locked to upgrade.")
		return nil
	}
	configured, err := config.LoadSources(config.SourcesConfigPath(home))
	if err != nil {
		return err
	}

	ctx := context.Background()
	commits := make(map[string]string) // source name -> commit, set once synced
	updated := make([]lock.Entry, 0, len(current.Artifacts))
	changed := 0

	for _, entry := range current.Artifacts {
		s, ok := configured.Find(entry.Source)
		if !ok {
			fmt.Fprintf(out, "  %s/%s: source not configured; left unchanged\n", entry.Source, entry.Name)
			updated = append(updated, entry)
			continue
		}
		repo := source.NewGitRepository(s.Name, s.URL, s.Ref, config.SourceCloneDir(home, s.Name), artifact.SourceShared)
		commit, synced := commits[s.Name]
		if !synced {
			if err := repo.Sync(ctx); err != nil {
				return fmt.Errorf("refresh %s: %w", s.Name, err)
			}
			commit, _ = repo.Commit(ctx)
			commits[s.Name] = commit
		}

		resolved, _, err := repo.Resolve()
		if err != nil {
			return err
		}
		found, ok := findArtifact(resolved, entry.Kind, entry.Name)
		if !ok {
			fmt.Fprintf(out, "  %s/%s: no longer in source; left unchanged\n", entry.Source, entry.Name)
			updated = append(updated, entry)
			continue
		}

		_, newEntry, err := vendor.Vendor(found, projectRoot, commit)
		if err != nil {
			return err
		}
		if newEntry.ContentHash != entry.ContentHash {
			changed++
			fmt.Fprintf(out, "  %s/%s: updated %s → %s\n",
				entry.Source, entry.Name, shortCommit(entry.Commit), shortCommit(newEntry.Commit))
		}
		updated = append(updated, newEntry)
	}

	if err := writeLock(projectRoot, updated); err != nil {
		return err
	}
	fmt.Fprintf(out, "Upgrade complete: %d changed of %d locked.\n", changed, len(updated))
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

// shortCommit abbreviates a commit SHA for human-readable output.
func shortCommit(commit string) string {
	switch {
	case commit == "":
		return "(none)"
	case len(commit) > 7:
		return commit[:7]
	default:
		return commit
	}
}
