package tui

import (
	"fmt"
	"strings"

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

// composeView is the composition screen for one abstract artifact: capabilities
// are chosen for each of its contracts. When multiple is false (the default) each
// contract binds at most one capability (a radio choice with a "no
// implementation" option); when true each capability toggles independently, so a
// contract may bind several at once (e.g. an MCP enabled for several agents).
type composeView struct {
	abstract  artifact.Artifact
	multiple  bool
	contracts []contractChoice
	cursor    int // single mode: a contract index; multi mode: a candidate-row index
}

// contractChoice is one contract and the capabilities that can fulfil it. picked
// runs parallel to candidates; in single mode at most one entry is true.
type contractChoice struct {
	contract   string
	candidates []artifact.Artifact
	picked     []bool
}

// single returns the index of the chosen candidate in single-select mode, or -1
// when the contract is left without an implementation.
func (c contractChoice) single() int {
	for i, on := range c.picked {
		if on {
			return i
		}
	}
	return -1
}

// setSingle selects exactly one candidate (idx), or none when idx is negative.
func (c *contractChoice) setSingle(idx int) {
	for i := range c.picked {
		c.picked[i] = i == idx
	}
}

// bound reports whether at least one capability is chosen for the contract.
func (c contractChoice) bound() bool { return c.single() >= 0 }

// chosenNames returns the names of every chosen capability, in candidate order.
func (c contractChoice) chosenNames() []string {
	var names []string
	for i, on := range c.picked {
		if on {
			names = append(names, c.candidates[i].Name)
		}
	}
	return names
}

// candidateRows flattens every (contract, candidate) pair, the navigation unit of
// multi-select mode.
func (v composeView) candidateRows() []struct{ contract, candidate int } {
	var rows []struct{ contract, candidate int }
	for ci := range v.contracts {
		for cj := range v.contracts[ci].candidates {
			rows = append(rows, struct{ contract, candidate int }{ci, cj})
		}
	}
	return rows
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

// newComposition builds the composition screen for one abstract skill. When the
// manifest recorded bindings for it (a reopened project), those are
// authoritative per contract — honoring an explicit "no implementation";
// otherwise it falls back to whichever capability is already selected.
func (m Model) newComposition(abstract artifact.Artifact) *composeView {
	wanted := m.priorBindings[abstract.Identity()]
	view := &composeView{abstract: abstract, multiple: abstract.Multiple}
	for _, contract := range abstract.Contracts {
		choice := contractChoice{contract: contract}
		for _, it := range m.capabilities {
			capability := it.artifact
			// A capability implements an abstract of the same kind by name.
			if capability.Kind == abstract.Kind && capability.Implements == abstract.Name && contains(capability.Provides, contract) {
				choice.candidates = append(choice.candidates, capability)
				choice.picked = append(choice.picked, picksCandidate(wanted, contract, capability.Name, it.selected))
			}
		}
		view.contracts = append(view.contracts, choice)
	}
	return view
}

// picksCandidate decides whether a candidate is chosen for a contract. With
// recorded bindings it matches by name (an absent contract stays unbound);
// without them it falls back to whether the capability is already selected.
func picksCandidate(wanted map[string][]string, contract, capability string, selected bool) bool {
	if wanted != nil {
		return contains(wanted[contract], capability)
	}
	return selected
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
		view.cursor = clampIndex(view.cursor-1, composeRowCount(view))
	case "down", "j":
		view.cursor = clampIndex(view.cursor+1, composeRowCount(view))
	case "left", "h":
		if !view.multiple {
			cycle(view, -1)
		}
	case "right", "l":
		if !view.multiple {
			cycle(view, 1)
		}
	case " ":
		toggle(view)
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

// composeRowCount is the number of navigable rows: one per contract in single
// mode, one per candidate in multi mode.
func composeRowCount(view *composeView) int {
	if view.multiple {
		return len(view.candidateRows())
	}
	return len(view.contracts)
}

// cycle moves the focused contract's single choice through its candidates and the
// "no implementation" option, wrapping around. Single-select only.
func cycle(view *composeView, delta int) {
	if len(view.contracts) == 0 {
		return
	}
	choice := &view.contracts[view.cursor]
	options := len(choice.candidates) + 1 // +1 for "no implementation"
	index := choice.single() + 1
	index = ((index+delta)%options + options) % options
	choice.setSingle(index - 1)
}

// toggle flips the focused candidate. In single mode space advances the radio
// (like →); in multi mode it toggles just the focused candidate independently.
func toggle(view *composeView) {
	if !view.multiple {
		cycle(view, 1)
		return
	}
	rows := view.candidateRows()
	if len(rows) == 0 {
		return
	}
	r := rows[clampIndex(view.cursor, len(rows))]
	choice := &view.contracts[r.contract]
	choice.picked[r.candidate] = !choice.picked[r.candidate]
}

// applyView folds one composition's choices into the selection: every chosen
// capability is checked, every other capability of that abstract unchecked.
func (m *Model) applyView(view *composeView) {
	chosen := make(map[artifact.Identity]bool)
	for _, choice := range view.contracts {
		for i, on := range choice.picked {
			if on {
				chosen[choice.candidates[i].Identity()] = true
			}
		}
	}
	for i := range m.capabilities {
		c := m.capabilities[i].artifact
		if c.Kind == view.abstract.Kind && c.Implements == view.abstract.Name {
			m.capabilities[i].selected = chosen[c.Identity()]
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
	help := "↑/↓ contract · ←/→ choose · enter next · esc back · ctrl+c quit"
	if view.multiple {
		help = "↑/↓ row · space toggle · enter next · esc back · ctrl+c quit"
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(inner),
		m.renderTitledBox(m.styles.chip.Render(title), inner, content),
		m.footerLine(inner, help),
	)
}

// composeRows builds the box content for one composition, padded to the content
// height. Single-select renders one row per contract; multi-select renders each
// contract's candidates as independent checkboxes.
func (m Model) composeRows(view *composeView, width int) []string {
	header := "Choose an implementation for each contract"
	if view.multiple {
		header = "Toggle every implementation you want active"
	}
	rows := []string{
		m.paint(width, m.styles.subtitle.Render(header)),
		m.paint(width, ""),
	}
	if view.multiple {
		rows = append(rows, m.multiContractRows(view, width)...)
	} else {
		for index, choice := range view.contracts {
			rows = append(rows, m.renderContractRow(view, index, choice, width))
		}
	}
	rows = append(rows, m.paint(width, ""), m.paint(width, composeStatus(view, m.styles)))
	return m.fitRows(rows, width)
}

// multiContractRows renders each contract as a header followed by a checkbox per
// candidate; the cursor walks the flattened candidate rows.
func (m Model) multiContractRows(view *composeView, width int) []string {
	var rows []string
	flat := 0
	for ci := range view.contracts {
		choice := view.contracts[ci]
		rows = append(rows, m.paint(width, m.styles.section.Render(strings.ToUpper(choice.contract))))
		for cj, candidate := range choice.candidates {
			cursor := m.styles.base.Render("  ")
			if flat == view.cursor {
				cursor = m.styles.cursor.Render("› ")
			}
			box := m.styles.checkOff.Render("[ ]")
			if choice.picked[cj] {
				box = m.styles.checkOn.Render("[x]")
			}
			label := candidate.Name
			if candidate.Stack != "" {
				label += " · " + candidate.Stack
			}
			line := cursor + box + m.styles.base.Render(" ") + m.styles.name.Render(label)
			rows = append(rows, m.styles.base.Width(width).MaxWidth(width).Render(line))
			flat++
		}
	}
	return rows
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
	if sel := choice.single(); sel >= 0 {
		capability := choice.candidates[sel]
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
		if !choice.bound() {
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
		m.renderTitledBox(m.styles.chip.Render("review"), inner, content),
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
