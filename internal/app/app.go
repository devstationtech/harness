// Package app wires the CLI commands to the domain: it resolves locations, loads
// the merged catalog, drives the selection TUI and persists the result.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/catalog"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/library"
	"github.com/devstationtech/harness/internal/lock"
	"github.com/devstationtech/harness/internal/source"
	"github.com/devstationtech/harness/internal/tui"
	"github.com/devstationtech/harness/internal/vendor"
	"github.com/devstationtech/harness/internal/workspace"
)

// Init seeds the shared library and reports what happened.
func Init(out io.Writer) error {
	home, err := config.SharedHome()
	if err != nil {
		return err
	}
	result, err := library.Init(home)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Shared library ready at %s\n", result.Home)
	if len(result.Created) > 0 {
		fmt.Fprintf(out, "  seeded %d file(s):\n", len(result.Created))
		for _, name := range result.Created {
			fmt.Fprintf(out, "    + %s\n", name)
		}
	}
	if len(result.Skipped) > 0 {
		fmt.Fprintf(out, "  kept %d existing file(s) untouched\n", len(result.Skipped))
	}
	fmt.Fprintln(out, "\nNext: run `harness` inside a project to select artifacts.")
	return nil
}

// List prints the merged catalog for the current project as plain text.
func List(out io.Writer) error {
	cat, _, _, err := loadCatalog()
	if err != nil {
		return err
	}
	artifacts := cat.All()
	if len(artifacts) == 0 {
		fmt.Fprintln(out, "No artifacts found. Run `harness init`, or add artifacts under .agents/.")
		printIssues(out, cat.Issues())
		return nil
	}
	var currentKind artifact.Kind
	first := true
	for _, a := range artifacts {
		if first || a.Kind != currentKind {
			currentKind = a.Kind
			first = false
			fmt.Fprintf(out, "\n%s\n", a.Kind.Title())
		}
		source := string(a.Source)
		if a.OverridesShared {
			source = "local (override)"
		}
		fmt.Fprintf(out, "  [ ] %s | %s | %s\n", label(a.Name, a.Version), source, a.Description)
	}
	printIssues(out, cat.Issues())
	return nil
}

// label renders an artifact as name@version, or just name when unversioned.
func label(name, version string) string {
	if version == "" {
		return name
	}
	return name + "@" + version
}

// printIssues reports artifacts that were skipped during loading, with reasons.
func printIssues(out io.Writer, issues []source.Issue) {
	if len(issues) == 0 {
		return
	}
	fmt.Fprintf(out, "\n⚠ %d artifact(s) skipped (fix and re-run):\n", len(issues))
	for _, issue := range issues {
		fmt.Fprintf(out, "  %s\n    %s\n", issue.Path, issue.Reason)
	}
}

// Run launches the interactive selection TUI and persists the chosen artifacts.
func Run(out io.Writer, version string) error {
	cat, projectRoot, home, err := loadCatalog()
	if err != nil {
		return err
	}

	manifest, err := workspace.LoadManifest(config.ManifestPath(projectRoot))
	if err != nil {
		return err
	}
	preselected := preselectedSet(manifest.Identities())

	result, err := tui.Run(cat.All(), preselected, version, len(cat.Issues()))
	if err != nil {
		return err
	}
	if !result.Confirmed {
		fmt.Fprintln(out, "No changes saved.")
		printIssues(out, cat.Issues())
		return nil
	}

	// Vendor any selections that come from a remote source, locking them, then
	// persist the manifest and AGENTS.md over the now-local set.
	resolved, entries, err := materialize(result.Selected, projectRoot, home)
	if err != nil {
		return err
	}
	if err := writeLock(projectRoot, entries); err != nil {
		return err
	}
	if err := workspace.Apply(projectRoot, resolved); err != nil {
		return err
	}
	printSaveSummary(out, projectRoot, resolved)
	printIssues(out, cat.Issues())
	return nil
}

