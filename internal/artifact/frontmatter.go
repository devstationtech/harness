package artifact

import (
	"bytes"
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Frontmatter is the YAML metadata block at the top of an entry document. The
// fields mirror the Agent Skills specification so that skills authored for that
// ecosystem are valid harness artifacts unchanged.
type Frontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty"`

	// Composition fields. An abstract skill declares the contracts it needs
	// implemented; a capability declares which abstract it implements, the
	// contracts it provides, and its stack. These are harness-specific and
	// ignored by other agents (agentskills-compatible).
	Contracts  []string `yaml:"contracts,omitempty"`
	Implements string   `yaml:"implements,omitempty"`
	Provides   []string `yaml:"provides,omitempty"`
	Stack      string   `yaml:"stack,omitempty"`
}

const (
	maxNameLength        = 64
	maxDescriptionLength = 1024
)

var (
	frontmatterPattern = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n?(.*)\z`)
	namePattern        = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// ParseDocument splits an entry document into its frontmatter and body. It does
// not validate semantics beyond the YAML being well-formed; use ValidateName for
// the naming rules.
func ParseDocument(content []byte) (Frontmatter, string, error) {
	matches := frontmatterPattern.FindSubmatch(content)
	if matches == nil {
		return Frontmatter{}, "", fmt.Errorf("missing YAML frontmatter delimited by '---'")
	}

	var front Frontmatter
	if err := yaml.Unmarshal(matches[1], &front); err != nil {
		return Frontmatter{}, "", fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	body := string(bytes.TrimLeft(matches[2], "\r\n"))
	return front, body, nil
}

// ValidateName enforces the Agent Skills naming rules: 1-64 characters, lowercase
// alphanumerics and single hyphens, no leading/trailing/consecutive hyphens.
func ValidateName(name string) error {
	switch {
	case name == "":
		return fmt.Errorf("name is required")
	case len(name) > maxNameLength:
		return fmt.Errorf("name must be at most %d characters", maxNameLength)
	case !namePattern.MatchString(name):
		return fmt.Errorf("name %q must be lowercase alphanumeric with single hyphens (no leading, trailing or consecutive hyphens)", name)
	default:
		return nil
	}
}

// Validate checks a frontmatter block against the required fields and naming
// rules. expectedName, when non-empty, must equal the frontmatter name (the spec
// requires name to match the parent directory).
func (f Frontmatter) Validate(expectedName string) error {
	if err := ValidateName(f.Name); err != nil {
		return err
	}
	if expectedName != "" && f.Name != expectedName {
		return fmt.Errorf("name %q must match the directory name %q", f.Name, expectedName)
	}
	switch {
	case f.Description == "":
		return fmt.Errorf("description is required")
	case len(f.Description) > maxDescriptionLength:
		return fmt.Errorf("description must be at most %d characters", maxDescriptionLength)
	}
	if len(f.Contracts) > 0 && f.Implements != "" {
		return fmt.Errorf("an artifact is either an abstract (contracts) or a capability (implements), not both")
	}
	return nil
}
