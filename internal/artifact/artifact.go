// Package artifact defines the core domain of harness: the reusable units
// (skills, rules and agents) that are authored once and shared across projects.
//
// Every artifact, regardless of kind, follows the same on-disk convention,
// adapted from the Agent Skills specification (https://agentskills.io):
//
//	<container>/<name>/<ENTRY>.md   (+ optional scripts/ references/ assets/)
//
// where <container> is the plural kind directory (skills, rules, agents) and
// <ENTRY> is the canonical document for that kind (SKILL, RULE, AGENT).
package artifact

import "path/filepath"

// Kind is the category of an artifact. Each kind shares the same directory and
// document conventions, but plays a different role in the generated AGENTS.md.
type Kind string

const (
	// KindSkill is an on-demand capability: a reusable procedure loaded only
	// when a task matches its description (progressive disclosure).
	KindSkill Kind = "skill"
	// KindRule is a project invariant: a non-negotiable constraint that agents
	// must always load and respect.
	KindRule Kind = "rule"
	// KindAgent is a specialized executor that work can be delegated to.
	KindAgent Kind = "agent"
)

// Kinds returns every supported kind in canonical display order.
func Kinds() []Kind { return []Kind{KindRule, KindSkill, KindAgent} }

// Container is the plural directory that holds artifacts of this kind, e.g.
// "skills". It is identical under both the shared library and a project.
func (kind Kind) Container() string {
	switch kind {
	case KindSkill:
		return "skills"
	case KindRule:
		return "rules"
	case KindAgent:
		return "agents"
	default:
		return string(kind) + "s"
	}
}

// EntryFile is the canonical document name for this kind, e.g. "SKILL.md".
func (kind Kind) EntryFile() string {
	switch kind {
	case KindSkill:
		return "SKILL.md"
	case KindRule:
		return "RULE.md"
	case KindAgent:
		return "AGENT.md"
	default:
		return "ARTIFACT.md"
	}
}

// Title is the human-facing label for this kind, e.g. "Skills".
func (kind Kind) Title() string {
	switch kind {
	case KindSkill:
		return "Skills"
	case KindRule:
		return "Rules"
	case KindAgent:
		return "Agents"
	default:
		return string(kind)
	}
}

// Source records where a resolved artifact came from: the user's shared library
// (~/.harness) or the current project (.agents).
type Source string

const (
	// SourceShared means the artifact lives in the shared library (~/.harness)
	// and is reused across projects.
	SourceShared Source = "shared"
	// SourceLocal means the artifact lives in the project (.agents) and is not
	// shared with anyone else.
	SourceLocal Source = "local"
)

// Artifact is a resolved artifact discovered on disk: its identity, metadata and
// the absolute location of its directory and entry document.
type Artifact struct {
	Kind        Kind
	Name        string
	Description string
	Source      Source
	// Origin is the name of the source the artifact resolved from ("local",
	// "home", or a configured remote source name). Unlike Source, which is the
	// coarse shared/local class, Origin identifies the exact source — needed to
	// vendor and lock artifacts that come from a remote repository.
	Origin string
	// Directory is the absolute path to the artifact directory.
	Directory string
	// EntryPath is the absolute path to the entry document (SKILL.md, ...).
	EntryPath string
	// Metadata carries the optional frontmatter metadata map verbatim.
	Metadata map[string]string
	// OverridesShared is true when a local artifact shadows a shared one of the
	// same kind and name.
	OverridesShared bool
}

// Identity uniquely keys an artifact within a catalog by kind and name.
type Identity struct {
	Kind Kind
	Name string
}

// Identity returns the artifact's identity key.
func (a Artifact) Identity() Identity {
	return Identity{Kind: a.Kind, Name: a.Name}
}

// EntryFileFor returns the absolute entry-file path for an artifact directory of
// the given kind.
func EntryFileFor(kind Kind, directory string) string {
	return filepath.Join(directory, kind.EntryFile())
}
