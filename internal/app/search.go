package app

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/index"
	"github.com/devstationtech/harness/internal/source"
)

// Search prints artifacts across all sources whose name or description matches
// the query, reading only the local index (offline, no token cost). If the
// index does not exist yet, it is built on demand from the currently available
// sources without touching the network.
func Search(out io.Writer, args []string) error {
	// A multi-word query is matched as one phrase (the index match is
	// substring-based), so `harness search git source` is not silently
	// truncated to its first word.
	query := strings.Join(args, " ")

	home, err := config.SharedHome()
	if err != nil {
		return err
	}
	if err := ensureIndex(home); err != nil {
		return err
	}

	records, err := index.Search(config.IndexDir(home), query)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		fmt.Fprintln(out, "No matching artifacts. Run `harness update` to refresh the index.")
		return nil
	}
	for _, r := range records {
		fmt.Fprintf(out, "  %s/%s\t%s\t%s\n", r.Source, label(r.Name, r.Version), r.Kind, r.Description)
	}
	return nil
}

// ensureIndex builds the index from the currently available sources (the shared
// library and each git source's existing clone) when it does not exist yet. It
// stays offline — it never fetches; use `update` to refresh from the network.
func ensureIndex(home string) error {
	indexDir := config.IndexDir(home)
	if _, err := os.Stat(indexDir); err == nil {
		return nil
	}
	if _, err := index.Refresh(indexDir, source.NewLocalDirectory(source.HomeName, home, artifact.SourceShared)); err != nil {
		return err
	}
	configured, err := config.LoadSources(config.SourcesPath(home))
	if err != nil {
		return err
	}
	for _, s := range configured.Sources {
		if s.Type != "git" {
			continue
		}
		repo := source.NewGitRepository(s.Name, s.URL, s.Ref, config.SourceCloneDir(home, s.Name), artifact.SourceShared)
		if _, err := index.Refresh(indexDir, repo); err != nil {
			return err
		}
	}
	return nil
}
