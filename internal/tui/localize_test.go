package tui

import (
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
)

func TestLocalizeSharedArtifact(t *testing.T) {
	// @Given a shared artifact under the cursor
	m := New([]artifact.Artifact{
		{Kind: artifact.KindSkill, Name: "foo", Source: artifact.SourceShared},
	}, nil, "v", 0)

	// @When it is localized
	m.toggleLocalize()

	// @Then it is flagged for a local copy and auto-selected
	if got := m.Localized(); len(got) != 1 || got[0].Name != "foo" {
		t.Errorf("expected foo localized, got %v", got)
	}
	if !selectedNames(m)["foo"] {
		t.Error("localize should auto-select the artifact")
	}

	// @When toggled again
	m.toggleLocalize()

	// @Then the local-copy flag is cleared
	if len(m.Localized()) != 0 {
		t.Errorf("expected localize toggled off, got %v", m.Localized())
	}
}

func TestLocalizeIgnoresAlreadyLocalArtifact(t *testing.T) {
	// @Given an artifact that is already local
	m := New([]artifact.Artifact{
		{Kind: artifact.KindSkill, Name: "bar", Source: artifact.SourceLocal},
	}, nil, "v", 0)

	// @When localize is attempted
	m.toggleLocalize()

	// @Then there is nothing to copy
	if len(m.Localized()) != 0 {
		t.Errorf("a local artifact should not be localizable, got %v", m.Localized())
	}
}
