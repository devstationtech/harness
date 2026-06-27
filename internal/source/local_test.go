package source_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/source"
)

// writeArtifact lays down a minimal valid artifact under base for the given kind.
func writeArtifact(t *testing.T, base string, kind artifact.Kind, name, description string) {
	t.Helper()
	dir := filepath.Join(base, kind.Container(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, kind.EntryFile()), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLocalDirectoryResolvesArtifacts(t *testing.T) {
	// @Given a base with artifacts of several kinds
	base := t.TempDir()
	writeArtifact(t, base, artifact.KindSkill, "cqrs", "a skill")
	writeArtifact(t, base, artifact.KindRule, "hexagonal", "a rule")

	// @When the directory is resolved
	got, issues, err := source.NewLocalDirectory("home", base, artifact.SourceShared).Resolve()
	if err != nil {
		t.Fatal(err)
	}

	// @Then every artifact is found, tagged with the source, and free of issues
	if len(got) != 2 {
		t.Fatalf("resolved %d artifacts, want 2", len(got))
	}
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	for _, a := range got {
		if a.Source != artifact.SourceShared {
			t.Errorf("artifact %q source = %q, want shared", a.Name, a.Source)
		}
		if a.Directory == "" || a.EntryPath == "" {
			t.Errorf("artifact %q missing directory/entry path", a.Name)
		}
	}
}

func TestLocalDirectoryReportsNameMismatchAsIssue(t *testing.T) {
	// @Given a directory whose frontmatter name differs from the folder
	base := t.TempDir()
	dir := filepath.Join(base, "skills", "local")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: code-reviewer\ndescription: Mismatched name.\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// @When the directory is resolved
	got, issues, err := source.NewLocalDirectory("home", base, artifact.SourceShared).Resolve()
	if err != nil {
		t.Fatal(err)
	}

	// @Then no artifact is produced, but the mismatch is surfaced as an issue
	if len(got) != 0 {
		t.Errorf("resolved %d artifacts, want 0", len(got))
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if !strings.Contains(issues[0].Reason, "code-reviewer") || !strings.Contains(issues[0].Reason, "local") {
		t.Errorf("reason should mention the name/directory mismatch: %q", issues[0].Reason)
	}
}

func TestLocalDirectoryIgnoresDirectoryWithoutEntryDocument(t *testing.T) {
	// @Given a directory under skills/ with no SKILL.md
	base := t.TempDir()
	dir := filepath.Join(base, "skills", "not-a-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// @When the directory is resolved
	got, issues, err := source.NewLocalDirectory("home", base, artifact.SourceShared).Resolve()
	if err != nil {
		t.Fatal(err)
	}

	// @Then it is ignored silently, producing neither an artifact nor an issue
	if len(got) != 0 || len(issues) != 0 {
		t.Errorf("got %d artifacts and %d issues, want 0 and 0", len(got), len(issues))
	}
}

func TestLocalDirectoryEmptyOrMissingBaseResolvesToNothing(t *testing.T) {
	// @Given an empty base and a non-existent base
	cases := map[string]string{
		"empty":   "",
		"missing": filepath.Join(t.TempDir(), "does-not-exist"),
	}
	for name, base := range cases {
		t.Run(name, func(t *testing.T) {
			// @When the directory is resolved
			got, issues, err := source.NewLocalDirectory("home", base, artifact.SourceShared).Resolve()
			// @Then it succeeds with nothing
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 0 || len(issues) != 0 {
				t.Errorf("got %d artifacts and %d issues, want 0 and 0", len(got), len(issues))
			}
		})
	}
}
