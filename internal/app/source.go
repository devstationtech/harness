package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/index"
	"github.com/devstationtech/harness/internal/source"
)

// Source dispatches the `harness source` subcommands that manage where
// artifacts come from.
func Source(out io.Writer, args []string) error {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "add":
		return sourceAdd(out, args[1:])
	case "list", "ls":
		return sourceList(out)
	case "remove", "rm":
		return sourceRemove(out, args[1:])
	default:
		return fmt.Errorf("usage: harness source <add|list|remove>")
	}
}

// sourceAdd registers a git source, clones it, and reports what it contains.
func sourceAdd(out io.Writer, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: harness source add <git-url> [--name NAME] [--ref REF]")
	}
	// The URL is the first argument (git-style); flags follow it.
	url := args[0]
	flags := flag.NewFlagSet("source add", flag.ContinueOnError)
	flags.SetOutput(out)
	name := flags.String("name", "", "source name (defaults to the repository name)")
	ref := flags.String("ref", "", "branch or tag to track (defaults to the repository default)")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	sourceName := *name
	if sourceName == "" {
		sourceName = deriveName(url)
	}
	if err := artifact.ValidateName(sourceName); err != nil {
		return fmt.Errorf("source name %q is invalid: %w; pass --name", sourceName, err)
	}

	home, err := config.SharedHome()
	if err != nil {
		return err
	}
	configPath := config.SourcesConfigPath(home)
	sources, err := config.LoadSources(configPath)
	if err != nil {
		return err
	}
	if _, exists := sources.Find(sourceName); exists {
		return fmt.Errorf("a source named %q already exists", sourceName)
	}

	repo := source.NewGitRepository(sourceName, url, *ref, config.SourceCloneDir(home, sourceName), artifact.SourceShared)
	fmt.Fprintf(out, "Cloning %s …\n", url)
	if err := repo.Sync(context.Background()); err != nil {
		return err
	}

	sources.Sources = append(sources.Sources, config.SourceConfig{
		Name: sourceName, Type: "git", URL: url, Ref: *ref,
	})
	if err := sources.Save(configPath); err != nil {
		return err
	}

	resolved, _, err := repo.Resolve()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Added source %q with %d artifact(s). Run `harness` to select them.\n", sourceName, len(resolved))
	return nil
}

// sourceList prints the configured sources.
func sourceList(out io.Writer) error {
	home, err := config.SharedHome()
	if err != nil {
		return err
	}
	sources, err := config.LoadSources(config.SourcesConfigPath(home))
	if err != nil {
		return err
	}
	if len(sources.Sources) == 0 {
		fmt.Fprintln(out, "No sources configured. Add one with `harness source add <git-url>`.")
		return nil
	}
	for _, s := range sources.Sources {
		ref := s.Ref
		if ref == "" {
			ref = "(default)"
		}
		fmt.Fprintf(out, "  %s\t%s\t%s\t%s\n", s.Name, s.Type, s.URL, ref)
	}
	return nil
}

// sourceRemove drops a source and its working copy, leaving any artifacts a
// project already vendored from it untouched.
func sourceRemove(out io.Writer, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: harness source remove <name>")
	}
	name := args[0]

	home, err := config.SharedHome()
	if err != nil {
		return err
	}
	configPath := config.SourcesConfigPath(home)
	sources, err := config.LoadSources(configPath)
	if err != nil {
		return err
	}
	if _, ok := sources.Find(name); !ok {
		return fmt.Errorf("no source named %q", name)
	}

	kept := sources.Sources[:0]
	for _, s := range sources.Sources {
		if s.Name != name {
			kept = append(kept, s)
		}
	}
	sources.Sources = kept
	if err := sources.Save(configPath); err != nil {
		return err
	}
	if err := os.RemoveAll(config.SourceCloneDir(home, name)); err != nil {
		return err
	}
	if err := index.Remove(config.IndexDir(home), name); err != nil {
		return err
	}
	fmt.Fprintf(out, "Removed source %q. Artifacts already vendored into projects are left untouched.\n", name)
	return nil
}

// deriveName extracts a source name from a clone URL: the final path segment
// without a trailing ".git".
func deriveName(url string) string {
	trimmed := strings.TrimRight(strings.TrimSuffix(url, ".git"), "/")
	if index := strings.LastIndexAny(trimmed, "/:"); index >= 0 {
		trimmed = trimmed[index+1:]
	}
	return trimmed
}
