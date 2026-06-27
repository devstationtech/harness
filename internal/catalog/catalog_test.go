package catalog_test

import (
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/catalog"
	"github.com/devstationtech/harness/internal/source"
)

// fakeSource is a hand-written test double over the source.Source port, used to
// exercise the catalog's merge and precedence logic in isolation from disk.
type fakeSource struct {
	name      string
	artifacts []artifact.Artifact
	issues    []source.Issue
}

func (f fakeSource) Name() string { return f.name }

func (f fakeSource) Resolve() ([]artifact.Artifact, []source.Issue, error) {
	return f.artifacts, f.issues, nil
}

func art(kind artifact.Kind, name, description string, tag artifact.Source) artifact.Artifact {
	return artifact.Artifact{Kind: kind, Name: name, Description: description, Source: tag}
}

func TestLoadMergesAllSources(t *testing.T) {
	// @Given two sources with distinct artifacts
	home := fakeSource{name: "home", artifacts: []artifact.Artifact{
		art(artifact.KindSkill, "cqrs", "shared cqrs", artifact.SourceShared),
		art(artifact.KindRule, "hexagonal", "shared rule", artifact.SourceShared),
	}}
	project := fakeSource{name: "local", artifacts: []artifact.Artifact{
		art(artifact.KindSkill, "legacy-import", "local only", artifact.SourceLocal),
	}}

	// @When the catalog loads them (project precedence first)
	cat, err := catalog.Load(project, home)
	if err != nil {
		t.Fatal(err)
	}

	// @Then every artifact from both sources is present
	if got := len(cat.All()); got != 3 {
		t.Fatalf("merged count = %d, want 3", got)
	}
	if _, ok := cat.Find(artifact.Identity{Kind: artifact.KindSkill, Name: "legacy-import"}); !ok {
		t.Error("expected local-only artifact to be present")
	}
}

func TestLoadHigherPrecedenceWins(t *testing.T) {
	// @Given the same identity in a higher- and a lower-precedence source
	home := fakeSource{name: "home", artifacts: []artifact.Artifact{
		art(artifact.KindSkill, "cqrs", "shared description", artifact.SourceShared),
	}}
	project := fakeSource{name: "local", artifacts: []artifact.Artifact{
		art(artifact.KindSkill, "cqrs", "local description", artifact.SourceLocal),
	}}

	// @When the catalog loads with the project first
	cat, err := catalog.Load(project, home)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the higher-precedence artifact wins and is flagged as an override
	resolved, ok := cat.Find(artifact.Identity{Kind: artifact.KindSkill, Name: "cqrs"})
	if !ok {
		t.Fatal("cqrs not found")
	}
	if resolved.Source != artifact.SourceLocal {
		t.Errorf("source = %q, want local", resolved.Source)
	}
	if resolved.Description != "local description" {
		t.Errorf("description = %q, want local description", resolved.Description)
	}
	if !resolved.OverridesShared {
		t.Error("expected OverridesShared to be true")
	}
}

func TestLoadIsOrderedByKindThenName(t *testing.T) {
	// @Given artifacts of several kinds out of order
	src := fakeSource{name: "home", artifacts: []artifact.Artifact{
		art(artifact.KindSkill, "zeta", "s", artifact.SourceShared),
		art(artifact.KindSkill, "alpha", "s", artifact.SourceShared),
		art(artifact.KindRule, "rule-one", "r", artifact.SourceShared),
		art(artifact.KindAgent, "agent-one", "a", artifact.SourceShared),
	}}

	// @When the catalog loads
	cat, err := catalog.Load(src)
	if err != nil {
		t.Fatal(err)
	}

	// @Then results are grouped rules, skills, agents and sorted by name within
	want := []string{"rule-one", "alpha", "zeta", "agent-one"}
	all := cat.All()
	if len(all) != len(want) {
		t.Fatalf("count = %d, want %d", len(all), len(want))
	}
	for i, name := range want {
		if all[i].Name != name {
			t.Errorf("position %d = %q, want %q", i, all[i].Name, name)
		}
	}
}

func TestLoadCollectsIssuesFromAllSources(t *testing.T) {
	// @Given a source that reports an issue
	src := fakeSource{name: "home", issues: []source.Issue{
		{Path: "skills/local/SKILL.md", Reason: "name mismatch"},
	}}

	// @When the catalog loads
	cat, err := catalog.Load(src)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the issue is surfaced
	if len(cat.Issues()) != 1 {
		t.Fatalf("got %d issues, want 1", len(cat.Issues()))
	}
}

func TestLoadWithNoSources(t *testing.T) {
	// @Given no sources
	// @When the catalog loads
	cat, err := catalog.Load()
	// @Then it succeeds and is empty
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cat.All()) != 0 {
		t.Errorf("expected empty catalog, got %d", len(cat.All()))
	}
}
