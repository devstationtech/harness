// Package app wires the CLI commands to the domain: it resolves locations, loads
// the merged catalog, drives the selection TUI and persists the result.
package app

import (
	"fmt"
	"io"
	"sort"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/catalog"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/library"
	"github.com/devstationtech/harness/internal/source"
	"github.com/devstationtech/harness/internal/tui"
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
	cat, _, err := loadCatalog()
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
		fmt.Fprintf(out, "  [ ] %s | %s | %s\n", a.Name, source, a.Description)
	}
	printIssues(out, cat.Issues())
	return nil
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
	cat, projectRoot, err := loadCatalog()
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

	if err := workspace.Apply(projectRoot, result.Selected); err != nil {
		return err
	}
	printSaveSummary(out, projectRoot, result.Selected)
	printIssues(out, cat.Issues())
	return nil
}

func loadCatalog() (catalog.Catalog, string, error) {
	home, err := config.SharedHome()
	if err != nil {
		return catalog.Catalog{}, "", err
	}
	projectRoot, err := config.ProjectRoot()
	if err != nil {
		return catalog.Catalog{}, "", err
	}
	// Sources in precedence order, highest first: the project shadows the
	// shared library. Remote sources will append after these.
	cat, err := catalog.Load(
		source.NewLocalDirectory("local", config.AgentsDir(projectRoot), artifact.SourceLocal),
		source.NewLocalDirectory("home", home, artifact.SourceShared),
	)
	if err != nil {
		return catalog.Catalog{}, "", err
	}
	return cat, projectRoot, nil
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
