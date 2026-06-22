package tui

import "github.com/charmbracelet/lipgloss"

// palette keeps the interface calm and consistent: one accent, restrained
// neutrals, and two quiet hues to distinguish artifact sources.
var (
	accent    = lipgloss.AdaptiveColor{Light: "#4C5BD4", Dark: "#8A93FF"}
	textColor = lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#EDEDED"}
	muted     = lipgloss.AdaptiveColor{Light: "#7A7A7A", Dark: "#8A8A8A"}
	faint     = lipgloss.AdaptiveColor{Light: "#A0A0A0", Dark: "#6A6A6A"}
	sharedHue = lipgloss.AdaptiveColor{Light: "#2C7A7B", Dark: "#5EC8C9"}
	localHue  = lipgloss.AdaptiveColor{Light: "#B7791F", Dark: "#E0A85A"}
	success   = lipgloss.AdaptiveColor{Light: "#2F855A", Dark: "#68D391"}
)

type styles struct {
	app         lipgloss.Style
	title       lipgloss.Style
	subtitle    lipgloss.Style
	section     lipgloss.Style
	cursor      lipgloss.Style
	name        lipgloss.Style
	nameActive  lipgloss.Style
	checkOn     lipgloss.Style
	checkOff    lipgloss.Style
	description lipgloss.Style
	badgeShared lipgloss.Style
	badgeLocal  lipgloss.Style
	override    lipgloss.Style
	footer      lipgloss.Style
	count       lipgloss.Style
	empty       lipgloss.Style
}

func newStyles() styles {
	return styles{
		app:         lipgloss.NewStyle().Padding(1, 2),
		title:       lipgloss.NewStyle().Bold(true).Foreground(accent),
		subtitle:    lipgloss.NewStyle().Foreground(muted),
		section:     lipgloss.NewStyle().Bold(true).Foreground(textColor).MarginTop(1),
		cursor:      lipgloss.NewStyle().Foreground(accent).Bold(true),
		name:        lipgloss.NewStyle().Foreground(textColor),
		nameActive:  lipgloss.NewStyle().Foreground(accent).Bold(true),
		checkOn:     lipgloss.NewStyle().Foreground(success).Bold(true),
		checkOff:    lipgloss.NewStyle().Foreground(faint),
		description: lipgloss.NewStyle().Foreground(muted),
		badgeShared: lipgloss.NewStyle().Foreground(sharedHue),
		badgeLocal:  lipgloss.NewStyle().Foreground(localHue),
		override:    lipgloss.NewStyle().Foreground(localHue).Italic(true),
		footer:      lipgloss.NewStyle().Foreground(faint).MarginTop(1),
		count:       lipgloss.NewStyle().Foreground(success),
		empty:       lipgloss.NewStyle().Foreground(muted).MarginTop(1),
	}
}
