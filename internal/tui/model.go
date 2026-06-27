package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/devstationtech/harness/internal/artifact"
)

// logoArt is the HARNESS wordmark shown top-right. All lines are equal width.
const logoArt = `██╗  ██╗ █████╗ ██████╗ ███╗   ██╗███████╗███████╗███████╗
██║  ██║██╔══██╗██╔══██╗████╗  ██║██╔════╝██╔════╝██╔════╝
███████║███████║██████╔╝██╔██╗ ██║█████╗  ███████╗███████╗
██╔══██║██╔══██║██╔══██╗██║╚██╗██║██╔══╝  ╚════██║╚════██║
██║  ██║██║  ██║██║  ██║██║ ╚████║███████╗███████║███████║
╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝╚══════╝`

const (
	logoLines   = 6
	logoCols    = 58
	headerRows  = logoLines // wide layout: the wordmark sets the height
	compactRows = 2         // title + version chip (narrow layout)
	footerRows  = 1

	// The titled box around the list: top+bottom border + top/bottom padding
	// (rows), left+right border + left/right padding (cols).
	boxChromeRows = 4
	boxChromeCols = 4

	// Table column widths. The name column is sized to the catalog (see New);
	// these bound it so one long name can't crowd out the description.
	minNameWidth   = 10
	maxNameWidth   = 30
	sourceColWidth = 6 // fits "shared" / "local" / "local*"
)

// item is one selectable artifact row in the selection screen.
type item struct {
	artifact artifact.Artifact
	selected bool
	// localized marks a shared/remote artifact the user asked to copy into the
	// project's .agents (committed, overriding the shared one). The copy itself
	// happens on save, keeping the model free of I/O.
	localized bool
}

// detailView holds the info page shown for a single artifact (opened with `i`).
// The excerpt is read on demand from the artifact's source document.
type detailView struct {
	artifact artifact.Artifact
	excerpt  string
	err      error
}

// Model is the artifact selection screen: a full-screen, grouped, checkbox list
// over the merged catalog. It performs no I/O; the caller reads Selected after
// the program exits and persists the result.
type Model struct {
	styles  styles
	version string
	items   []item
	// capabilities are hidden from the selection list — they implement an
	// abstract skill and are chosen on its composition screen, not on their own.
	capabilities []item
	nameWidth    int           // shared width of the name column (table alignment)
	activeKind   artifact.Kind // the selected tab/category
	cursor       int           // index into items, always within the active tab
	offset       int           // first visible visual-line (scroll position)
	width        int
	height       int
	warnings     int // count of artifacts skipped while loading
	confirmed    bool
	detail       *detailView // non-nil while the info page is open

	// Wizard state: after the selection list, the user steps through a
	// composition screen per selected abstract skill, then a confirmation step.
	step         step
	compositions []*composeView
	composeIndex int
	// priorBindings are the manifest's recorded bindings (abstract identity →
	// contract → capability name), used to pre-fill the wizard on reopen so an
	// explicit "no implementation" survives a round trip.
	priorBindings map[artifact.Identity]map[string]string
}

// New builds a selection model from the merged catalog. Artifacts whose identity
// is in preselected start checked. warnings is the number of skipped artifacts.
func New(artifacts []artifact.Artifact, preselected map[artifact.Identity]bool, version string, warnings int) Model {
	var items, capabilities []item
	nameWidth := minNameWidth
	for _, a := range artifacts {
		entry := item{artifact: a, selected: preselected[a.Identity()]}
		if a.IsCapability() {
			// Hidden from the list; surfaced on the abstract's composition screen.
			capabilities = append(capabilities, entry)
			continue
		}
		items = append(items, entry)
		if w := lipgloss.Width(a.Name); w > nameWidth {
			nameWidth = w
		}
	}
	if nameWidth > maxNameWidth {
		nameWidth = maxNameWidth
	}
	m := Model{
		styles:       newStyles(),
		version:      version,
		items:        items,
		capabilities: capabilities,
		nameWidth:    nameWidth,
		warnings:     warnings,
	}
	// Start on the first non-empty tab, with the cursor on its first row.
	if kinds := m.tabKinds(); len(kinds) > 0 {
		m.activeKind = kinds[0]
		if idx := m.activeIndices(); len(idx) > 0 {
			m.cursor = idx[0]
		}
	}
	return m
}

