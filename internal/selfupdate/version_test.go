package selfupdate

import "testing"

func TestNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"0.1.0", "0.2.0", true},
		{"v0.1.0", "v0.1.1", true},
		{"0.1.0", "v0.1.0", false},    // equal (v prefix ignored)
		{"0.2.0", "0.1.9", false},     // older
		{"1.0.0-rc.1", "1.0.0", true}, // release beats pre-release
		{"1.0.0", "1.0.0-rc.1", false},
		{"1.2.3", "1.2.4", true},
		{"1.2.3", "2.0.0", true},
		{"dev", "v0.2.0", false}, // dev build is never nagged
		{"abc123-dirty", "v9.9.9", false},
		{"0.1.0", "garbage", false},
	}
	for _, c := range cases {
		if got := Newer(c.current, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestIsRelease(t *testing.T) {
	for _, v := range []string{"0.1.0", "v1.2.3", "1.0.0-rc.1"} {
		if !isRelease(v) {
			t.Errorf("isRelease(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"dev", "", "abc-dirty", "1.2"} {
		if isRelease(v) {
			t.Errorf("isRelease(%q) = true, want false", v)
		}
	}
}
