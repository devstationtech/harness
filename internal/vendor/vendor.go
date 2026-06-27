// Package vendor materializes a remote artifact into a project: it copies the
// artifact's directory under .agents/<container>/<name>/ and produces the lock
// entry that pins exactly what was copied. Local and shared artifacts are not
// vendored — they are referenced in place; only artifacts resolved from a remote
// source are copied so the project is reproducible without that source.
package vendor

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/lock"
	"github.com/devstationtech/harness/internal/source"
)

// Vendor copies a's directory into the project's .agents tree and returns the
// artifact as it now lives locally, together with the lock entry that records
// where it came from and a content hash of the copy. commit is the source's
// resolved commit, recorded for provenance (may be empty). Vendoring is
// idempotent: the destination is replaced wholesale, so the content hash is
// stable across repeated runs of the same source revision.
func Vendor(a artifact.Artifact, projectRoot, commit string) (artifact.Artifact, lock.Entry, error) {
	container := a.Kind.Container()
	dest := filepath.Join(config.AgentsDir(projectRoot), container, a.Name)

	if err := os.RemoveAll(dest); err != nil {
		return artifact.Artifact{}, lock.Entry{}, err
	}
	if err := copyTree(a.Directory, dest); err != nil {
		return artifact.Artifact{}, lock.Entry{}, err
	}
	hash, err := lock.ContentHash(dest)
	if err != nil {
		return artifact.Artifact{}, lock.Entry{}, err
	}

	entry := lock.Entry{
		Kind:        string(a.Kind),
		Name:        a.Name,
		Source:      a.Origin,
		Commit:      commit,
		ContentHash: hash,
		Path:        container + "/" + a.Name,
	}

	// The artifact now lives locally; later runs discover it under .agents.
	vendored := a
	vendored.Source = artifact.SourceLocal
	vendored.Origin = source.LocalName
	vendored.Directory = dest
	vendored.EntryPath = artifact.EntryFileFor(a.Kind, dest)
	return vendored, entry, nil
}

// copyTree recursively copies the directory at src to dst, preserving file
// permissions. It does not follow symlinks.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

// copyFile copies a single file, creating parent directories as needed.
func copyFile(src, dst string, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
