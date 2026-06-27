package tui

import (
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
)

func skill(name string) artifact.Artifact {
	return artifact.Artifact{Kind: artifact.KindSkill, Name: name}
}

func abstract(name string, contracts ...string) artifact.Artifact {
	a := skill(name)
	a.Contracts = contracts
	return a
}

func capability(name, implements string, provides ...string) artifact.Artifact {
	a := skill(name)
	a.Implements = implements
	a.Provides = provides
	return a
}

func chooseByName(t *testing.T, view *composeView, contract, capabilityName string) {
	t.Helper()
	for i := range view.contracts {
		if view.contracts[i].contract != contract {
			continue
		}
		for j, candidate := range view.contracts[i].candidates {
			if candidate.Name == capabilityName {
				view.contracts[i].chosen = j
				return
			}
		}
		t.Fatalf("capability %q is not a candidate for contract %q", capabilityName, contract)
	}
	t.Fatalf("contract %q not found", contract)
}

func selectedNames(m Model) map[string]bool {
	out := map[string]bool{}
	for _, it := range m.items {
		if it.selected {
			out[it.artifact.Name] = true
		}
	}
	return out
}

func TestOpenComposeBuildsContractsWithCandidates(t *testing.T) {
	// @Given an abstract and two capabilities, cursor on the abstract
	m := New([]artifact.Artifact{
		abstract("lld", "domain", "persistence"),
		capability("lld-ts", "lld", "domain", "persistence"),
		capability("lld-go", "lld", "domain"),
	}, nil, "v", 0)

	// @When the composition screen opens
	m.openCompose()

	// @Then it has one entry per contract with the right candidates
	if m.compose == nil || len(m.compose.contracts) != 2 {
		t.Fatalf("compose not built: %+v", m.compose)
	}
	if got := len(m.compose.contracts[0].candidates); got != 2 { // domain: ts + go
		t.Errorf("domain candidates = %d, want 2", got)
	}
	if got := len(m.compose.contracts[1].candidates); got != 1 { // persistence: ts only
		t.Errorf("persistence candidates = %d, want 1", got)
	}
	// @And the abstract itself is now selected
	if !selectedNames(m)["lld"] {
		t.Error("abstract should be selected on compose")
	}
}

func TestApplyComposeSelectsOnlyChosenCapabilities(t *testing.T) {
	// @Given a composition with two capabilities available per contract
	m := New([]artifact.Artifact{
		abstract("lld", "domain", "persistence"),
		capability("lld-ts", "lld", "domain", "persistence"),
		capability("lld-go", "lld", "domain", "persistence"),
	}, nil, "v", 0)
	m.openCompose()

	// @When lld-ts is chosen for both contracts
	chooseByName(t, m.compose, "domain", "lld-ts")
	chooseByName(t, m.compose, "persistence", "lld-ts")
	m.applyCompose()

	// @Then only lld-ts (and the abstract) are selected; lld-go is not
	selected := selectedNames(m)
	if !selected["lld"] || !selected["lld-ts"] {
		t.Errorf("expected lld and lld-ts selected, got %v", selected)
	}
	if selected["lld-go"] {
		t.Errorf("lld-go should not be selected, got %v", selected)
	}
}

func TestCycleChoiceWrapsThroughNone(t *testing.T) {
	// @Given a contract with two candidates, nothing chosen
	m := New([]artifact.Artifact{
		abstract("lld", "domain"),
		capability("lld-ts", "lld", "domain"),
		capability("lld-go", "lld", "domain"),
	}, nil, "v", 0)
	m.openCompose()

	// @When cycling forward three times
	// @Then it goes none → first → second → none
	if m.compose.contracts[0].chosen != -1 {
		t.Fatalf("initial choice = %d, want -1", m.compose.contracts[0].chosen)
	}
	m.cycleChoice(1)
	if m.compose.contracts[0].chosen != 0 {
		t.Errorf("after one cycle = %d, want 0", m.compose.contracts[0].chosen)
	}
	m.cycleChoice(1)
	m.cycleChoice(1)
	if m.compose.contracts[0].chosen != -1 {
		t.Errorf("after wrapping = %d, want -1", m.compose.contracts[0].chosen)
	}
}
