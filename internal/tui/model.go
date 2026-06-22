package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/devstationtech/harness/internal/artifact"
)

// item is one selectable artifact row in the selection screen.
type item struct {
	artifact artifact.Artifact
	selected bool
}

// Model is the artifact selection screen: a grouped, checkbox list over the
// merged catalog. It performs no I/O; the caller reads Selected after the program
// exits and persists the result.
type Model struct {
	styles    styles
	items     []item
	cursor    int
	width     int
	height    int
	confirmed bool
}

// New builds a selection model from the merged catalog. Artifacts whose identity
// is in preselected start checked (so re-running in a configured project shows
// the current selection).
func New(artifacts []artifact.Artifact, preselected map[artifact.Identity]bool) Model {
	items := make([]item, 0, len(artifacts))
	for _, a := range artifacts {
		items = append(items, item{artifact: a, selected: preselected[a.Identity()]})
	}
	return Model{styles: newStyles(), items: items, width: 80}
}

// Confirmed reports whether the user chose to save (Enter) rather than quit.
func (m Model) Confirmed() bool { return m.confirmed }

// Selected returns the artifacts the user checked.
func (m Model) Selected() []artifact.Artifact {
	var out []artifact.Artifact
	for _, it := range m.items {
		if it.selected {
			out = append(out, it.artifact)
		}
	}
	return out
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
		}
	case " ", "x":
		if len(m.items) > 0 {
			m.items[m.cursor].selected = !m.items[m.cursor].selected
		}
	case "a":
		m.toggleSection()
	case "enter":
		m.confirmed = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) moveCursor(delta int) {
	if len(m.items) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
}

// toggleSection flips every item that shares the kind under the cursor. If any in
// the section is unchecked it checks all; otherwise it clears them.
func (m *Model) toggleSection() {
	if len(m.items) == 0 {
		return
	}
	kind := m.items[m.cursor].artifact.Kind
	anyOff := false
	for _, it := range m.items {
		if it.artifact.Kind == kind && !it.selected {
			anyOff = true
			break
		}
	}
	for index := range m.items {
		if m.items[index].artifact.Kind == kind {
			m.items[index].selected = anyOff
		}
	}
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(m.styles.title.Render("harness"))
	b.WriteString("  ")
	b.WriteString(m.styles.subtitle.Render("select the artifacts for this project"))
	b.WriteString("\n")

	if len(m.items) == 0 {
		b.WriteString(m.styles.empty.Render("No artifacts found. Run `harness init` to seed your shared library (~/.harness),\nor add artifacts under .agents/."))
		b.WriteString("\n")
		b.WriteString(m.styles.footer.Render("q quit"))
		return m.styles.app.Render(b.String())
	}

	var currentKind artifact.Kind
	first := true
	for index, it := range m.items {
		if first || it.artifact.Kind != currentKind {
			currentKind = it.artifact.Kind
			first = false
			b.WriteString(m.styles.section.Render(m.sectionTitle(currentKind)))
			b.WriteString("\n")
		}
		b.WriteString(m.renderRow(index, it))
		b.WriteString("\n")
	}

	b.WriteString(m.footer())
	return m.styles.app.Render(b.String())
}

func (m Model) sectionTitle(kind artifact.Kind) string {
	hint := map[artifact.Kind]string{
		artifact.KindRule:  "load ALWAYS",
		artifact.KindSkill: "load on NEED",
		artifact.KindAgent: "delegate on NEED",
	}[kind]
	count, total := m.sectionCounts(kind)
	return fmt.Sprintf("%s · %s  (%d/%d)", kind.Title(), hint, count, total)
}

func (m Model) sectionCounts(kind artifact.Kind) (selected, total int) {
	for _, it := range m.items {
		if it.artifact.Kind != kind {
			continue
		}
		total++
		if it.selected {
			selected++
		}
	}
	return selected, total
}

func (m Model) renderRow(index int, it item) string {
	cursor := "  "
	if index == m.cursor {
		cursor = m.styles.cursor.Render("›") + " "
	}

	checkbox := m.styles.checkOff.Render("[ ]")
	if it.selected {
		checkbox = m.styles.checkOn.Render("[x]")
	}

	nameStyle := m.styles.name
	if index == m.cursor {
		nameStyle = m.styles.nameActive
	}
	name := nameStyle.Render(it.artifact.Name)

	badge := m.styles.badgeShared.Render("shared")
	if it.artifact.Source == artifact.SourceLocal {
		badge = m.styles.badgeLocal.Render("local")
		if it.artifact.OverridesShared {
			badge = m.styles.override.Render("local (override)")
		}
	}

	left := fmt.Sprintf("%s%s %s  %s", cursor, checkbox, name, badge)
	description := m.styles.description.Render(truncate(it.artifact.Description, m.descriptionWidth(left)))
	return left + "  " + description
}

// descriptionWidth returns how much horizontal room is left for the description.
func (m Model) descriptionWidth(prefix string) int {
	available := m.width - 4 - lipgloss.Width(prefix) - 2
	if available < 16 {
		return 16
	}
	return available
}

func (m Model) footer() string {
	selected := 0
	for _, it := range m.items {
		if it.selected {
			selected++
		}
	}
	help := "↑/↓ move · space toggle · a section · enter save · q quit"
	tally := m.styles.count.Render(fmt.Sprintf("%d selected", selected))
	return m.styles.footer.Render(help+"    ") + tally
}

// truncate shortens text to width runes, appending an ellipsis when cut.
func truncate(text string, width int) string {
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	if width <= 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}
