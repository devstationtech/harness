// Package app wires the CLI commands to the domain: it resolves locations, loads
// the merged catalog, drives the selection TUI and persists the result.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/catalog"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/library"
	"github.com/devstationtech/harness/internal/selfupdate"
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
	priorBindings := manifestBindings(manifest)

	result, err := tui.Run(cat.All(), preselected, priorBindings, version, len(cat.Issues()), updateChecker(version))
	if err != nil {
		return err
	}
	if result.RequestUpdate {
		return performSelfUpdate(out, version)
	}
	if !result.Confirmed {
		fmt.Fprintln(out, "No changes saved.")
		printIssues(out, cat.Issues())
		return nil
	}

	// Vendor any selections that come from a remote source or that the user
	// asked to localize (an abstract pulls its bound capabilities along), then
	// persist the root manifest and AGENTS.md.
	localized := expandLocalized(result.Localized, result.Bindings)
	resolved, digests, err := materialize(result.Selected, projectRoot, home, localized)
	if err != nil {
		return err
	}
	if err := workspace.Apply(projectRoot, resolved, digests, result.Bindings); err != nil {
		return err
	}
	printSaveSummary(out, projectRoot, resolved)
	printIssues(out, cat.Issues())
	return nil
}

// updateChecker returns a non-blocking probe for a newer release, used by the
// selection TUI's footer. It is disabled when HARNESS_NO_UPDATE_CHECK is set
// (CI, offline, scripted use) and fails silently otherwise.
func updateChecker(version string) func() (string, bool) {
	if os.Getenv("HARNESS_NO_UPDATE_CHECK") != "" {
		return nil
	}
	return func() (string, bool) {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		return selfupdate.New(version).Available(ctx)
	}
}

// SelfUpdate downloads and installs the latest release over the running binary.
func SelfUpdate(out io.Writer, version string) error {
	newVersion, err := selfupdate.New(version).Update(context.Background(), out)
	if errors.Is(err, selfupdate.ErrUpToDate) {
		fmt.Fprintf(out, "harness %s is already the latest version.\n", version)
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Updated to %s. Re-run harness to use the new version.\n", newVersion)
	return nil
}

// performSelfUpdate applies an update requested from the selection TUI, then
// relaunches so the user lands in the new version (close and reopen updated).
func performSelfUpdate(out io.Writer, version string) error {
	newVersion, err := selfupdate.New(version).Update(context.Background(), out)
	if errors.Is(err, selfupdate.ErrUpToDate) {
		fmt.Fprintln(out, "Already on the latest version.")
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Updated to %s — relaunching …\n", newVersion)
	return selfupdate.Relaunch()
}

// materialize vendors every selection that comes from a remote source into the
// project, returning the selection as it now lives locally plus the content
// digest of each vendored artifact (keyed by identity). Local and shared
// selections pass through untouched (they are referenced in place).
func materialize(selected []artifact.Artifact, projectRoot, home string, localized map[artifact.Identity]bool) ([]artifact.Artifact, map[artifact.Identity]string, error) {
	remotes, err := config.LoadSources(config.SourcesPath(home))
	if err != nil {
		return nil, nil, err
	}
	digests := make(map[artifact.Identity]string)
	final := make([]artifact.Artifact, 0, len(selected))

	for _, a := range selected {
		if a.Source == artifact.SourceLocal {
			final = append(final, a) // already under .agents; never re-copy onto itself
			continue
		}
		_, isRemote := remotes.Find(a.Origin)
		if !isRemote && !localized[a.Identity()] {
			final = append(final, a)
			continue
		}
		vendored, digest, err := vendor.Vendor(a, projectRoot)
		if err != nil {
			return nil, nil, err
		}
		final = append(final, vendored)
		digests[vendored.Identity()] = digest
	}
	return final, digests, nil
}

// expandLocalized turns the user's localize requests into the full set to
// vendor: localizing an abstract artifact also localizes the capabilities its
// bindings point to, so the composition is complete for anyone who clones the
// project. A capability shares the abstract's kind.
func expandLocalized(requested []artifact.Identity, bindings map[artifact.Identity]map[string][]string) map[artifact.Identity]bool {
	set := make(map[artifact.Identity]bool, len(requested))
	for _, id := range requested {
		set[id] = true
		for _, capabilities := range bindings[id] {
			for _, capability := range capabilities {
				set[artifact.Identity{Kind: id.Kind, Name: capability}] = true
			}
		}
	}
	return set
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
	configured, err := config.LoadSources(config.SourcesPath(home))
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

// manifestBindings extracts the recorded contract→capability bindings per
// abstract artifact from a manifest, keyed by identity.
func manifestBindings(manifest workspace.Manifest) map[artifact.Identity]map[string][]string {
	out := make(map[artifact.Identity]map[string][]string)
	for _, selection := range manifest.Selections {
		if bound := selection.BindingsAsMap(); len(bound) > 0 {
			out[artifact.Identity{Kind: selection.Kind, Name: selection.Name}] = bound
		}
	}
	return out
}

func printSaveSummary(out io.Writer, projectRoot string, selected []artifact.Artifact) {
	counts := map[artifact.Kind]int{}
	for _, a := range selected {
		counts[a.Kind]++
	}
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
