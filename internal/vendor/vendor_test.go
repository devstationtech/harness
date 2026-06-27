package vendor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/vendor"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// resolvedSkill builds a skill directory (with a nested reference) and returns
// it as a resolved artifact tagged with a remote origin.
func resolvedSkill(t *testing.T) artifact.Artifact {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "skills", "reviewer")
	write(t, filepath.Join(dir, "SKILL.md"), "---\nname: reviewer\ndescription: d\n---\nbody\n")
	write(t, filepath.Join(dir, "references", "guide.md"), "guide\n")
	return artifact.Artifact{
		Kind:      artifact.KindSkill,
		Name:      "reviewer",
		Source:    artifact.SourceShared,
		Origin:    "mine",
		Directory: dir,
		EntryPath: filepath.Join(dir, "SKILL.md"),
	}
}

func TestVendorCopiesTreeAndPinsLockEntry(t *testing.T) {
	// @Given a resolved artifact from a remote source
	a := resolvedSkill(t)
	project := t.TempDir()

	// @When it is vendored
	vendored, entry, err := vendor.Vendor(a, project, "abc123")
	if err != nil {
		t.Fatal(err)
	}

	// @Then the whole tree is copied under .agents, including nested files
	dest := filepath.Join(project, ".agents", "skills", "reviewer")
	if _, err := os.Stat(filepath.Join(dest, "references", "guide.md")); err != nil {
		t.Errorf("nested reference not copied: %v", err)
	}

	// @And the artifact now points at the local copy
	if vendored.Directory != dest {
		t.Errorf("vendored dir = %q, want %q", vendored.Directory, dest)
	}
	if vendored.Source != artifact.SourceLocal {
		t.Errorf("vendored source = %q, want local", vendored.Source)
	}

	// @And the lock entry pins the source, commit, path and a content hash
	if entry.Source != "mine" || entry.Commit != "abc123" || entry.Path != "skills/reviewer" {
		t.Errorf("unexpected lock entry: %+v", entry)
	}
	if entry.ContentHash == "" {
		t.Error("expected a content hash")
	}
}

func TestVendorIsIdempotent(t *testing.T) {
	// @Given an artifact already vendored once
	a := resolvedSkill(t)
	project := t.TempDir()
	_, first, err := vendor.Vendor(a, project, "abc123")
	if err != nil {
		t.Fatal(err)
	}

	// @When it is vendored again from the same source revision
	_, second, err := vendor.Vendor(a, project, "abc123")
	if err != nil {
		t.Fatal(err)
	}

	// @Then the content hash is unchanged
	if first.ContentHash != second.ContentHash {
		t.Errorf("re-vendor changed the hash: %s vs %s", first.ContentHash, second.ContentHash)
	}
}
