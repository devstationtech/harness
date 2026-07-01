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

func selectedID(name string) map[artifact.Identity]bool {
	return map[artifact.Identity]bool{{Kind: artifact.KindSkill, Name: name}: true}
}

func chooseByName(t *testing.T, view *composeView, contract, capabilityName string) {
	t.Helper()
	for i := range view.contracts {
		if view.contracts[i].contract != contract {
			continue
		}
		for j, candidate := range view.contracts[i].candidates {
			if candidate.Name == capabilityName {
				view.contracts[i].setSingle(j)
				return
			}
		}
		t.Fatalf("capability %q is not a candidate for contract %q", capabilityName, contract)
	}
	t.Fatalf("contract %q not found", contract)
}

func selectedNames(m Model) map[string]bool {
	out := map[string]bool{}
	for _, a := range m.Selected() {
		out[a.Name] = true
	}
	return out
}

func TestStartWizardComposesSelectedAbstracts(t *testing.T) {
	// @Given a selected abstract with two contracts and two capabilities
	m := New([]artifact.Artifact{
		abstract("lld", "domain", "persistence"),
		capability("lld-ts", "lld", "domain", "persistence"),
		capability("lld-go", "lld", "domain"),
	}, selectedID("lld"), "v", 0)

	// @When the wizard starts
	next, _ := m.startWizard()
	m = next.(Model)

	// @Then it enters the compose step with one composition and the right candidates
	if m.step != stepCompose || len(m.compositions) != 1 {
		t.Fatalf("wizard not started: step=%d compositions=%d", m.step, len(m.compositions))
	}
	view := m.compositions[0]
	if len(view.contracts[0].candidates) != 2 { // domain: ts + go
		t.Errorf("domain candidates = %d, want 2", len(view.contracts[0].candidates))
	}
	if len(view.contracts[1].candidates) != 1 { // persistence: ts only
		t.Errorf("persistence candidates = %d, want 1", len(view.contracts[1].candidates))
	}
}

func TestStartWizardWithNoAbstractSavesImmediately(t *testing.T) {
	// @Given only a plain skill selected
	m := New([]artifact.Artifact{skill("plain")}, selectedID("plain"), "v", 0)

	// @When the wizard starts
	next, _ := m.startWizard()
	m = next.(Model)

	// @Then it confirms (saves) without a composition step
	if !m.Confirmed() {
		t.Error("expected an immediate save when no abstracts are selected")
	}
}

func TestApplyViewSelectsOnlyChosenCapabilities(t *testing.T) {
	// @Given a started wizard with two capabilities available
	m := New([]artifact.Artifact{
		abstract("lld", "domain", "persistence"),
		capability("lld-ts", "lld", "domain", "persistence"),
		capability("lld-go", "lld", "domain", "persistence"),
	}, selectedID("lld"), "v", 0)
	next, _ := m.startWizard()
	m = next.(Model)
	view := m.compositions[0]

	// @When lld-ts is chosen for both contracts and applied
	chooseByName(t, view, "domain", "lld-ts")
	chooseByName(t, view, "persistence", "lld-ts")
	m.applyView(view)

	// @Then only lld-ts (and the abstract) are selected
	selected := selectedNames(m)
	if !selected["lld"] || !selected["lld-ts"] {
		t.Errorf("expected lld and lld-ts, got %v", selected)
	}
	if selected["lld-go"] {
		t.Errorf("lld-go should not be selected, got %v", selected)
	}
}

func TestPriorBindingsHonorExplicitNone(t *testing.T) {
	// @Given a reopened project: the capability provides two contracts but the
	// manifest only bound it to one (the other was left "no implementation")
	m := New([]artifact.Artifact{
		abstract("lld", "domain", "command"),
		capability("lld-ts", "lld", "domain", "command"),
	}, selectedID("lld"), "v", 0)
	m.priorBindings = map[artifact.Identity]map[string][]string{
		{Kind: artifact.KindSkill, Name: "lld"}: {"domain": {"lld-ts"}},
	}

	// @When the wizard rebuilds the composition
	next, _ := m.startWizard()
	m = next.(Model)
	view := m.compositions[0]

	// @Then domain is pre-chosen and command stays unbound (none), not re-bound
	if view.contracts[0].contract != "domain" || view.contracts[0].single() != 0 {
		t.Errorf("domain not pre-chosen: %+v", view.contracts[0])
	}
	if view.contracts[1].contract != "command" || view.contracts[1].single() != -1 {
		t.Errorf("command should stay unbound, got %+v", view.contracts[1])
	}
}

