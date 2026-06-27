package artifact

import (
	"fmt"
	"regexp"
)

// semverPattern matches a SemVer 2.0.0 string: MAJOR.MINOR.PATCH (no leading
// zeros) with optional pre-release and build metadata.
var semverPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z][0-9A-Za-z.-]*)?(\+[0-9A-Za-z][0-9A-Za-z.-]*)?$`)

// ValidateVersion checks that v is a valid SemVer string (e.g. "1.3.0" or
// "1.0.0-rc.1+build.5"). An empty string is rejected; callers that permit an
// "unversioned" artifact check for empty before calling. harness validates
// versions but does not yet order them — range resolution arrives with a
// registry.
func ValidateVersion(v string) error {
	if !semverPattern.MatchString(v) {
		return fmt.Errorf("version %q must be SemVer (MAJOR.MINOR.PATCH)", v)
	}
	return nil
}
