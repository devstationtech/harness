package lock_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/devstationtech/harness/internal/lock"
)

// writeFile lays down content at a forward-slash relative path under dir.
func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestContentHashIsStableAcrossLineEndings(t *testing.T) {
	// @Given two directories whose only difference is LF vs CRLF newlines
	lf := t.TempDir()
	crlf := t.TempDir()
	writeFile(t, lf, "SKILL.md", "line one\nline two\n")
	writeFile(t, crlf, "SKILL.md", "line one\r\nline two\r\n")

	// @When each directory is hashed
	first, err := lock.ContentHash(lf)
	if err != nil {
		t.Fatal(err)
	}
	second, err := lock.ContentHash(crlf)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the hashes are identical, proving cross-OS stability
	if first != second {
		t.Errorf("line-ending difference changed the hash: %s vs %s", first, second)
	}
}

func TestContentHashChangesWithContent(t *testing.T) {
	// @Given two directories with different file contents
	a := t.TempDir()
	b := t.TempDir()
	writeFile(t, a, "SKILL.md", "alpha\n")
	writeFile(t, b, "SKILL.md", "beta\n")

	// @When both are hashed
	ha, err := lock.ContentHash(a)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := lock.ContentHash(b)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the hashes differ
	if ha == hb {
		t.Error("different content produced the same hash")
	}
}

func TestContentHashDistinguishesPaths(t *testing.T) {
	// @Given identical content under different file names
	a := t.TempDir()
	b := t.TempDir()
	writeFile(t, a, "SKILL.md", "x\n")
	writeFile(t, b, "RULE.md", "x\n")

	// @When both are hashed
	ha, err := lock.ContentHash(a)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := lock.ContentHash(b)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the file path affects the hash
	if ha == hb {
		t.Error("expected the file path to affect the hash")
	}
}

func TestLockfileRoundTrip(t *testing.T) {
	// @Given a lockfile with one entry, saved under a not-yet-existing .agents dir
	path := filepath.Join(t.TempDir(), ".agents", "harness.lock")
	want := lock.Lockfile{Version: 1, Artifacts: []lock.Entry{
		{Kind: "skill", Name: "reviewer", Source: "mine", Commit: "abc123", ContentHash: "sha256:deadbeef", Path: "skills/reviewer"},
	}}

	// @When it is saved and reloaded
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := lock.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the reloaded lockfile equals the original
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("round-trip mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadMissingLockfileIsEmpty(t *testing.T) {
	// @Given no lockfile
	// @When loading from a non-existent path
	got, err := lock.Load(filepath.Join(t.TempDir(), "harness.lock"))
	// @Then it resolves to an empty lockfile, not an error
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Artifacts) != 0 {
		t.Errorf("expected empty lockfile, got %d entries", len(got.Artifacts))
	}
}
