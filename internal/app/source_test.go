package app_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devstationtech/harness/internal/app"
	"github.com/devstationtech/harness/internal/source/gitcli"
)

// gitFixture creates a throwaway git repository containing one skill.
func gitFixture(t *testing.T) string {
	t.Helper()
	if err := gitcli.Available(); err != nil {
		t.Skip("git not available")
	}
	repo := t.TempDir()
	ctx := context.Background()
	run := func(args ...string) {
		if _, err := gitcli.Run(ctx, repo, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Harness Test")

	skill := filepath.Join(repo, "skills", "reviewer")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: reviewer\ndescription: d\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "seed")
	return repo
}

func TestSourceAddListRemove(t *testing.T) {
	// @Given a git repository and an isolated shared home
	origin := gitFixture(t)
	home := t.TempDir()
	t.Setenv("HARNESS_HOME", home)

	// @When a source is added
	var out bytes.Buffer
	if err := app.Source(&out, []string{"add", "file://" + origin, "--name", "mine"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	// @Then it is cloned and reported
	if !strings.Contains(out.String(), `Added source "mine"`) {
		t.Errorf("unexpected add output: %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(home, "sources", "mine", "skills", "reviewer", "SKILL.md")); err != nil {
		t.Errorf("expected the clone to contain the skill: %v", err)
	}

	// @And it appears in the list
	out.Reset()
	if err := app.Source(&out, []string{"list"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "mine") {
		t.Errorf("list is missing the source: %q", out.String())
	}

	// @And adding the same name again is rejected
	if err := app.Source(&bytes.Buffer{}, []string{"add", "file://" + origin, "--name", "mine"}); err == nil {
		t.Error("expected a duplicate add to fail")
	}

	// @When the source is removed
	out.Reset()
	if err := app.Source(&out, []string{"remove", "mine"}); err != nil {
		t.Fatalf("remove: %v", err)
	}

	// @Then its working copy is gone and the list is empty again
	if _, err := os.Stat(filepath.Join(home, "sources", "mine")); !os.IsNotExist(err) {
		t.Error("expected the clone directory to be removed")
	}
	out.Reset()
	if err := app.Source(&out, []string{"list"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No sources configured") {
		t.Errorf("expected an empty list after removal: %q", out.String())
	}
}
