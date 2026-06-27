package source

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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

// Resolve reads the artifacts under the base directory. When the base contains
// a package manifest (harness.artifacts.yaml) it is authoritative — artifacts
// are resolved from its entries, with versions. Otherwise the base is scanned by
// directory convention (skills/, rules/, agents/) and artifacts are unversioned.
func (d LocalDirectory) Resolve() ([]artifact.Artifact, []Issue, error) {
	if d.base == "" {
		return nil, nil, nil
	}
	manifest, present, err := LoadArtifactsManifest(filepath.Join(d.base, ArtifactsManifestFile))
	if err != nil {
		return nil, nil, err
	}
	if present {
		found, issues := d.resolveFromManifest(manifest)
		return found, issues, nil
	}
	return d.resolveByConvention()
}

// resolveByConvention scans every kind container under the base. A directory
// without an entry document is not an artifact and is ignored silently; a
// directory whose entry document is malformed is reported as an Issue.
func (d LocalDirectory) resolveByConvention() ([]artifact.Artifact, []Issue, error) {
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

// resolveFromManifest resolves exactly the artifacts the package manifest
// lists. A bad entry (unknown kind, invalid version, escaping path, name
// mismatch, missing document, duplicate) is reported as an Issue and skipped.
func (d LocalDirectory) resolveFromManifest(manifest ArtifactsManifest) ([]artifact.Artifact, []Issue) {
	var found []artifact.Artifact
	var issues []Issue
	seen := make(map[artifact.Identity]bool)
	for _, entry := range manifest.Artifacts {
		a, issue := d.resolveEntry(entry, seen)
		if issue != nil {
			issues = append(issues, *issue)
			continue
		}
		found = append(found, a)
	}
	return found, issues
}

// resolveEntry validates and loads one manifest entry.
func (d LocalDirectory) resolveEntry(entry ArtifactEntry, seen map[artifact.Identity]bool) (artifact.Artifact, *Issue) {
	kind, ok := artifact.ParseKind(entry.Kind)
	if !ok {
		return artifact.Artifact{}, manifestIssue(entry, fmt.Sprintf("unknown kind %q", entry.Kind))
	}
	if entry.Version != "" {
		if err := artifact.ValidateVersion(entry.Version); err != nil {
			return artifact.Artifact{}, manifestIssue(entry, err.Error())
		}
	}
	directory, ok := d.resolveWithinBase(entry.Path)
	if !ok {
		return artifact.Artifact{}, manifestIssue(entry, fmt.Sprintf("path %q escapes the source", entry.Path))
	}
	id := artifact.Identity{Kind: kind, Name: entry.Name}
	if seen[id] {
		return artifact.Artifact{}, manifestIssue(entry, "duplicate entry")
	}
	seen[id] = true

	a, err := d.read(kind, entry.Name, directory)
	if err != nil {
		return artifact.Artifact{}, &Issue{Path: artifact.EntryFileFor(kind, directory), Reason: err.Error()}
	}
	a.Version = entry.Version
	return a, nil
}

// resolveWithinBase converts a forward-slash manifest path to an absolute
// directory, rejecting anything that escapes the source base.
func (d LocalDirectory) resolveWithinBase(path string) (string, bool) {
	directory := filepath.Join(d.base, filepath.FromSlash(path))
	within, err := filepath.Rel(d.base, directory)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return "", false
	}
	return directory, true
}

// manifestIssue builds an Issue attributed to the package manifest for a bad
// entry, mentioning the kind and name.
func manifestIssue(entry ArtifactEntry, reason string) *Issue {
	return &Issue{
		Path:   ArtifactsManifestFile,
		Reason: fmt.Sprintf("%s %q: %s", entry.Kind, entry.Name, reason),
	}
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
