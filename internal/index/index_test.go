package index_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/index"
	"github.com/devstationtech/harness/internal/source"
)

// fakeSource is a hand-written test double over the source.Source port.
type fakeSource struct {
	name      string
	artifacts []artifact.Artifact
}

func (f fakeSource) Name() string { return f.name }

func (f fakeSource) Resolve() ([]artifact.Artifact, []source.Issue, error) {
	return f.artifacts, nil, nil
}

func sampleSource(name string) fakeSource {
	return fakeSource{name: name, artifacts: []artifact.Artifact{
		{Kind: artifact.KindSkill, Name: "api-designer", Description: "Designs REST and RPC APIs"},
		{Kind: artifact.KindRule, Name: "twelve-factor", Description: "Cloud-native constraints"},
	}}
}

func TestRefreshThenSearchByName(t *testing.T) {
	// @Given an indexed source
	dir := t.TempDir()
	n, err := index.Refresh(dir, sampleSource("acme"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("indexed %d records, want 2", n)
	}

	// @When searching by a name fragment
	got, err := index.Search(dir, "designer")
	if err != nil {
		t.Fatal(err)
	}

	// @Then the matching record is returned, namespaced by source
	if len(got) != 1 || got[0].Name != "api-designer" || got[0].Source != "acme" {
		t.Fatalf("unexpected results: %+v", got)
	}
}

func TestSearchMatchesDescriptionCaseInsensitively(t *testing.T) {
	// @Given an indexed source
	dir := t.TempDir()
	if _, err := index.Refresh(dir, sampleSource("acme")); err != nil {
		t.Fatal(err)
	}

	// @When searching a description word in a different case
	got, err := index.Search(dir, "CLOUD")
	if err != nil {
		t.Fatal(err)
	}

	// @Then it matches
	if len(got) != 1 || got[0].Name != "twelve-factor" {
		t.Fatalf("unexpected results: %+v", got)
	}
}

func TestSearchEmptyQueryReturnsAllSortedBySource(t *testing.T) {
	// @Given two indexed sources
	dir := t.TempDir()
	if _, err := index.Refresh(dir, sampleSource("zeta")); err != nil {
		t.Fatal(err)
	}
	if _, err := index.Refresh(dir, sampleSource("alpha")); err != nil {
		t.Fatal(err)
	}

	// @When searching with an empty query
	got, err := index.Search(dir, "")
	if err != nil {
		t.Fatal(err)
	}

	// @Then all records are returned, the "alpha" source first
	if len(got) != 4 {
		t.Fatalf("got %d records, want 4", len(got))
	}
	if got[0].Source != "alpha" {
		t.Errorf("first source = %q, want alpha", got[0].Source)
	}
}

func TestSearchMissingIndexDirIsEmpty(t *testing.T) {
	// @Given no index directory
	// @When searching
	got, err := index.Search(filepath.Join(t.TempDir(), "absent"), "x")
	// @Then it yields no results and no error
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected no results, got %d", len(got))
	}
}

func TestRemoveDropsSourceIndex(t *testing.T) {
	// @Given two indexed sources
	dir := t.TempDir()
	if _, err := index.Refresh(dir, sampleSource("acme")); err != nil {
		t.Fatal(err)
	}
	if _, err := index.Refresh(dir, sampleSource("other")); err != nil {
		t.Fatal(err)
	}

	// @When one is removed
	if err := index.Remove(dir, "acme"); err != nil {
		t.Fatal(err)
	}

	// @Then its file is gone and only the other's records remain
	if _, err := os.Stat(filepath.Join(dir, "acme.yaml")); !os.IsNotExist(err) {
		t.Error("expected acme index file to be removed")
	}
	got, err := index.Search(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range got {
		if r.Source == "acme" {
			t.Errorf("did not expect records from the removed source: %+v", r)
		}
	}

	// @And removing a missing source is not an error
	if err := index.Remove(dir, "absent"); err != nil {
		t.Errorf("removing an absent index should be a no-op, got %v", err)
	}
}
