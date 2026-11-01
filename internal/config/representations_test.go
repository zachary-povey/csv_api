package config

import "testing"

func TestCompilePatternValueGroup(t *testing.T) {
	cases := []struct {
		pattern, input, want string
		wholeMatch           bool
	}{
		{"^[0-9]+$", "42", "42", true},
		{"^(?:live)$", "live", "live", true},
		{"(?i)^yes$", "YES", "YES", true},
		{"^£?([0-9]+)$", "£12", "12", false},
		{"^(?:£)?([0-9]+)$", "£12", "12", false},
		{"^[(]([0-9]+)[)]$", "(7)", "7", false},
	}
	for _, c := range cases {
		rgx, whole, err := CompilePattern(c.pattern)
		if err != nil {
			t.Fatalf("%s: %v", c.pattern, err)
		}
		if whole != c.wholeMatch {
			t.Errorf("%s: wholeMatch = %v, want %v", c.pattern, whole, c.wholeMatch)
		}
		m := rgx.FindStringSubmatch(c.input)
		idx := rgx.SubexpIndex("value")
		if m == nil || idx < 0 || m[idx] != c.want {
			t.Errorf("%s on %q: value = %v, want %q", c.pattern, c.input, m, c.want)
		}
	}
}