// tabKinds returns the kinds that have at least one (visible) item, in canonical
// order — one tab each. New kinds (e.g. MCP) appear automatically once present.
func (m Model) tabKinds() []artifact.Kind {
	var kinds []artifact.Kind
	for _, kind := range artifact.Kinds() {
		for _, it := range m.items {
			if it.artifact.Kind == kind {
				kinds = append(kinds, kind)
				break
			}
		}
	}
	return kinds
}

// activeIndices returns the indices into items of the active tab's artifacts.
func (m Model) activeIndices() []int {
	var idx []int
	for i, it := range m.items {
		if it.artifact.Kind == m.activeKind {
			idx = append(idx, i)
		}
	}
	return idx
}

// Confirmed reports whether the user chose to save (Enter) rather than quit.
func (m Model) Confirmed() bool { return m.confirmed }

// Selected returns the artifacts the user checked, including capabilities chosen
// on a composition screen. A capability is included only when the abstract skill
// it implements is also selected — never as an orphan (e.g. after its abstract
// is deselected), so it cannot linger in the manifest or AGENTS.md.
func (m Model) Selected() []artifact.Artifact {
	selectedAbstracts := make(map[string]bool)
	var out []artifact.Artifact
	for _, it := range m.items {
		if it.selected {
			out = append(out, it.artifact)
			if it.artifact.IsAbstract() {
				selectedAbstracts[it.artifact.Name] = true
			}
		}
	}
	for _, it := range m.capabilities {
		if it.selected && selectedAbstracts[it.artifact.Implements] {
			out = append(out, it.artifact)
		}
	}
	return out
}

// Bindings returns, per composed abstract skill, the capability chosen for each
// contract exactly as the user set it — a contract left without an
// implementation is omitted. These explicit choices are the source of truth for
// the manifest and AGENTS.md, so an unset contract is never re-bound.
func (m Model) Bindings() map[artifact.Identity]map[string]string {
	out := make(map[artifact.Identity]map[string]string)
	for _, view := range m.compositions {
		bound := make(map[string]string)
		for _, choice := range view.contracts {
			if choice.chosen >= 0 {
				bound[choice.contract] = choice.candidates[choice.chosen].Name
			}
		}
		out[view.abstract.Identity()] = bound
	}
	return out
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureVisible()
	case tea.KeyMsg:
		if m.detail != nil {
			return m.handleDetailKey(msg)
		}
		switch m.step {
		case stepCompose:
			return m.handleComposeKey(msg)
		case stepConfirm:
			return m.handleConfirmKey(msg)
		default:
			return m.handleKey(msg)
		}
	}
	return m, nil
}

// handleDetailKey handles keys while the info page is open: most keys close it
// and return to the list; only ctrl+c quits the program.
func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "q", "i", "enter", "left", "h", "backspace":
		m.detail = nil
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
	case "left", "h", "shift+tab":
		m.switchTab(-1)
	case "right", "l", "tab":
		m.switchTab(1)
	case "g", "home":
		if idx := m.activeIndices(); len(idx) > 0 {
			m.cursor = idx[0]
		}
	case "G", "end":
		if idx := m.activeIndices(); len(idx) > 0 {
			m.cursor = idx[len(idx)-1]
		}
	case " ", "x":
		if len(m.items) > 0 {
			m.items[m.cursor].selected = !m.items[m.cursor].selected
		}
	case "a":
		m.toggleSection()
	case "v", "V":
		m.toggleLocalize()
	case "i":
		m.openDetail()
	case "enter":
		// Enter always advances the wizard: it composes each selected abstract
		// in turn, then reaches the confirmation step. With no abstracts
		// selected it saves immediately. (This is also how you save when only
		// abstract skills are selected.)
		return m.startWizard()
	}
	m.ensureVisible()
	return m, nil
}

