package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/assets"
	"github.com/devstationtech/harness/internal/config"
)

// reference is a single artifact row rendered into AGENTS.md.
type reference struct {
	Name        string
	Description string
	Source      string
	Path        string
}

// contractLine renders one contract of a composition and the capability bound to
// it (Capability/Path empty when the contract is unbound).
type contractLine struct {
	Contract   string
	Capability string
	Path       string
}

// compositionView renders an abstract skill as its contract plus the chosen
// implementations.
type compositionView struct {
	Abstract  string
	Path      string
	Complete  bool
	Contracts []contractLine
}

// agentsData is the template payload for AGENTS.md.
type agentsData struct {
	SpecsDir     string
	Rules        []reference
	Skills       []reference
	Agents       []reference
	Compositions []compositionView
}

var agentsTemplate = template.Must(
	template.New("agents.md").Parse(assets.AgentsTemplate),
)

// RenderAgentsFile renders AGENTS.md for the given selected artifacts. Paths are
// made relative to projectRoot when possible (local artifacts) and kept absolute
// otherwise (shared artifacts referenced in place).
func RenderAgentsFile(projectRoot string, selected []artifact.Artifact, bindings map[artifact.Identity]map[string]string) ([]byte, error) {
	byIdentity := make(map[artifact.Identity]artifact.Artifact, len(selected))
	for _, a := range selected {
		byIdentity[a.Identity()] = a
	}

	// Abstract skills and the capabilities bound to them render in the
	// compositions section and are hidden from the flat tables. Bindings are the
	// user's explicit choices — a contract with no binding stays unimplemented.
	hidden := make(map[artifact.Identity]bool)
	var compositions []compositionView
	for _, a := range selected {
		if !a.IsAbstract() {
			continue
		}
		hidden[a.Identity()] = true
		bound := bindings[a.Identity()]
		view := compositionView{
			Abstract: a.Name,
			Path:     displayPath(projectRoot, a.EntryPath),
			Complete: true,
		}
		for _, contract := range a.Contracts {
			line := contractLine{Contract: contract}
			if name := bound[contract]; name != "" {
				capabilityID := artifact.Identity{Kind: artifact.KindSkill, Name: name}
				line.Capability = name
				if capability, ok := byIdentity[capabilityID]; ok {
					line.Path = displayPath(projectRoot, capability.EntryPath)
					hidden[capabilityID] = true
				}
			} else {
				view.Complete = false
			}
			view.Contracts = append(view.Contracts, line)
		}
		compositions = append(compositions, view)
	}

	data := agentsData{
		SpecsDir:     config.AgentsDirName + "/" + config.SpecsDirName,
		Compositions: compositions,
	}
	for _, a := range selected {
		if hidden[a.Identity()] {
			continue
		}
		ref := reference{
			Name:        a.Name,
			Description: sanitizeCell(a.Description),
			Source:      string(a.Source),
			Path:        displayPath(projectRoot, a.EntryPath),
		}
		switch a.Kind {
		case artifact.KindRule:
			data.Rules = append(data.Rules, ref)
		case artifact.KindSkill:
			data.Skills = append(data.Skills, ref)
		case artifact.KindAgent:
			data.Agents = append(data.Agents, ref)
		}
	}

	var buffer bytes.Buffer
	if err := agentsTemplate.Execute(&buffer, data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// displayPath returns a project-relative path when entryPath is inside the
// project. A shared artifact lives outside the project (in ~/.harness): it is
// shown under "~/" rather than an absolute path, so the committed AGENTS.md does
// not bake in the maintainer's home directory (and username).
func displayPath(projectRoot, entryPath string) string {
	if relative, err := filepath.Rel(projectRoot, entryPath); err == nil && !strings.HasPrefix(relative, "..") {
		return relative
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, entryPath); err == nil && !strings.HasPrefix(rel, "..") {
			return "~/" + filepath.ToSlash(rel)
		}
	}
	return entryPath
}

// sanitizeCell makes a description safe to render inside a Markdown table cell by
// collapsing newlines and escaping pipe characters.
func sanitizeCell(text string) string {
	text = strings.ReplaceAll(text, "\r\n", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "|", "\\|")
	return strings.TrimSpace(text)
}
