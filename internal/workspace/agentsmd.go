package workspace

import (
	"bytes"
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

// agentsData is the template payload for AGENTS.md.
type agentsData struct {
	SpecsDir string
	Rules    []reference
	Skills   []reference
	Agents   []reference
}

var agentsTemplate = template.Must(
	template.New("agents.md").Parse(assets.AgentsTemplate),
)

// RenderAgentsFile renders AGENTS.md for the given selected artifacts. Paths are
// made relative to projectRoot when possible (local artifacts) and kept absolute
// otherwise (shared artifacts referenced in place).
func RenderAgentsFile(projectRoot string, selected []artifact.Artifact) ([]byte, error) {
	data := agentsData{
		SpecsDir: config.AgentsDirName + "/" + config.SpecsDirName,
	}
	for _, a := range selected {
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