func TestSelectedPrunesOrphanCapability(t *testing.T) {
	// @Given an abstract and its capability both selected (a configured project)
	m := New([]artifact.Artifact{
		abstract("lld", "domain"),
		capability("lld-go", "lld", "domain"),
	}, map[artifact.Identity]bool{
		{Kind: artifact.KindSkill, Name: "lld"}:    true,
		{Kind: artifact.KindSkill, Name: "lld-go"}: true,
	}, "v", 0)

	// @When the abstract is deselected in the list
	m.items[0].selected = false

	// @Then the capability is not selected on its own (no orphan leaks to save)
	if selectedNames(m)["lld-go"] {
		t.Error("capability should be pruned when its abstract is deselected")
	}
}

func TestCycleWrapsThroughNone(t *testing.T) {
	// @Given a contract with two candidates, nothing chosen
	m := New([]artifact.Artifact{
		abstract("lld", "domain"),
		capability("lld-ts", "lld", "domain"),
		capability("lld-go", "lld", "domain"),
	}, selectedID("lld"), "v", 0)
	next, _ := m.startWizard()
	m = next.(Model)
	view := m.compositions[0]

	// @When cycling forward through every option
	// @Then it goes none → first → second → none
	if view.contracts[0].single() != -1 {
		t.Fatalf("initial = %d, want -1", view.contracts[0].single())
	}
	cycle(view, 1)
	if view.contracts[0].single() != 0 {
		t.Errorf("after one cycle = %d, want 0", view.contracts[0].single())
	}
	cycle(view, 1)
	cycle(view, 1)
	if view.contracts[0].single() != -1 {
		t.Errorf("after wrapping = %d, want -1", view.contracts[0].single())
	}
}

// mcpAbstract and mcpCapability build a multi-select abstract MCP and a
// capability the way the github MCP artifacts do.
func mcpAbstract(name string, contracts ...string) artifact.Artifact {
	return artifact.Artifact{Kind: artifact.KindMCP, Name: name, Contracts: contracts, Multiple: true}
}

func mcpCapability(name, implements string, provides ...string) artifact.Artifact {
	return artifact.Artifact{Kind: artifact.KindMCP, Name: name, Implements: implements, Provides: provides}
}

func TestMultiSelectBindsSeveralCapabilities(t *testing.T) {
	// @Given a multi-select MCP abstract with one contract and two capabilities
	m := New([]artifact.Artifact{
		mcpAbstract("github", "target"),
		mcpCapability("github-claude-code", "github", "target"),
		mcpCapability("github-codex", "github", "target"),
	}, map[artifact.Identity]bool{{Kind: artifact.KindMCP, Name: "github"}: true}, "v", 0)
	next, _ := m.startWizard()
	m = next.(Model)
	view := m.compositions[0]
	if !view.multiple {
		t.Fatal("composition should be multi-select")
	}

	// @When both candidates are toggled on (rows 0 and 1) and applied
	view.cursor = 0
	toggle(view)
	view.cursor = 1
	toggle(view)
	m.applyView(view)

	// @Then both capabilities are bound to the contract and both are selected
	binding := m.Bindings()[artifact.Identity{Kind: artifact.KindMCP, Name: "github"}]
	if got := binding["target"]; len(got) != 2 {
		t.Fatalf("target bindings = %v, want both capabilities", got)
	}
	names := selectedNames(m)
	if !names["github-claude-code"] || !names["github-codex"] {
		t.Errorf("expected both capabilities selected, got %v", names)
	}
}
