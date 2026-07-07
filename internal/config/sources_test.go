package config_test

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/devstationtech/harness/internal/config"
)

func TestLoadSourcesMissingFileIsEmpty(t *testing.T) {
	// @Given no sources file on disk
	// @When loading from a non-existent path
	got, err := config.LoadSources(filepath.Join(t.TempDir(), "sources.yaml"))
	// @Then it resolves to an empty configuration, not an error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Sources) != 0 {
		t.Errorf("expected no sources, got %d", len(got.Sources))
	}
}

func TestSourcesRoundTrip(t *testing.T) {
	// @Given a sources config with one git source
	path := filepath.Join(t.TempDir(), "home", "sources.yaml")
	want := config.Sources{Sources: []config.Source{
		{Name: "mine", Type: "git", URL: "git@example.com:me/skills.git", Ref: "main"},
	}}

	// @When it is saved (creating the parent dir) and reloaded
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := config.LoadSources(path)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the reloaded config equals the original
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("round-trip mismatch (-want +got):\n%s", diff)
	}
}

func TestSourcesFind(t *testing.T) {
	// @Given a config with one source
	cfg := config.Sources{Sources: []config.Source{
		{Name: "mine", Type: "git", URL: "u"},
	}}

	// @When looking up a present and an absent name
	// @Then only the present one is found
	if _, ok := cfg.Find("mine"); !ok {
		t.Error("expected to find 'mine'")
	}
	if _, ok := cfg.Find("absent"); ok {
		t.Error("did not expect to find 'absent'")
	}
}
