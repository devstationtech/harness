package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/devstationtech/harness/internal/artifact"
)

// step is the wizard stage the selection screen is in.
type step int

const (
	stepSelect  step = iota // the artifact list
	stepCompose             // composing one abstract skill
	stepConfirm             // final review / save
)

// composeView is the composition screen for one abstract skill: a capability is
// chosen for each of the abstract's contracts.
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

// startWizard advances from the selection list. With no abstract skills
// selected it saves immediately; otherwise it opens a composition screen for
// each selected abstract, in order, ending at a confirmation step.
func (m Model) startWizard() (tea.Model, tea.Cmd) {
	m.compositions = nil
	for _, it := range m.items {
		if it.selected && it.artifact.IsAbstract() {
			m.compositions = append(m.compositions, m.newComposition(it.artifact))
		}
	}
	if len(m.compositions) == 0 {
		m.confirmed = true
		return m, tea.Quit
	}
	m.step = stepCompose
	m.composeIndex = 0
	return m, nil
}

// newComposition builds the composition screen for one abstract skill,
// pre-selecting any capability already chosen.
func (m Model) newComposition(abstract artifact.Artifact) *composeView {
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
	return view
}

// handleComposeKey handles keys on a composition step: navigate contracts, cycle
// the chosen capability, advance to the next abstract (or the confirm step), or
// step back to the previous abstract (or the list).
func (m Model) handleComposeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	view := m.compositions[m.composeIndex]
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		view.cursor = clampIndex(view.cursor-1, len(view.contracts))
	case "down", "j":
		view.cursor = clampIndex(view.cursor+1, len(view.contracts))
	case "left", "h":
		cycle(view, -1)
	case "right", "l", " ":
		cycle(view, 1)
	case "enter":
		m.applyView(view)
		if m.composeIndex < len(m.compositions)-1 {
			m.composeIndex++
		} else {
			m.step = stepConfirm
		}
	case "esc", "backspace":
		m.applyView(view)
		if m.composeIndex > 0 {
			m.composeIndex--
		} else {
			m.step = stepSelect
		}
	}
	return m, nil
}

// cycle moves the focused contract's choice through its candidates and the
// "no implementation" option, wrapping around.
func cycle(view *composeView, delta int) {
	if len(view.contracts) == 0 {
		return
	}
	choice := &view.contracts[view.cursor]
	options := len(choice.candidates) + 1 // +1 for "no implementation"
	index := choice.chosen + 1
	index = ((index+delta)%options + options) % options
	choice.chosen = index - 1
}

// applyView folds one composition's choices into the selection: every chosen
// capability is checked, every other capability of that abstract unchecked.
func (m *Model) applyView(view *composeView) {
	chosen := make(map[artifact.Identity]bool)
	for _, choice := range view.contracts {
		if choice.chosen >= 0 {
			chosen[choice.candidates[choice.chosen].Identity()] = true
		}
	}
	for i := range m.capabilities {
		if m.capabilities[i].artifact.Implements == view.abstract.Name {
			m.capabilities[i].selected = chosen[m.capabilities[i].artifact.Identity()]
		}
	}
}

// handleConfirmKey handles the final review step: save, go back to the start, or
// quit without saving.
func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.confirmed = true
		return m, tea.Quit
	case "b", "left", "esc":
		m.step = stepSelect
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// --- rendering ---

// renderCompose draws a composition step with the main screen's base structure:
// wordmark header, titled box, footer.
func (m Model) renderCompose(inner int) string {
	view := m.compositions[m.composeIndex]
	title := "compose · " + view.abstract.Name
	if len(m.compositions) > 1 {
		title += fmt.Sprintf("  (%d/%d)", m.composeIndex+1, len(m.compositions))
	}
	content := m.composeRows(view, inner-boxChromeCols)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(inner),
		m.renderTitledBox(title, inner, content),
		m.footerLine(inner, "↑/↓ contract · ←/→ choose · enter next · esc back · ctrl+c quit"),
	)
}

// composeRows builds the box content for one composition, padded to the content
// height.
func (m Model) composeRows(view *composeView, width int) []string {
	rows := []string{
		m.paint(width, m.styles.subtitle.Render("Choose an implementation for each contract")),
		m.paint(width, ""),
	}
	for index, choice := range view.contracts {
		rows = append(rows, m.renderContractRow(view, index, choice, width))
	}
	rows = append(rows, m.paint(width, ""), m.paint(width, composeStatus(view, m.styles)))
	return m.fitRows(rows, width)
}

// renderContractRow draws one contract and its chosen capability, padded to width.
func (m Model) renderContractRow(view *composeView, index int, choice contractChoice, width int) string {
	const contractWidth = 18

	cursor := m.styles.base.Render("  ")
	contractStyle := m.styles.name
	if index == view.cursor {
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

// composeStatus reports whether every contract of a view has an implementation.
func composeStatus(view *composeView, s styles) string {
	unbound := unboundCount(view)
	if unbound > 0 {
		return s.warn.Render(fmt.Sprintf("⚠ %d contract(s) without an implementation", unbound))
	}
	return s.count.Render("✓ composition complete")
}

func unboundCount(view *composeView) int {
	unbound := 0
	for _, choice := range view.contracts {
		if choice.chosen < 0 {
			unbound++
		}
	}
	return unbound
}

// renderConfirm draws the final review step: counts, per-composition status, and
// the save/back/quit choices.
func (m Model) renderConfirm(inner int) string {
	content := m.confirmRows(inner - boxChromeCols)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(inner),
		m.renderTitledBox("review", inner, content),
		m.footerLine(inner, "enter save · b back to start · q quit"),
	)
}

// confirmRows summarises the selection and the composition completeness.
func (m Model) confirmRows(width int) []string {
	rows := []string{
		m.paint(width, m.styles.subtitle.Render("Review — saving writes harness.yaml and AGENTS.md")),
		m.paint(width, ""),
	}

	counts := map[artifact.Kind]int{}
	for _, a := range m.Selected() {
		counts[a.Kind]++
	}
	for _, kind := range artifact.Kinds() {
		rows = append(rows, m.paint(width, m.styles.name.Render(fmt.Sprintf("  %-7s %d selected", kind.Title(), counts[kind]))))
	}
	rows = append(rows, m.paint(width, ""))

	for _, view := range m.compositions {
		if unbound := unboundCount(view); unbound > 0 {
			rows = append(rows, m.paint(width, m.styles.warn.Render(
				fmt.Sprintf("  ⚠ %s — %d contract(s) without an implementation", view.abstract.Name, unbound),
			)))
		} else {
			rows = append(rows, m.paint(width, m.styles.count.Render(
				fmt.Sprintf("  ✓ %s — fully composed", view.abstract.Name),
			)))
		}
	}
	return m.fitRows(rows, width)
}

// footerLine renders a single help line across the inner width.
func (m Model) footerLine(inner int, help string) string {
	return m.styles.base.Width(inner).MaxWidth(inner).Render(m.styles.footer.Render(help))
}

// fitRows pads or clamps rows to the content height.
func (m Model) fitRows(rows []string, width int) []string {
	ch := m.contentHeight()
	for len(rows) < ch {
		rows = append(rows, m.paint(width, ""))
	}
	if len(rows) > ch {
		rows = rows[:ch]
	}
	return rows
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
