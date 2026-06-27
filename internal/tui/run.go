package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/devstationtech/harness/internal/artifact"
)

// Result is the outcome of running the selection screen.
type Result struct {
	Confirmed bool
	Selected  []artifact.Artifact
}

// Run launches the full-screen selection UI over the merged catalog and blocks
// until the user saves or quits. warnings is the number of artifacts that were
// skipped while loading, surfaced as a footer indicator.
func Run(artifacts []artifact.Artifact, preselected map[artifact.Identity]bool, version string, warnings int) (Result, error) {
	program := tea.NewProgram(
		New(artifacts, preselected, version, warnings),
		tea.WithAltScreen(),
	)
	finalModel, err := program.Run()
	if err != nil {
		return Result{}, err
	}
	model := finalModel.(Model)
	return Result{Confirmed: model.Confirmed(), Selected: model.Selected()}, nil
}
