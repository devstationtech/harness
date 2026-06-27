package source

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/devstationtech/harness/internal/artifact"
)

// LocalDirectory resolves artifacts from a base directory laid out with the
// standard kind containers (skills/, rules/, agents/). It backs both the shared
// library (~/.harness) and a project's .agents directory.
type LocalDirectory struct {
	name string
	base string
	tag  artifact.Source
}

// NewLocalDirectory returns a source over base, tagging every artifact it finds
// with tag. An empty or non-existent base is valid and resolves to nothing.
func NewLocalDirectory(name, base string, tag artifact.Source) LocalDirectory {
	return LocalDirectory{name: name, base: base, tag: tag}
}

// Name reports the source identifier.
func (d LocalDirectory) Name() string { return d.name }

// Resolve scans every kind container under the base directory. A directory
// without an entry document is not an artifact and is ignored silently; a
// directory whose entry document is malformed is reported as an Issue.
func (d LocalDirectory) Resolve() ([]artifact.Artifact, []Issue, error) {
	if d.base == "" {
		return nil, nil, nil
	}
	var found []artifact.Artifact
	var issues []Issue
	for _, kind := range artifact.Kinds() {
		container := filepath.Join(d.base, kind.Container())
		entries, err := os.ReadDir(container)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			directory := filepath.Join(container, entry.Name())
			found, issues = d.collect(kind, entry.Name(), directory, found, issues)
		}
	}
	return found, issues, nil
}

// collect appends the artifact at directory to found, or records an Issue, or
// (for a directory that is not an artifact) does neither.
func (d LocalDirectory) collect(kind artifact.Kind, dirName, directory string, found []artifact.Artifact, issues []Issue) ([]artifact.Artifact, []Issue) {
	a, err := d.read(kind, dirName, directory)
	switch {
	case err == nil:
		return append(found, a), issues
	case errors.Is(err, fs.ErrNotExist):
		// No entry document: not an artifact directory. Ignore.
		return found, issues
	default:
		return found, append(issues, Issue{
			Path:   artifact.EntryFileFor(kind, directory),
			Reason: err.Error(),
		})
	}
}

// read loads and validates a single artifact directory.
func (d LocalDirectory) read(kind artifact.Kind, dirName, directory string) (artifact.Artifact, error) {
	entryPath := artifact.EntryFileFor(kind, directory)
	content, err := os.ReadFile(entryPath)
	if err != nil {
		return artifact.Artifact{}, err
	}
	front, _, err := artifact.ParseDocument(content)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if err := front.Validate(dirName); err != nil {
		return artifact.Artifact{}, err
	}
	return artifact.Artifact{
		Kind:        kind,
		Name:        front.Name,
		Description: front.Description,
		Source:      d.tag,
		Origin:      d.name,
		Directory:   directory,
		EntryPath:   entryPath,
		Metadata:    front.Metadata,
	}, nil
}