// openDetail loads the info page for the artifact under the cursor.
func (m *Model) openDetail() {
	if len(m.items) == 0 {
		return
	}
	a := m.items[m.cursor].artifact
	excerpt, err := loadExcerpt(a.EntryPath)
	m.detail = &detailView{artifact: a, excerpt: excerpt, err: err}
}

func (m *Model) moveCursor(delta int) {
	idx := m.activeIndices()
	if len(idx) == 0 {
		return
	}
	pos := 0
	for p, i := range idx {
		if i == m.cursor {
			pos = p
			break
		}
	}
	m.cursor = idx[clampIndex(pos+delta, len(idx))]
}

// switchTab moves to the next or previous non-empty tab, wrapping, and puts the
// cursor on its first row.
func (m *Model) switchTab(delta int) {
	kinds := m.tabKinds()
	if len(kinds) == 0 {
		return
	}
	cur := 0
	for i, kind := range kinds {
		if kind == m.activeKind {
			cur = i
			break
		}
	}
	cur = ((cur+delta)%len(kinds) + len(kinds)) % len(kinds)
	m.activeKind = kinds[cur]
	m.offset = 0
	if idx := m.activeIndices(); len(idx) > 0 {
		m.cursor = idx[0]
	}
}

// toggleLocalize flags (or unflags) the artifact under the cursor for a
// committed local copy. Already-local artifacts have nothing to copy. Localizing
// also selects the artifact, since you clearly want it active.
func (m *Model) toggleLocalize() {
	if len(m.items) == 0 {
		return
	}
	it := &m.items[m.cursor]
	if it.artifact.Source == artifact.SourceLocal && !it.localized {
		return // already local content
	}
	it.localized = !it.localized
	if it.localized {
		it.selected = true
	}
}

// Localized returns the identities of artifacts the user asked to copy locally.
func (m Model) Localized() []artifact.Identity {
	var out []artifact.Identity
	for _, it := range m.items {
		if it.localized {
			out = append(out, it.artifact.Identity())
		}
	}
	return out
}

// toggleSection flips every item in the active tab (all on if any is off).
func (m *Model) toggleSection() {
	anyOff := false
	for _, it := range m.items {
		if it.artifact.Kind == m.activeKind && !it.selected {
			anyOff = true
			break
		}
	}
	for index := range m.items {
		if m.items[index].artifact.Kind == m.activeKind {
			m.items[index].selected = anyOff
		}
	}
}

// --- layout helpers ---

// compact reports whether the terminal is too narrow for the ASCII wordmark.
func (m Model) compact() bool {
	return (m.width - 2*hPad) < logoCols+24
}

// headerHeight is the number of rows the header occupies for the current width.
func (m Model) headerHeight() int {
	if m.compact() {
		return compactRows
	}
	return headerRows
}

// contentHeight is the number of list rows available inside the titled box,
// between the header and the footer.
func (m Model) contentHeight() int {
	h := m.height - 2*vPad - m.headerHeight() - footerRows - boxChromeRows
	if h < 1 {
		return 1
	}
	return h
}

// cursorVisualIndex is the cursor's row index within the active tab's list.
func (m Model) cursorVisualIndex() int {
	for pos, i := range m.activeIndices() {
		if i == m.cursor {
			return pos
		}
	}
	return 0
}

// ensureVisible scrolls the body so the cursor row stays within view.
func (m *Model) ensureVisible() {
	ch := m.contentHeight()
	ci := m.cursorVisualIndex()
	if ci < m.offset {
		m.offset = ci
	}
	if ci >= m.offset+ch {
		m.offset = ci - ch + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	inner := m.width - 2*hPad

	var doc string
	switch {
	case m.detail != nil:
		doc = m.renderDetail(inner)
	case m.step == stepCompose:
		doc = m.renderCompose(inner)
	case m.step == stepConfirm:
		doc = m.renderConfirm(inner)
	case len(m.items) == 0:
		doc = m.renderEmpty(inner)
	default:
		content := m.listWindow(inner - boxChromeCols)
		doc = lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderHeader(inner),
			m.renderTitledBox(m.tabsTitle(), inner, content),
			m.renderFooter(inner),
		)
	}

	return m.styles.base.
		Width(m.width).
		Height(m.height).
		Padding(vPad, hPad).
		Render(doc)
}

