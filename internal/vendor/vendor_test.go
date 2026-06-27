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
		Version:   "1.2.0",
		Source:    artifact.SourceShared,
		Origin:    "mine",
		Directory: dir,
		EntryPath: filepath.Join(dir, "SKILL.md"),
	}
}

func TestVendorCopiesTreeAndReturnsDigest(t *testing.T) {
	// @Given a resolved artifact from a remote source
	a := resolvedSkill(t)
	project := t.TempDir()

	// @When it is vendored
	vendored, digest, err := vendor.Vendor(a, project)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the whole tree is copied under .agents, including nested files
	dest := filepath.Join(project, ".agents", "skills", "reviewer")
	if _, err := os.Stat(filepath.Join(dest, "references", "guide.md")); err != nil {
		t.Errorf("nested reference not copied: %v", err)
	}

	// @And the artifact now points at the local copy, keeping its version
	if vendored.Directory != dest {
		t.Errorf("vendored dir = %q, want %q", vendored.Directory, dest)
	}
	if vendored.Source != artifact.SourceLocal {
		t.Errorf("vendored source = %q, want local", vendored.Source)
	}
	if vendored.Version != "1.2.0" {
		t.Errorf("vendored version = %q, want 1.2.0", vendored.Version)
	}
	if digest == "" {
		t.Error("expected a content digest")
	}
}

func TestVendorIsIdempotent(t *testing.T) {
	// @Given an artifact already vendored once
	a := resolvedSkill(t)
	project := t.TempDir()
	_, first, err := vendor.Vendor(a, project)
	if err != nil {
		t.Fatal(err)
	}

	// @When it is vendored again from the same source revision
	_, second, err := vendor.Vendor(a, project)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the digest is unchanged
	if first != second {
		t.Errorf("re-vendor changed the digest: %s vs %s", first, second)
	}
}

func TestContentHashIsStableAcrossLineEndings(t *testing.T) {
	// @Given two directories whose only difference is LF vs CRLF newlines
	lf := t.TempDir()
	crlf := t.TempDir()
	write(t, filepath.Join(lf, "SKILL.md"), "line one\nline two\n")
	write(t, filepath.Join(crlf, "SKILL.md"), "line one\r\nline two\r\n")

	// @When each is hashed
	first, err := vendor.ContentHash(lf)
	if err != nil {
		t.Fatal(err)
	}
	second, err := vendor.ContentHash(crlf)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the hashes are identical, proving cross-OS stability
	if first != second {
		t.Errorf("line-ending difference changed the hash: %s vs %s", first, second)
	}
}

func TestContentHashChangesWithContent(t *testing.T) {
	// @Given two directories with different contents
	a := t.TempDir()
	b := t.TempDir()
	write(t, filepath.Join(a, "SKILL.md"), "alpha\n")
	write(t, filepath.Join(b, "SKILL.md"), "beta\n")

	// @When both are hashed
	ha, err := vendor.ContentHash(a)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := vendor.ContentHash(b)
	if err != nil {
		t.Fatal(err)
	}

	// @Then the hashes differ
	if ha == hb {
		t.Error("different content produced the same hash")
	}
}
