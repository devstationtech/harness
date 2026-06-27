package artifact_test

import (
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
)

func TestFrontmatterAbstractIsValid(t *testing.T) {
	// @Given a frontmatter declaring contracts (an abstract skill)
	front := artifact.Frontmatter{Name: "low-level-design", Description: "d", Contracts: []string{"domain", "command"}}

	// @When validated
	// @Then it is accepted
	if err := front.Validate("low-level-design"); err != nil {
		t.Errorf("expected an abstract skill to be valid: %v", err)
	}
}

func TestFrontmatterCapabilityIsValid(t *testing.T) {
	// @Given a frontmatter implementing an abstract (a capability)
	front := artifact.Frontmatter{Name: "lld-typescript", Description: "d", Implements: "low-level-design", Provides: []string{"domain"}, Stack: "typescript"}

	// @When validated
	// @Then it is accepted
	if err := front.Validate("lld-typescript"); err != nil {
		t.Errorf("expected a capability to be valid: %v", err)
	}
}

func TestFrontmatterRejectsAbstractAndCapabilityTogether(t *testing.T) {
	// @Given a frontmatter declaring both contracts and implements
	front := artifact.Frontmatter{Name: "confused", Description: "d", Contracts: []string{"domain"}, Implements: "low-level-design"}

	// @When validated
	// @Then it is rejected
	if err := front.Validate("confused"); err == nil {
		t.Error("expected an artifact that is both abstract and capability to be invalid")
	}
}