// paint renders s onto a width-fixed cell with the canvas background, so any
// alignment padding carries the background instead of the terminal default.
func (m Model) paint(width int, s string) string {
	if width < 0 {
		width = 0
	}
	return m.styles.base.Width(width).Render(s)
}

func (m Model) renderHeader(inner int) string {
	chip := m.styles.chip.Render(m.version)

	// Narrow terminals: drop the wordmark, keep a compact title + version chip.
	if m.compact() {
		title := m.paint(inner, m.styles.title.Render("harness"))
		return lipgloss.JoinVertical(lipgloss.Left, m.paint(inner, chip), title)
	}

	// Left column: the wordmark, one painted cell per line.
	var leftRows []string
	for _, line := range strings.Split(logoArt, "\n") {
		leftRows = append(leftRows, m.paint(logoCols, m.styles.logo.Render(line)))
	}

	// Right column: the version chip, a light gap, then the instructions.
	instructions := []string{
		chip,
		"",
		m.styles.title.Render("Select artifacts"),
		m.styles.subtitle.Render("for this project"),
		m.styles.subtitle.Render(m.tally()),
	}
	rightWidth := 0
	for _, row := range instructions {
		if w := lipgloss.Width(row); w > rightWidth {
			rightWidth = w
		}
	}

	gap := inner - logoCols - rightWidth
	if gap < 1 {
		gap = 1
	}

	// Assemble row by row so every cell carries the canvas background.
	rows := make([]string, headerRows)
	for i := range rows {
		rightCell := ""
		if i < len(instructions) {
			rightCell = instructions[i]
		}
		rows[i] = leftRows[i] + m.paint(gap, "") +
			m.styles.base.Width(rightWidth).Align(lipgloss.Right).Render(rightCell)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m Model) tally() string {
	selected := 0
	for _, it := range m.items {
		if it.selected {
			selected++
		}
	}
	return fmt.Sprintf("%d available · %d selected", len(m.items), selected)
}

// listWindow returns exactly contentHeight rows of the (windowed) artifact list,
// each `width` columns wide, including a scrollbar when the list overflows.
func (m Model) listWindow(width int) []string {
	ch := m.contentHeight()
	total := m.bodyLineCount()

	// Fits without scrolling: render full width, pad to the content height.
	if total <= ch {
		window := m.bodyLines(width)
		for len(window) < ch {
			window = append(window, m.paint(width, ""))
		}
		return window
	}

	// Scrollable: reserve a gap column and a scrollbar column on the right.
	const gapWidth, barWidth = 1, 1
	lines := m.bodyLines(width - gapWidth - barWidth)

	start := m.offset
	if start > total-ch {
		start = total - ch
	}
	if start < 0 {
		start = 0
	}

	bar := m.scrollbar(ch, total, start)
	rows := make([]string, ch)
	for i := 0; i < ch; i++ {
		rows[i] = lines[start+i] + m.paint(gapWidth, "") + bar[i]
	}
	return rows
}

// renderTitledBox draws content inside a bordered box with title embedded in the
// top border — mirroring the devstation TitledBox primitive. title is already
// rendered (a chip, or the category tabs). width is the box's outer width; each
// content line must be (width-boxChromeCols) wide.
func (m Model) renderTitledBox(title string, width int, content []string) string {
	border := m.styles.divider
	interior := width - 2
	if interior < 0 {
		interior = 0
	}

	dashes := interior - lipgloss.Width(title)
	if dashes < 0 {
		dashes = 0
	}
	top := border.Render("┌") + title + border.Render(strings.Repeat("─", dashes)+"┐")
	blank := border.Render("│") + m.paint(interior, "") + border.Render("│")
	bottom := border.Render("└" + strings.Repeat("─", interior) + "┘")

	rows := make([]string, 0, len(content)+4)
	rows = append(rows, top, blank)
	for _, line := range content {
		rows = append(rows, border.Render("│ ")+line+border.Render(" │"))
	}
	rows = append(rows, blank, bottom)
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// bodyLineCount is the number of rows in the active tab.
func (m Model) bodyLineCount() int {
	return len(m.activeIndices())
}

// scrollbar builds a ch-row vertical scrollbar: a proportional thumb over a
// track, positioned by the current offset.
func (m Model) scrollbar(ch, total, offset int) []string {
	thumb := ch * ch / total
	if thumb < 1 {
		thumb = 1
	}
	if thumb > ch {
		thumb = ch
	}
	pos := 0
	if maxOffset := total - ch; maxOffset > 0 {
		pos = offset * (ch - thumb) / maxOffset
	}
	cells := make([]string, ch)
	for i := range cells {
		if i >= pos && i < pos+thumb {
			cells[i] = m.styles.scrollThumb.Render("█")
		} else {
			cells[i] = m.styles.scrollTrack.Render("│")
		}
	}
	return cells
}

// bodyLines renders the active tab's rows, one full-width line each.
func (m Model) bodyLines(inner int) []string {
	var lines []string
	for _, i := range m.activeIndices() {
		lines = append(lines, m.renderRow(i, m.items[i], inner))
	}
	return lines
}

// tabsTitle renders the category tabs to embed in the list box's top border: one
// label per non-empty kind with its selected/total count, the active tab
// accented and underlined.
func (m Model) tabsTitle() string {
	var builder strings.Builder
	builder.WriteString(m.paint(1, "")) // leading space inside the border
	for i, kind := range m.tabKinds() {
		if i > 0 {
			builder.WriteString(m.paint(2, "")) // gap between tabs
		}
		selected, total := m.sectionCounts(kind)
		label := fmt.Sprintf("%s %d/%d", kind.Title(), selected, total)
		if kind == m.activeKind {
			builder.WriteString(m.styles.tabActive.Render(label))
		} else {
			builder.WriteString(m.styles.tabInactive.Render(label))
		}
	}
	builder.WriteString(m.paint(1, "")) // trailing space
	return builder.String()
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

func (m Model) renderRow(index int, it item, inner int) string {
	cursor := m.styles.base.Render("  ")
	if index == m.cursor {
		cursor = m.styles.cursor.Render("› ")
	}

	checkbox := m.styles.checkOff.Render("[ ]")
	if it.selected {
		checkbox = m.styles.checkOn.Render("[x]")
	}

	nameStyle := m.styles.name
	if index == m.cursor {
		nameStyle = m.styles.nameActive
	}
	// Fixed-width columns make the rows line up like a borderless table.
	nameCell := nameStyle.Width(m.nameWidth).MaxWidth(m.nameWidth).
		Render(truncate(it.artifact.Name, m.nameWidth))

	label, sourceStyle := "shared", m.styles.badgeShared
	switch {
	case it.localized:
		label, sourceStyle = "→local", m.styles.override
	case it.artifact.Source == artifact.SourceLocal && it.artifact.OverridesShared:
		label, sourceStyle = "local*", m.styles.override
	case it.artifact.Source == artifact.SourceLocal:
		label, sourceStyle = "local", m.styles.badgeLocal
	}
	sourceCell := sourceStyle.Width(sourceColWidth).MaxWidth(sourceColWidth).Render(label)

	prefix := cursor + checkbox + m.styles.base.Render(" ") + nameCell +
		m.styles.base.Render("  ") + sourceCell + m.styles.base.Render("  ")
	descWidth := inner - lipgloss.Width(prefix)
	if descWidth < 0 {
		descWidth = 0
	}
	desc := m.styles.description.Render(truncate(it.artifact.Description, descWidth))
	return m.styles.base.Width(inner).MaxWidth(inner).Render(prefix + desc)
}

func (m Model) renderFooter(inner int) string {
	help := "↑/↓ rows · ←/→ tabs · space select · v localize · i info · enter continue · q quit"
	scroll := ""
	if total := m.bodyLineCount(); total > m.contentHeight() {
		scroll = fmt.Sprintf(
			"  %d–%d/%d",
			min(m.offset+1, total),
			min(m.offset+m.contentHeight(), total),
			total,
		)
	}
	left := m.styles.footer.Render(help)

	right := m.styles.scrollInfo.Render(scroll)
	if m.warnings > 0 {
		warn := m.styles.warn.Render(fmt.Sprintf("⚠ %d invalid", m.warnings))
		if scroll != "" {
			right = warn + m.styles.base.Render("   ") + right
		} else {
			right = warn
		}
	}

	gap := inner - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return m.styles.base.Width(inner).MaxWidth(inner).Render(left)
	}
	return m.styles.base.Width(inner).MaxWidth(inner).Render(
		left + m.styles.base.Render(strings.Repeat(" ", gap)) + right,
	)
}

func (m Model) renderEmpty(inner int) string {
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.logo.Render(logoArt),
		m.styles.base.Render(""),
		m.styles.empty.Render("No artifacts found."),
		m.styles.empty.Render("Run `harness init` to seed your shared library (~/.harness),"),
		m.styles.empty.Render("or add artifacts under .agents/."),
		m.styles.base.Render(""),
		m.styles.footer.Render("q quit"),
	)
	return m.styles.base.Width(inner).Render(body)
}

// renderDetail draws the info page for the selected artifact: title, kind/source,
// the full description and a short excerpt from the source document.
func (m Model) renderDetail(inner int) string {
	d := m.detail

	// lipgloss Width wraps long text to the column and paints the background.
	wrap := func(style lipgloss.Style, text string) string {
		return style.Width(inner).Render(text)
	}

	var about string
	switch {
	case d.err != nil:
		about = wrap(m.styles.subtitle, "(could not read source: "+d.err.Error()+")")
	case strings.TrimSpace(d.excerpt) == "":
		about = wrap(m.styles.subtitle, "(no further details in the source document)")
	default:
		about = wrap(m.styles.description, d.excerpt)
	}

	parts := []string{
		m.paint(inner, m.styles.nameActive.Render(d.artifact.Name)),
		m.paint(inner, m.styles.subtitle.Render(fmt.Sprintf("%s · %s", d.artifact.Kind.Title(), d.artifact.Source))),
		m.styles.divider.Render(strings.Repeat("─", max(inner, 0))),
		m.paint(inner, m.styles.section.Render("DESCRIPTION")),
		wrap(m.styles.base, d.artifact.Description),
		m.paint(inner, ""),
		m.paint(inner, m.styles.section.Render("ABOUT")),
		about,
	}
	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Clamp to the available height, leaving one row for the footer hint.
	avail := m.height - 2*vPad - footerRows
	if avail < 1 {
		avail = 1
	}
	lines := strings.Split(content, "\n")
	if len(lines) > avail {
		lines = lines[:avail]
	}
	for len(lines) < avail {
		lines = append(lines, m.paint(inner, ""))
	}
	lines = append(lines, m.paint(inner, m.styles.footer.Render("i/esc back · ctrl+c quit")))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// loadExcerpt reads an artifact's source document and returns its introductory
// prose for the info page.
func loadExcerpt(entryPath string) (string, error) {
	content, err := os.ReadFile(entryPath)
	if err != nil {
		return "", err
	}
	_, body, err := artifact.ParseDocument(content)
	if err != nil {
		return "", err
	}
	return excerpt(body, 10), nil
}

// excerpt returns the introductory prose of a document body: the lines before
// the first sub-heading, table, code block or blockquote, skipping a leading H1
// title, capped at maxLines non-empty lines.
func excerpt(body string, maxLines int) string {
	var out []string
	nonEmpty := 0
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "# "):
			continue // skip the H1 title
		case strings.HasPrefix(trimmed, "#"),
			strings.HasPrefix(trimmed, "|"),
			strings.HasPrefix(trimmed, "```"),
			strings.HasPrefix(trimmed, "> "):
			if nonEmpty > 0 {
				return strings.TrimSpace(strings.Join(out, "\n"))
			}
			continue // skip leading structural lines before any prose
		}
		out = append(out, line)
		if trimmed != "" {
			nonEmpty++
			if nonEmpty >= maxLines {
				break
			}
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// truncate shortens text to width runes, appending an ellipsis when cut.
func truncate(text string, width int) string {
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	if width <= 1 {
		if width == 1 {
			return "…"
		}
		return ""
	}
	return string(runes[:width-1]) + "…"
}
