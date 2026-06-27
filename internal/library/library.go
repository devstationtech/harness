// Package library manages the shared artifact library in the user's home
// (~/.harness): creating its structure and seeding it with starter artifacts.
package library

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/assets"
)

const (
	dirPermission  = 0o755
	filePermission = 0o644
)

// InitResult reports what Init did, so callers can give precise feedback.
type InitResult struct {
	Home    string
	Created []string // files written from seeds
	Skipped []string // files left untouched because they already existed
}

// Init ensures the shared library exists at home and seeds it with the embedded
// starter artifacts. Existing files are never overwritten, so Init is safe to run
// repeatedly.
func Init(home string) (InitResult, error) {
	result := InitResult{Home: home}

	if err := ensureContainers(home); err != nil {
		return result, err
	}

	err := fs.WalkDir(assets.SeedFS, assets.SeedRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative := strings.TrimPrefix(path, assets.SeedRoot+"/")
		if relative == assets.SeedRoot || relative == "" {
			return nil
		}
		destination := filepath.Join(home, filepath.FromSlash(relative))

		if entry.IsDir() {
			return os.MkdirAll(destination, dirPermission)
		}

		if _, statErr := os.Stat(destination); statErr == nil {
			result.Skipped = append(result.Skipped, relative)
			return nil
		}

		content, readErr := assets.SeedFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if mkErr := os.MkdirAll(filepath.Dir(destination), dirPermission); mkErr != nil {
			return mkErr
		}
		if writeErr := os.WriteFile(destination, content, filePermission); writeErr != nil {
			return writeErr
		}
		result.Created = append(result.Created, relative)
		return nil
	})
	return result, err
}

// ensureContainers creates the kind directories under home even when seeds for a
// kind are absent, so the library always has a complete structure.
func ensureContainers(home string) error {
	if err := os.MkdirAll(home, dirPermission); err != nil {
		return err
	}
	for _, kind := range artifact.Kinds() {
		if err := os.MkdirAll(filepath.Join(home, kind.Container()), dirPermission); err != nil {
			return err
		}
	}
	return nil
}
