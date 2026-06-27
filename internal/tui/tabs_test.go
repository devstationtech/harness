package tui

import (
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
)

func TestTabsScopeNavigation(t *testing.T) {
	// @Given a rule and two skills
	m := New([]artifact.Artifact{
		{Kind: artifact.KindRule, Name: "r1"},
		{Kind: artifact.KindSkill, Name: "s1"},
		{Kind: artifact.KindSkill, Name: "s2"},
	}, nil, "v", 0)

	// @Then the first non-empty tab (Rules) is active
	if m.activeKind != artifact.KindRule {
		t.Fatalf("active tab = %v, want rule", m.activeKind)
	}

	// @When switching to the next tab
	m.switchTab(1)

	// @Then Skills is active with the cursor on its first row
	if m.activeKind != artifact.KindSkill {
		t.Fatalf("active tab = %v, want skill", m.activeKind)
	}
	if m.items[m.cursor].artifact.Name != "s1" {
		t.Errorf("cursor on %q, want s1", m.items[m.cursor].artifact.Name)
	}

	// @When moving down within the tab and past the end
	m.moveCursor(1)
	if m.items[m.cursor].artifact.Name != "s2" {
		t.Errorf("cursor on %q, want s2", m.items[m.cursor].artifact.Name)
	}
	m.moveCursor(1)

	// @Then navigation stays within the active tab (no overflow into Rules)
	if m.items[m.cursor].artifact.Name != "s2" {
		t.Errorf("cursor overflowed the tab to %q", m.items[m.cursor].artifact.Name)
	}

	// @When switching back, wrapping
	m.switchTab(-1)
	if m.activeKind != artifact.KindRule {
		t.Errorf("active tab = %v, want rule after wrap", m.activeKind)
	}
}

func TestTabKindsAreOnlyNonEmpty(t *testing.T) {
	// @Given only skills
	m := New([]artifact.Artifact{{Kind: artifact.KindSkill, Name: "s1"}}, nil, "v", 0)

	// @Then there is a single tab (Skills); empty kinds get no tab
	kinds := m.tabKinds()
	if len(kinds) != 1 || kinds[0] != artifact.KindSkill {
		t.Errorf("expected only the Skills tab, got %v", kinds)
	}
}
