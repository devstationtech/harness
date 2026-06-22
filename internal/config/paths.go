// Package config resolves the filesystem locations harness works with: the
// shared library in the user's home and the per-project .agents directory.
package config

import (
	"os"
	"path/filepath"
)

const (
	// SharedHomeEnv overrides the default shared library location. Useful for
	// tests and for users who keep their library outside $HOME.
	SharedHomeEnv = "HARNESS_HOME"

	// sharedDirName is the default shared library directory under $HOME.
	sharedDirName = ".harness"

	// AgentsDirName is the per-project directory that mirrors the shared
	// library structure and holds project-local artifacts and configuration.
	AgentsDirName = ".agents"

	// ManifestFileName records which artifacts are active in a project.
	ManifestFileName = "harness.yaml"

	// AgentsFileName is the generated entry point read by AI agents.
	AgentsFileName = "AGENTS.md"

	// SpecsDirName holds spec-driven development artifacts, one directory per
	// spec, under the project's .agents directory.
	SpecsDirName = "specs"
)

// SharedHome returns the absolute path to the shared library (~/.harness),
// honoring the HARNESS_HOME override.
func SharedHome() (string, error) {
	if override := os.Getenv(SharedHomeEnv); override != "" {
		return filepath.Abs(override)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, sharedDirName), nil
}

// ProjectRoot returns the directory harness was invoked from; this is the root
// that will hold AGENTS.md and the .agents directory.
func ProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Abs(cwd)
}

// AgentsDir returns the project's .agents directory for a given root.
func AgentsDir(projectRoot string) string {
	return filepath.Join(projectRoot, AgentsDirName)
}

// ManifestPath returns the path to the project manifest.
func ManifestPath(projectRoot string) string {
	return filepath.Join(AgentsDir(projectRoot), ManifestFileName)
}

// AgentsFilePath returns the path to the generated AGENTS.md.
func AgentsFilePath(projectRoot string) string {
	return filepath.Join(projectRoot, AgentsFileName)
}

// SpecsDir returns the project's specs directory under .agents.
func SpecsDir(projectRoot string) string {
	return filepath.Join(AgentsDir(projectRoot), SpecsDirName)
}
