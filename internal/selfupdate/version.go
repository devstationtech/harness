package selfupdate

import (
	"strconv"
	"strings"
)

// semver is the core MAJOR.MINOR.PATCH of a version plus any pre-release tag.
// Build metadata is ignored (it does not affect precedence).
type semver struct {
	major, minor, patch int
	pre                 string
}

// parseSemver parses "v1.2.3", "1.2.3-rc.1" and the like. It reports ok=false
// for anything that is not a clean release version (e.g. "dev" or a git-describe
// string), which callers use to leave development builds alone.
func parseSemver(v string) (semver, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if v == "" {
		return semver{}, false
	}
	core := v
	pre := ""
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		core = v[:i]
		if v[i] == '-' {
			pre = v[i+1:]
			if j := strings.IndexByte(pre, '+'); j >= 0 {
				pre = pre[:j] // drop build metadata
			}
		}
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return semver{}, false
		}
		nums[i] = n
	}
	return semver{nums[0], nums[1], nums[2], pre}, true
}

// compare orders two versions: -1 if a<b, 0 if equal, +1 if a>b. A release
// outranks a pre-release of the same core (1.0.0 > 1.0.0-rc.1).
func compare(a, b semver) int {
	for _, d := range []int{a.major - b.major, a.minor - b.minor, a.patch - b.patch} {
		if d > 0 {
			return 1
		}
		if d < 0 {
			return -1
		}
	}
	switch {
	case a.pre == "" && b.pre == "":
		return 0
	case a.pre == "":
		return 1
	case b.pre == "":
		return -1
	case a.pre > b.pre:
		return 1
	case a.pre < b.pre:
		return -1
	default:
		return 0
	}
}

// Newer reports whether latest is a strictly greater release than current. Both
// may carry a leading "v". A current version that is not a clean release version
// is treated as not-upgradable, so development builds are never nagged.
func Newer(current, latest string) bool {
	c, ok := parseSemver(current)
	if !ok {
		return false
	}
	l, ok := parseSemver(latest)
	if !ok {
		return false
	}
	return compare(l, c) > 0
}

// isRelease reports whether v is a clean release version (vs a dev build).
func isRelease(v string) bool {
	_, ok := parseSemver(v)
	return ok
}
