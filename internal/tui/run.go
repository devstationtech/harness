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

// Run launches the selection screen over the merged catalog and blocks until the
// user saves or quits.
func Run(artifacts []artifact.Artifact, preselected map[artifact.Identity]bool) (Result, error) {
	program := tea.NewProgram(New(artifacts, preselected))
	finalModel, err := program.Run()
	if err != nil {
		return Result{}, err
	}
	model := finalModel.(Model)
	return Result{Confirmed: model.Confirmed(), Selected: model.Selected()}, nil
}
