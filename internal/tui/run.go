package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/devstationtech/harness/internal/artifact"
)

// Result is the outcome of running the selection screen.
type Result struct {
	Confirmed bool
	Selected  []artifact.Artifact
	// Bindings is, per composed abstract skill, the capability chosen for each
	// contract (a contract left unimplemented is omitted).
	Bindings map[artifact.Identity]map[string]string
	// Localized are artifacts the user asked to copy into the project's .agents,
	// to be vendored (committed, overriding the shared one) on save.
	Localized []artifact.Identity
}

// Run launches the full-screen selection UI over the merged catalog and blocks
// until the user saves or quits. warnings is the number of artifacts that were
// skipped while loading, surfaced as a footer indicator. priorBindings are the
// manifest's recorded bindings, used to pre-fill the composition wizard.
func Run(
	artifacts []artifact.Artifact,
	preselected map[artifact.Identity]bool,
	priorBindings map[artifact.Identity]map[string]string,
	version string,
	warnings int,
) (Result, error) {
	model := New(artifacts, preselected, version, warnings)
	model.priorBindings = priorBindings

	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return Result{}, err
	}
	m := finalModel.(Model)
	return Result{
		Confirmed: m.Confirmed(),
		Selected:  m.Selected(),
		Bindings:  m.Bindings(),
		Localized: m.Localized(),
	}, nil
}
