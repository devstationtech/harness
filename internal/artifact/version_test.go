package artifact_test

import (
	"testing"

	"github.com/devstationtech/harness/internal/artifact"
)

func TestValidateVersion(t *testing.T) {
	// @Given representative valid and invalid version strings
	valid := []string{"1.3.0", "0.0.1", "10.20.30", "1.0.0-rc.1", "1.0.0-rc.1+build.5"}
	invalid := []string{"", "1.0", "v1.0.0", "1", "1.2.3.4", "abc", "01.2.3"}

	// @When each is validated
	// @Then SemVer strings pass and the rest are rejected
	for _, v := range valid {
		if err := artifact.ValidateVersion(v); err != nil {
			t.Errorf("expected %q to be valid: %v", v, err)
		}
	}
	for _, v := range invalid {
		if err := artifact.ValidateVersion(v); err == nil {
			t.Errorf("expected %q to be invalid", v)
		}
	}
}