// materialize vendors every selection that comes from a remote source into the
// project, returning the selection as it now lives locally plus the lock entries
// that pin the vendored artifacts. Local and shared selections pass through
// untouched (they are referenced in place).
func materialize(selected []artifact.Artifact, projectRoot, home string) ([]artifact.Artifact, []lock.Entry, error) {
	remotes, err := config.LoadSources(config.SourcesConfigPath(home))
	if err != nil {
		return nil, nil, err
	}
	commits := make(map[string]string)
	final := make([]artifact.Artifact, 0, len(selected))
	var entries []lock.Entry

	for _, a := range selected {
		remote, isRemote := remotes.Find(a.Origin)
		if !isRemote {
			final = append(final, a)
			continue
		}
		commit, known := commits[a.Origin]
		if !known {
			commit = resolveCommit(home, remote)
			commits[a.Origin] = commit
		}
		vendored, entry, err := vendor.Vendor(a, projectRoot, commit)
		if err != nil {
			return nil, nil, err
		}
		final = append(final, vendored)
		entries = append(entries, entry)
	}
	return final, entries, nil
}

// resolveCommit reports the checked-out commit of a remote source for
// provenance. It is best-effort: an empty string is recorded if git cannot be
// queried.
func resolveCommit(home string, s config.SourceConfig) string {
	repo := source.NewGitRepository(s.Name, s.URL, s.Ref, config.SourceCloneDir(home, s.Name), artifact.SourceShared)
	commit, err := repo.Commit(context.Background())
	if err != nil {
		return ""
	}
	return commit
}

// writeLock persists the lock entries for a project, or removes a stale lockfile
// when nothing remote is vendored.
func writeLock(projectRoot string, entries []lock.Entry) error {
	path := config.LockPath(projectRoot)
	if len(entries) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Name < entries[j].Name
	})
	return lock.New(entries).Save(path)
}

// loadCatalog resolves and merges every source for the current project and
// returns the catalog, the project root, and the shared home.
func loadCatalog() (catalog.Catalog, string, string, error) {
	home, err := config.SharedHome()
	if err != nil {
		return catalog.Catalog{}, "", "", err
	}
	projectRoot, err := config.ProjectRoot()
	if err != nil {
		return catalog.Catalog{}, "", "", err
	}
	sources, err := projectSources(home, projectRoot)
	if err != nil {
		return catalog.Catalog{}, "", "", err
	}
	cat, err := catalog.Load(sources...)
	if err != nil {
		return catalog.Catalog{}, "", "", err
	}
	return cat, projectRoot, home, nil
}

// projectSources builds the ordered sources for a project, highest precedence
// first: the project's own .agents, then the shared library, then each
// configured remote source. Remote sources resolve their existing working copy
// only — no network access happens here; cloning is done by `source add` and
// refreshing by `update`.
func projectSources(home, projectRoot string) ([]source.Source, error) {
	sources := []source.Source{
		source.NewLocalDirectory(source.LocalName, config.AgentsDir(projectRoot), artifact.SourceLocal),
		source.NewLocalDirectory(source.HomeName, home, artifact.SourceShared),
	}
	configured, err := config.LoadSources(config.SourcesConfigPath(home))
	if err != nil {
		return nil, err
	}
	for _, s := range configured.Sources {
		if s.Type == "git" {
			sources = append(sources, source.NewGitRepository(
				s.Name, s.URL, s.Ref, config.SourceCloneDir(home, s.Name), artifact.SourceShared,
			))
		}
	}
	return sources, nil
}

func preselectedSet(ids []artifact.Identity) map[artifact.Identity]bool {
	set := make(map[artifact.Identity]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

func printSaveSummary(out io.Writer, projectRoot string, selected []artifact.Artifact) {
	counts := map[artifact.Kind]int{}
	for _, a := range selected {
		counts[a.Kind]++
	}
	kinds := artifact.Kinds()
	sort.SliceStable(kinds, func(i, j int) bool { return kinds[i].Title() < kinds[j].Title() })

	fmt.Fprintf(out, "Saved %d artifact(s).\n", len(selected))
	for _, kind := range artifact.Kinds() {
		fmt.Fprintf(out, "  %-7s %d\n", kind.Container()+":", counts[kind])
	}
	fmt.Fprintf(
		out, "Wrote %s and %s\n",
		config.AgentsFilePath(projectRoot),
		config.ManifestPath(projectRoot),
	)
}
