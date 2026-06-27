package tui

import "github.com/charmbracelet/lipgloss"

// Palette mirrors the devstation CLI (dark theme): cyan primary, gray neutrals,
// #666666 dividers, #323232/#aeaeae chips, and constant green/yellow/red status.
// The whole interface renders against a dark-gray canvas, so every text style
// also carries the canvas background to keep the fill uniform.
// All colors are fixed hex (truecolor) rather than ANSI palette indices, so they
// render identically across terminals. Palette indices (0-15 especially) are
// themed per terminal — e.g. "cyan"/14 shows bluish in GNOME Terminal but washes
// to near-white in Warp — which is why these were pinned. The hues match the
// Tango "bright" palette the devstation theme intends.
var (
	canvas  = lipgloss.Color("#1e1e1e") // dark gray full-screen background
	accent  = lipgloss.Color("#34e2e2") // cyan (devstation "primary")
	textCol = lipgloss.Color("#d0d0d0")
	muted   = lipgloss.Color("#8a8a8a")
	faint   = lipgloss.Color("#585858")
	border  = lipgloss.Color("#666666")
	chipBg  = lipgloss.Color("#323232")
	chipFg  = lipgloss.Color("#aeaeae")
	tabSel  = lipgloss.Color("#4d4d4d") // lighter chip tone for the active tab
	success = lipgloss.Color("#8ae234")
	warning = lipgloss.Color("#fce94f")
	sharedC = lipgloss.Color("#00afff") // calm blue for the "shared" source
)

const (
	hPad = 2 // horizontal canvas padding
	vPad = 1 // vertical canvas padding
)

type styles struct {
	base        lipgloss.Style
	logo        lipgloss.Style
	title       lipgloss.Style
	subtitle    lipgloss.Style
	chip        lipgloss.Style
	section     lipgloss.Style
	sectionHint lipgloss.Style
	cursor      lipgloss.Style
	name        lipgloss.Style
	nameActive  lipgloss.Style
	checkOn     lipgloss.Style
	checkOff    lipgloss.Style
	description lipgloss.Style
	badgeShared lipgloss.Style
	badgeLocal  lipgloss.Style
	override    lipgloss.Style
	divider     lipgloss.Style
	footer      lipgloss.Style
	count       lipgloss.Style
	scrollInfo  lipgloss.Style
	scrollThumb lipgloss.Style
	scrollTrack lipgloss.Style
	warn        lipgloss.Style
	empty       lipgloss.Style
	tabActive   lipgloss.Style
	tabInactive lipgloss.Style
}

func newStyles() styles {
	on := func() lipgloss.Style { return lipgloss.NewStyle().Background(canvas) }
	return styles{
		base:        on().Foreground(textCol),
		logo:        on().Foreground(accent).Bold(true),
		title:       on().Foreground(textCol).Bold(true),
		subtitle:    on().Foreground(muted),
		chip:        lipgloss.NewStyle().Background(chipBg).Foreground(chipFg).Padding(0, 1),
		section:     on().Foreground(accent).Bold(true),
		sectionHint: on().Foreground(muted),
		cursor:      on().Foreground(accent).Bold(true),
		name:        on().Foreground(textCol),
		nameActive:  on().Foreground(accent).Bold(true),
		checkOn:     on().Foreground(success).Bold(true),
		checkOff:    on().Foreground(faint),
		description: on().Foreground(muted),
		badgeShared: on().Foreground(sharedC),
		badgeLocal:  on().Foreground(warning),
		override:    on().Foreground(warning).Italic(true),
		divider:     on().Foreground(border),
		footer:      on().Foreground(faint),
		count:       on().Foreground(success),
		scrollInfo:  on().Foreground(faint),
		scrollThumb: on().Foreground(accent),
		scrollTrack: on().Foreground(border),
		warn:        on().Foreground(warning),
		empty:       on().Foreground(muted),
		tabActive:   lipgloss.NewStyle().Background(tabSel).Foreground(accent).Bold(true).Padding(0, 1),
		tabInactive: lipgloss.NewStyle().Background(chipBg).Foreground(chipFg).Padding(0, 1),
	}
}
