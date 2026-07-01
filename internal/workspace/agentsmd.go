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

// capabilityRef is one capability bound to a contract: its name and the path to
// its entry document (empty when the capability is not among the selection).
type capabilityRef struct {
	Name string
	Path string
}

// contractLine renders one contract of a composition and the capabilities bound
// to it (empty when the contract is unbound).
type contractLine struct {
	Contract     string
	Capabilities []capabilityRef
}

// compositionView renders an abstract artifact as its contract plus the chosen
// implementations.
type compositionView struct {
	Abstract  string
	Path      string
	Complete  bool
	Contracts []contractLine
}

// mcpView renders a composed MCP integration: its instruction document and the
// agent targets enabled for it. MCPs read as setup/usage instructions rather than
// "contracts", so they get their own section and a flattened target list instead
// of the per-contract layout.
type mcpView struct {
	Name     string
	Path     string
	Complete bool
	Targets  []capabilityRef
}

// agentsData is the template payload for AGENTS.md.
type agentsData struct {
	SpecsDir        string
	Rules           []reference
	Skills          []reference
	Agents          []reference
	MCPs            []reference
	Compositions    []compositionView
	MCPCompositions []mcpView
}

var agentsTemplate = template.Must(
	template.New("agents.md").Parse(assets.AgentsTemplate),
)

// mcpComposition builds the view for a composed MCP: its instruction document
// and the flattened set of enabled target capabilities. It marks the abstract and
// every bound capability hidden so they do not also appear in the flat tables.
func mcpComposition(projectRoot string, a artifact.Artifact, bound map[string][]string, byIdentity map[artifact.Identity]artifact.Artifact, hidden map[artifact.Identity]bool) mcpView {
	view := mcpView{
		Name:     a.Name,
		Path:     displayPath(projectRoot, a.EntryPath),
		Complete: true,
	}
	for _, contract := range a.Contracts {
		caps := bound[contract]
		if len(caps) == 0 {
			view.Complete = false
		}
		// A capability implements an abstract of the same kind by name.
		for _, name := range caps {
			ref := capabilityRef{Name: name}
			capabilityID := artifact.Identity{Kind: a.Kind, Name: name}
			if capability, ok := byIdentity[capabilityID]; ok {
				ref.Path = displayPath(projectRoot, capability.EntryPath)
				hidden[capabilityID] = true
			}
			view.Targets = append(view.Targets, ref)
		}
	}
	return view
}

// RenderAgentsFile renders AGENTS.md for the given selected artifacts. Paths are
// made relative to projectRoot when possible (local artifacts) and kept absolute
// otherwise (shared artifacts referenced in place).
func RenderAgentsFile(projectRoot string, selected []artifact.Artifact, bindings map[artifact.Identity]map[string][]string) ([]byte, error) {
	byIdentity := make(map[artifact.Identity]artifact.Artifact, len(selected))
	for _, a := range selected {
		byIdentity[a.Identity()] = a
	}

	// Abstract skills and the capabilities bound to them render in the
	// compositions section and are hidden from the flat tables. Bindings are the
	// user's explicit choices — a contract with no binding stays unimplemented.
	hidden := make(map[artifact.Identity]bool)
	var compositions []compositionView
	var mcpCompositions []mcpView
	for _, a := range selected {
		if !a.IsAbstract() {
			continue
		}
		hidden[a.Identity()] = true
		bound := bindings[a.Identity()]

		// MCPs are setup/usage instructions, not loaded context — they render in
		// their own section as a flattened target list, not the per-contract
		// layout used for skill compositions.
		if a.Kind == artifact.KindMCP {
			mcpCompositions = append(mcpCompositions, mcpComposition(projectRoot, a, bound, byIdentity, hidden))
			continue
		}

		view := compositionView{
			Abstract: a.Name,
			Path:     displayPath(projectRoot, a.EntryPath),
			Complete: true,
		}
		for _, contract := range a.Contracts {
			line := contractLine{Contract: contract}
			// A capability implements an abstract of the same kind by name.
			for _, name := range bound[contract] {
				ref := capabilityRef{Name: name}
				capabilityID := artifact.Identity{Kind: a.Kind, Name: name}
				if capability, ok := byIdentity[capabilityID]; ok {
					ref.Path = displayPath(projectRoot, capability.EntryPath)
					hidden[capabilityID] = true
				}
				line.Capabilities = append(line.Capabilities, ref)
			}
			if len(line.Capabilities) == 0 {
				view.Complete = false
			}
			view.Contracts = append(view.Contracts, line)
		}
		compositions = append(compositions, view)
	}

	data := agentsData{
		SpecsDir:        config.AgentsDirName + "/" + config.SpecsDirName,
		Compositions:    compositions,
		MCPCompositions: mcpCompositions,
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
		case artifact.KindMCP:
			data.MCPs = append(data.MCPs, ref)
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
