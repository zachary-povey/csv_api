package config

import (
	"strings"
	"testing"
)

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

// TASK-19 AC#1, AC#2, AC#3: decimal values are checked against precision and
// scale on their exact value.
func TestDecimalArgPrecisionScale(t *testing.T) {
	fits := map[string]string{
		"1.23": "123/100", "1.5": "3/2", "-99.99": "-9999/100", "0001.50": "3/2",
		"1.230": "123/100", "00.05": "1/20", "0": "0", "-0.01": "-1/100", "99": "99",
	}
	for in, want := range fits {
		r, err := DecimalArg(in, 4, 2)
		if err != nil {
			t.Errorf("%s: unexpected error %v", in, err)
		} else if r.String() != want && r.RatString() != want {
			t.Errorf("%s: got %s, want %s", in, r.RatString(), want)
		}
	}
	fails := map[string]string{
		"1.234":    "more than 2 fractional digits (scale 2)",
		"-0.001":   "more than 2 fractional digits (scale 2)",
		"123456.7": "more than 2 integer digits (precision 4, scale 2)",
		"100":      "more than 2 integer digits (precision 4, scale 2)",
		"-100.00":  "more than 2 integer digits (precision 4, scale 2)",
		"abc":      "is not a decimal",
		"1/3":      "is not a decimal",
	}
	for in, want := range fails {
		_, err := DecimalArg(in, 4, 2)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error %v, want containing %q", in, err, want)
		}
	}
}
