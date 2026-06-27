// Package source defines where artifacts come from. A Source is a port with one
// implementation per kind of location — a local directory today, a git
// repository next — and the catalog merges an ordered list of them. Generalizing
// the shared/local split into this port is the foundation for external and
// private artifact repositories.
package source

import "github.com/devstationtech/harness/internal/artifact"

// Source resolves the artifacts available at one location. Implementations are
// constructed ready to use (see NewLocalDirectory) and carry no global state.
type Source interface {
	// Name is the stable identifier of this source ("local", "home", or a
	// configured remote name). It namespaces artifacts across sources.
	Name() string
	// Resolve reads every artifact at this source. A directory that looks like
	// an artifact but cannot be loaded is returned as an Issue rather than an
	// error, so one bad entry neither aborts the scan nor disappears silently.
	// A genuine I/O failure (an unreadable container directory) is returned as
	// the error.
	Resolve() ([]artifact.Artifact, []Issue, error)
}

// Issue records a directory that looked like an artifact (it has an entry
// document) but could not be loaded, so it was skipped and surfaced to the user.
type Issue struct {
	Path   string // path to the offending entry document
	Reason string // why it was skipped
}
