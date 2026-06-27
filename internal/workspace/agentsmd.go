package workspace

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/assets"
	"github.com/devstationtech/harness/internal/compose"
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
func RenderAgentsFile(projectRoot string, selected []artifact.Artifact) ([]byte, error) {
	byIdentity := make(map[artifact.Identity]artifact.Artifact, len(selected))
	for _, a := range selected {
		byIdentity[a.Identity()] = a
	}

	// Compose: abstract skills and the capabilities bound to them render in the
	// compositions section and are hidden from the flat tables.
	hidden := make(map[artifact.Identity]bool)
	var compositions []compositionView
	for _, composition := range compose.Bind(selected) {
		abstract := byIdentity[composition.Abstract]
		hidden[composition.Abstract] = true
		bound := make(map[string]artifact.Identity)
		for _, binding := range composition.Bindings {
			bound[binding.Contract] = binding.Capability
			hidden[binding.Capability] = true
		}
		view := compositionView{
			Abstract: abstract.Name,
			Path:     displayPath(projectRoot, abstract.EntryPath),
			Complete: composition.Complete(),
		}
		for _, contract := range abstract.Contracts {
			line := contractLine{Contract: contract}
			if id, ok := bound[contract]; ok {
				capability := byIdentity[id]
				line.Capability = capability.Name
				line.Path = displayPath(projectRoot, capability.EntryPath)
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
// project, otherwise the absolute path (shared artifacts live in ~/.harness).
func displayPath(projectRoot, entryPath string) string {
	relative, err := filepath.Rel(projectRoot, entryPath)
	if err != nil || strings.HasPrefix(relative, "..") {
		return entryPath
	}
	return relative
}

// sanitizeCell makes a description safe to render inside a Markdown table cell by
// collapsing newlines and escaping pipe characters.
func sanitizeCell(text string) string {
	text = strings.ReplaceAll(text, "\r\n", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "|", "\\|")
	return strings.TrimSpace(text)
}
