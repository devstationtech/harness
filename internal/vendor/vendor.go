// Package vendor materializes a remote artifact into a project: it copies the
// artifact's directory under .agents/<container>/<name>/ and returns a content
// digest of the copy. Local and shared artifacts are not vendored — they are
// referenced in place; only artifacts resolved from a remote source are copied
// so the project is reproducible without that source.
package vendor

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/config"
	"github.com/devstationtech/harness/internal/source"
)

// Vendor copies a's directory into the project's .agents tree and returns the
// artifact as it now lives locally, together with a content digest of the copy.
// Vendoring is idempotent: the destination is replaced wholesale, so the digest
// is stable across repeated runs of the same source revision.
func Vendor(a artifact.Artifact, projectRoot string) (artifact.Artifact, string, error) {
	container := a.Kind.Container()
	dest := filepath.Join(config.AgentsDir(projectRoot), container, a.Name)

	if err := os.RemoveAll(dest); err != nil {
		return artifact.Artifact{}, "", err
	}
	if err := copyTree(a.Directory, dest); err != nil {
		return artifact.Artifact{}, "", err
	}
	digest, err := ContentHash(dest)
	if err != nil {
		return artifact.Artifact{}, "", err
	}

	// The artifact now lives locally; later runs discover it under .agents.
	vendored := a
	vendored.Source = artifact.SourceLocal
	vendored.Origin = source.LocalName
	vendored.Directory = dest
	vendored.EntryPath = artifact.EntryFileFor(a.Kind, dest)
	return vendored, digest, nil
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
