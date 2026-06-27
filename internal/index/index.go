// Package index caches a lightweight, serializable projection of each source's
// artifacts under the shared home, so that searching the available artifacts is
// fast, offline, and free of any agent-token cost. One file per source means a
// source can be re-indexed or dropped independently.
package index

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devstationtech/harness/internal/source"

	"gopkg.in/yaml.v3"
)

// Record is the indexed projection of one artifact: enough to search and to
// address it, without the on-disk location.
type Record struct {
	Source      string `yaml:"source"`
	Kind        string `yaml:"kind"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// sourceFile is the on-disk shape of a single source's index.
type sourceFile struct {
	Records []Record `yaml:"records"`
}

// Refresh resolves src and writes its records as <indexDir>/<source>.yaml,
// returning how many artifacts were indexed.
func Refresh(indexDir string, src source.Source) (int, error) {
	resolved, _, err := src.Resolve()
	if err != nil {
		return 0, err
	}
	records := make([]Record, 0, len(resolved))
	for _, a := range resolved {
		records = append(records, Record{
			Source:      src.Name(),
			Kind:        string(a.Kind),
			Name:        a.Name,
			Description: a.Description,
		})
	}
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return 0, err
	}
	content, err := yaml.Marshal(sourceFile{Records: records})
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(filepath.Join(indexDir, src.Name()+".yaml"), content, 0o644); err != nil {
		return 0, err
	}
	return len(records), nil
}

// Remove deletes a source's index file. A missing file is not an error.
func Remove(indexDir, sourceName string) error {
	err := os.Remove(filepath.Join(indexDir, sourceName+".yaml"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// Search returns every indexed record whose name or description contains query
// (case-insensitive); an empty query returns all records. Results are sorted by
// source, then kind, then name. A missing index directory yields no results.
func Search(indexDir, query string) ([]Record, error) {
	entries, err := os.ReadDir(indexDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	needle := strings.ToLower(query)
	var matches []Record
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(indexDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var file sourceFile
		if err := yaml.Unmarshal(content, &file); err != nil {
			return nil, fmt.Errorf("invalid index file %s: %w", entry.Name(), err)
		}
		for _, r := range file.Records {
			if matchesQuery(r, needle) {
				matches = append(matches, r)
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Source != matches[j].Source {
			return matches[i].Source < matches[j].Source
		}
		if matches[i].Kind != matches[j].Kind {
			return matches[i].Kind < matches[j].Kind
		}
		return matches[i].Name < matches[j].Name
	})
	return matches, nil
}

func matchesQuery(r Record, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(r.Name), needle) ||
		strings.Contains(strings.ToLower(r.Description), needle)
}
