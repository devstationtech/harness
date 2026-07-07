package app

import (
	"context"
	"fmt"
	"io"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/index"
	"github.com/devstationtech/harness/internal/source"
)

// Update refreshes every configured source and rebuilds the offline search
// index. Git sources are fetched from the network; a source that cannot be
// refreshed keeps its previously indexed records rather than aborting the run.
func Update(out io.Writer) error {
	home, err := config.SharedHome()
	if err != nil {
		return err
	}
	indexDir := config.IndexDir(home)
	ctx := context.Background()

	homeSource := source.NewLocalDirectory(source.HomeName, home, artifact.SourceShared)
	count, err := index.Refresh(indexDir, homeSource)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "  %-24s %d artifact(s)\n", source.HomeName, count)

	configured, err := config.LoadSources(config.SourcesPath(home))
	if err != nil {
		return err
	}
	for _, s := range configured.Sources {
		if s.Type != "git" {
			continue
		}
		repo := source.NewGitRepository(s.Name, s.URL, s.Ref, config.SourceCloneDir(home, s.Name), artifact.SourceShared)
		if err := repo.Sync(ctx); err != nil {
			fmt.Fprintf(out, "  %-24s could not refresh (%v); keeping cached index\n", s.Name, err)
			continue
		}
		count, err := index.Refresh(indexDir, repo)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "  %-24s %d artifact(s)\n", s.Name, count)
	}

	fmt.Fprintln(out, "Index updated.")
	return nil
}
