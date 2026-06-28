package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/devstationtech/harness/internal/artifact"
)

func updateModel() Model {
	return New([]artifact.Artifact{
		{Kind: artifact.KindSkill, Name: "foo", Source: artifact.SourceShared},
	}, nil, "v0.1.0", 0)
}

func TestUpdateNotificationAndKey(t *testing.T) {
	m := updateModel()

	// @When the background check reports a newer release
	updated, _ := m.Update(updateAvailableMsg{latest: "v0.2.0"})
	m = updated.(Model)

	// @Then the tag is recorded and surfaced bottom-left in the footer
	if m.updateLatest != "v0.2.0" {
		t.Fatalf("updateLatest = %q, want v0.2.0", m.updateLatest)
	}
	m.width, m.height = 120, 24
	if foot := m.renderFooter(110); !strings.Contains(foot, "v0.2.0") || !strings.Contains(foot, "press u to update") {
		t.Errorf("footer missing update note:\n%s", foot)
	}

	// @When the user presses u
	after, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = after.(Model)

	// @Then an update is requested and the program quits
	if !m.RequestUpdate() {
		t.Error("pressing u should request an update")
	}
	if cmd == nil {
		t.Fatal("pressing u should emit a command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("pressing u should quit the program")
	}
}

func TestUpdateKeyNoopWithoutRelease(t *testing.T) {
	m := updateModel()

	after, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})

	if after.(Model).RequestUpdate() {
		t.Error("u without an available update must be a no-op")
	}
	if cmd != nil {
		t.Error("u without an available update should not emit a command")
	}
}
