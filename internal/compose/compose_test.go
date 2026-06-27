package compose_test

import (
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/compose"
)

func abstractSkill(name string, contracts ...string) artifact.Artifact {
	return artifact.Artifact{Kind: artifact.KindSkill, Name: name, Source: artifact.SourceShared, Contracts: contracts}
}

func capability(name, implements string, source artifact.Source, provides ...string) artifact.Artifact {
	return artifact.Artifact{Kind: artifact.KindSkill, Name: name, Source: source, Implements: implements, Provides: provides}
}

func bindingFor(c compose.Composition, contract string) (compose.Binding, bool) {
	for _, b := range c.Bindings {
		if b.Contract == contract {
			return b, true
		}
	}
	return compose.Binding{}, false
}

func TestBindFullComposition(t *testing.T) {
	// @Given an abstract skill and one capability providing all its contracts
	selected := []artifact.Artifact{
		abstractSkill("lld", "domain", "command"),
		capability("lld-ts", "lld", artifact.SourceShared, "domain", "command"),
	}

	// @When bound
	got := compose.Bind(selected)

	// @Then there is one complete composition with both contracts bound
	if len(got) != 1 {
		t.Fatalf("compositions = %d, want 1", len(got))
	}
	if !got[0].Complete() {
		t.Errorf("expected a complete composition, unbound = %v", got[0].Unbound)
	}
	if b, ok := bindingFor(got[0], "domain"); !ok || b.Capability.Name != "lld-ts" {
		t.Errorf("domain not bound to lld-ts: %+v", b)
	}
}

func TestBindReportsUnboundContract(t *testing.T) {
	// @Given a capability that provides only one of two contracts
	selected := []artifact.Artifact{
		abstractSkill("lld", "domain", "command"),
		capability("lld-ts", "lld", artifact.SourceShared, "domain"),
	}

	// @When bound
	got := compose.Bind(selected)

	// @Then command is unbound
	if len(got) != 1 || got[0].Complete() {
		t.Fatalf("expected an incomplete composition, got %+v", got)
	}
	if len(got[0].Unbound) != 1 || got[0].Unbound[0] != "command" {
		t.Errorf("unbound = %v, want [command]", got[0].Unbound)
	}
}

func TestBindShadowsConflictingProvider(t *testing.T) {
	// @Given two capabilities providing the same contract, one local
	selected := []artifact.Artifact{
		abstractSkill("lld", "persistence"),
		capability("lld-ts-postgres", "lld", artifact.SourceShared, "persistence"),
		capability("lld-ts-fs", "lld", artifact.SourceLocal, "persistence"),
	}

	// @When bound
	got := compose.Bind(selected)

	// @Then the local one wins and the shared one is shadowed
	b, ok := bindingFor(got[0], "persistence")
	if !ok || b.Capability.Name != "lld-ts-fs" {
		t.Fatalf("expected local capability to win, got %+v", b)
	}
	if len(b.Shadowed) != 1 || b.Shadowed[0].Name != "lld-ts-postgres" {
		t.Errorf("shadowed = %v, want [lld-ts-postgres]", b.Shadowed)
	}
}

func TestBindMixesTraitsFromDifferentCapabilities(t *testing.T) {
	// @Given two capabilities each providing a different contract
	selected := []artifact.Artifact{
		abstractSkill("lld", "domain", "persistence"),
		capability("lld-core", "lld", artifact.SourceShared, "domain"),
		capability("lld-pg", "lld", artifact.SourceShared, "persistence"),
	}

	// @When bound
	got := compose.Bind(selected)

	// @Then each contract binds to its respective capability
	if !got[0].Complete() {
		t.Fatalf("expected complete, unbound = %v", got[0].Unbound)
	}
	domain, _ := bindingFor(got[0], "domain")
	persistence, _ := bindingFor(got[0], "persistence")
	if domain.Capability.Name != "lld-core" || persistence.Capability.Name != "lld-pg" {
		t.Errorf("traits not mixed: domain=%s persistence=%s", domain.Capability.Name, persistence.Capability.Name)
	}
}

func TestBindIgnoresUnrelatedCapability(t *testing.T) {
	// @Given a capability implementing a different abstract
	selected := []artifact.Artifact{
		abstractSkill("lld", "domain"),
		capability("other", "something-else", artifact.SourceShared, "domain"),
	}

	// @When bound
	got := compose.Bind(selected)

	// @Then the unrelated capability does not bind; the contract is unbound
	if len(got) != 1 || got[0].Complete() {
		t.Fatalf("expected domain unbound, got %+v", got)
	}
}
