package artifact

import "testing"

func TestParseDocument(t *testing.T) {
	// @Given an entry document with valid frontmatter and a body
	content := []byte("---\nname: cqrs\ndescription: Create a CQRS command and handler.\nmetadata:\n  version: \"1.0\"\n---\n# CQRS\n\nBody here.\n")

	// @When the document is parsed
	front, body, err := ParseDocument(content)

	// @Then the frontmatter fields and the body are returned
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if front.Name != "cqrs" {
		t.Errorf("name = %q, want cqrs", front.Name)
	}
	if front.Description != "Create a CQRS command and handler." {
		t.Errorf("description = %q", front.Description)
	}
	if front.Metadata["version"] != "1.0" {
		t.Errorf("metadata version = %q, want 1.0", front.Metadata["version"])
	}
	if body != "# CQRS\n\nBody here.\n" {
		t.Errorf("body = %q", body)
	}
}

func TestParseDocumentMissingFrontmatter(t *testing.T) {
	// @Given a document without a frontmatter block
	content := []byte("# No frontmatter here\n")

	// @When the document is parsed
	_, _, err := ParseDocument(content)

	// @Then parsing fails
	if err == nil {
		t.Fatal("expected an error for missing frontmatter")
	}
}

func TestValidateName(t *testing.T) {
	cases := map[string]bool{
		"cqrs":           true,
		"create-command": true,
		"go-service-1":   true,
		"PDF-Processing": false,
		"-leading":       false,
		"trailing-":      false,
		"double--hyphen": false,
		"":               false,
	}
	for name, valid := range cases {
		// @Given a candidate artifact name
		// @When the name is validated
		err := ValidateName(name)
		// @Then validity matches the naming rules
		if valid && err != nil {
			t.Errorf("name %q: expected valid, got %v", name, err)
		}
		if !valid && err == nil {
			t.Errorf("name %q: expected invalid, got nil", name)
		}
	}
}

func TestFrontmatterValidateNameMustMatchDirectory(t *testing.T) {
	// @Given a frontmatter whose name differs from the directory name
	front := Frontmatter{Name: "cqrs", Description: "x"}

	// @When validating against a different expected directory name
	err := front.Validate("commands")

	// @Then validation fails
	if err == nil {
		t.Fatal("expected error when name does not match directory")
	}
}
