// Package lock records the reproducible resolution of a project's vendored
// artifacts: the source and commit each came from, plus a content hash that is
// stable across operating systems so the same lockfile reproduces byte-identical
// artifacts on macOS, Linux and Windows.
package lock

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const lockVersion = 1

// Lockfile is the pinned state of a project's vendored artifacts.
type Lockfile struct {
	Version   int     `yaml:"version"`
	Artifacts []Entry `yaml:"artifacts"`
}

// Entry pins one vendored artifact to its source and content.
type Entry struct {
	Kind        string `yaml:"kind"`
	Name        string `yaml:"name"`
	Source      string `yaml:"source"`
	Commit      string `yaml:"commit,omitempty"`
	ContentHash string `yaml:"contentHash"`
	Path        string `yaml:"path"` // forward-slash, relative to .agents/
}

// Load reads a lockfile from path. A missing file yields an empty lockfile, not
// an error.
func Load(path string) (Lockfile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Lockfile{Version: lockVersion}, nil
		}
		return Lockfile{}, err
	}
	var loaded Lockfile
	if err := yaml.Unmarshal(content, &loaded); err != nil {
		return Lockfile{}, fmt.Errorf("invalid lockfile %s: %w", path, err)
	}
	return loaded, nil
}

// Save writes the lockfile to path as YAML, creating the parent directory.
func (l Lockfile) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

// ContentHash returns a stable sha256 over the contents of dir. Files are hashed
// in sorted, forward-slash relative-path order, and text newlines are normalized
// (CRLF to LF), so the result is identical regardless of the host operating
// system or git checkout settings. The relative path is mixed into the digest so
// that renaming a file changes the hash. File mode is deliberately excluded, as
// the executable bit is not preserved across platforms.
func ContentHash(dir string) (string, error) {
	var relPaths []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		relPaths = append(relPaths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(relPaths)

	digest := sha256.New()
	for _, rel := range relPaths {
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
		fmt.Fprintf(digest, "%s\n%d\n", rel, len(content))
		digest.Write(content)
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil)), nil
}
