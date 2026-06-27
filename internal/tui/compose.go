package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/devstationtech/harness/internal/artifact"
)

// composeView is the composition screen for one abstract skill: it lets the user
// choose a capability for each of the abstract's contracts. Choices are applied
// back to the selection (the chosen capabilities become checked) on return, so
// the existing save flow derives the bindings.
type composeView struct {
	abstract  artifact.Artifact
	contracts []contractChoice
	cursor    int
}

// contractChoice is one contract and the capabilities that can fulfil it.
type contractChoice struct {
	contract   string
	candidates []artifact.Artifact
	chosen     int // index into candidates, or -1 for "no implementation"
}

// abstractUnderCursor returns the abstract artifact under the cursor, if any.
func (m Model) abstractUnderCursor() (artifact.Artifact, bool) {
	if len(m.items) == 0 {
		return artifact.Artifact{}, false
	}
	a := m.items[m.cursor].artifact
	if a.IsAbstract() {
		return a, true
	}
	return artifact.Artifact{}, false
}

// openCompose builds the composition screen for the abstract under the cursor
// and selects the abstract (composing it makes it part of the selection).
func (m *Model) openCompose() {
	abstract, ok := m.abstractUnderCursor()
	if !ok {
		return
	}
	m.items[m.cursor].selected = true

	view := &composeView{abstract: abstract}
	for _, contract := range abstract.Contracts {
		choice := contractChoice{contract: contract, chosen: -1}
		for _, it := range m.capabilities {
			capability := it.artifact
			if capability.Implements == abstract.Name && contains(capability.Provides, contract) {
				choice.candidates = append(choice.candidates, capability)
				if it.selected {
					choice.chosen = len(choice.candidates) - 1
				}
			}
		}
		view.contracts = append(view.contracts, choice)
	}
	m.compose = view
}

// handleComposeKey handles keys on the composition screen.
func (m Model) handleComposeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.compose.cursor = clampIndex(m.compose.cursor-1, len(m.compose.contracts))
	case "down", "j":
		m.compose.cursor = clampIndex(m.compose.cursor+1, len(m.compose.contracts))
	case "left", "h":
		m.cycleChoice(-1)
	case "right", "l", " ":
		m.cycleChoice(1)
	case "enter", "esc", "q", "backspace":
		m.applyCompose()
		m.compose = nil
	}
	return m, nil
}

// cycleChoice moves the focused contract's choice through its candidates and the
// "no implementation" option, wrapping around.
func (m *Model) cycleChoice(delta int) {
	if m.compose == nil || len(m.compose.contracts) == 0 {
		return
	}
	choice := &m.compose.contracts[m.compose.cursor]
	options := len(choice.candidates) + 1 // +1 for "no implementation"
	index := choice.chosen + 1
	index = ((index+delta)%options + options) % options
	choice.chosen = index - 1
}

// applyCompose folds the screen's choices back into the selection: every chosen
// capability is checked and every other capability of this abstract is unchecked.
func (m *Model) applyCompose() {
	if m.compose == nil {
		return
	}
	chosen := make(map[artifact.Identity]bool)
	for _, choice := range m.compose.contracts {
		if choice.chosen >= 0 {
			chosen[choice.candidates[choice.chosen].Identity()] = true
		}
	}
	abstractName := m.compose.abstract.Name
	for i := range m.capabilities {
		if m.capabilities[i].artifact.Implements == abstractName {
			m.capabilities[i].selected = chosen[m.capabilities[i].artifact.Identity()]
		}
	}
}

// renderCompose draws the composition screen with the same base structure as the
// main selection screen — the wordmark header, a titled box, and a footer — so
// it feels like a sub-screen of the same tool rather than a separate page.
func (m Model) renderCompose(inner int) string {
	title := "compose · " + m.compose.abstract.Name
	content := m.composeRows(inner - boxChromeCols)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(inner),
		m.renderTitledBox(title, inner, content),
		m.renderComposeFooter(inner),
	)
}

// composeRows builds the box content: an intro line, one row per contract, and a
// completeness status, padded/clamped to the available content height.
func (m Model) composeRows(width int) []string {
	rows := []string{
		m.paint(width, m.styles.subtitle.Render("Choose an implementation for each contract")),
		m.paint(width, ""),
	}
	for index, choice := range m.compose.contracts {
		rows = append(rows, m.renderContractRow(index, choice, width))
	}
	rows = append(rows, m.paint(width, ""), m.paint(width, m.composeStatus()))

	ch := m.contentHeight()
	for len(rows) < ch {
		rows = append(rows, m.paint(width, ""))
	}
	if len(rows) > ch {
		rows = rows[:ch]
	}
	return rows
}

// composeStatus reports whether every contract has an implementation.
func (m Model) composeStatus() string {
	unbound := 0
	for _, choice := range m.compose.contracts {
		if choice.chosen < 0 {
			unbound++
		}
	}
	if unbound > 0 {
		return m.styles.warn.Render(fmt.Sprintf("⚠ %d contract(s) without an implementation", unbound))
	}
	return m.styles.count.Render("✓ composition complete")
}

// renderContractRow draws one contract and its chosen capability, padded to width.
func (m Model) renderContractRow(index int, choice contractChoice, width int) string {
	const contractWidth = 18

	cursor := m.styles.base.Render("  ")
	contractStyle := m.styles.name
	if index == m.compose.cursor {
		cursor = m.styles.cursor.Render("› ")
		contractStyle = m.styles.nameActive
	}
	contractCell := contractStyle.Width(contractWidth).MaxWidth(contractWidth).Render(truncate(choice.contract, contractWidth))

	chosen := m.styles.checkOff.Render("[ ] no implementation")
	if choice.chosen >= 0 {
		capability := choice.candidates[choice.chosen]
		label := capability.Name
		if capability.Stack != "" {
			label += " · " + capability.Stack
		}
		chosen = m.styles.checkOn.Render("[x] " + label)
	}

	hint := ""
	if count := len(choice.candidates); count > 1 {
		hint = m.styles.sectionHint.Render(fmt.Sprintf("   (%d options)", count))
	}

	line := cursor + contractCell + m.styles.base.Render("  ") + chosen + hint
	return m.styles.base.Width(width).MaxWidth(width).Render(line)
}

// renderComposeFooter mirrors the main footer with composition-specific help.
func (m Model) renderComposeFooter(inner int) string {
	help := "↑/↓ contract · ←/→ choose · enter done · esc back · ctrl+c quit"
	return m.styles.base.Width(inner).MaxWidth(inner).Render(m.styles.footer.Render(help))
}

// clampIndex bounds i to [0, length-1] (returns 0 when length is 0).
func clampIndex(i, length int) int {
	switch {
	case length == 0:
		return 0
	case i < 0:
		return 0
	case i >= length:
		return length - 1
	default:
		return i
	}
}

// contains reports whether values includes target.
func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
