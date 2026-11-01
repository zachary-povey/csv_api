package config

import (
	"fmt"
	"math/big"
	"strconv"
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

// decimalArgBefore is DecimalArg as it was before TASK-20 (always the exact
// path through big.Rat), kept to check the cheaper path against.
func decimalArgBefore(s string, precision, scale int) (*big.Rat, error) {
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return nil, fmt.Errorf("'%s' is not a decimal", s)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("'%s' is not a decimal", s)
	}
	unscaled := new(big.Rat).Mul(r, new(big.Rat).SetInt(pow10(scale)))
	if !unscaled.IsInt() {
		return nil, fmt.Errorf("'%s' has more than %d fractional digits (scale %d)", s, scale, scale)
	}
	if new(big.Int).Abs(unscaled.Num()).Cmp(pow10(precision)) >= 0 {
		return nil, fmt.Errorf("'%s' has more than %d integer digits (precision %d, scale %d)",
			s, precision-scale, precision, scale)
	}
	return r, nil
}

// TASK-20 AC#2: DecimalArg's cheaper path for plain decimals gives the same
// values and error messages as before.
func TestDecimalArgUnchanged(t *testing.T) {
	values := []string{
		"0", "-0", "+0", "0.0", ".5", "-.5", "5.", "1.230", "1.235", "-463.01", "012.340",
		"99.99", "100.00", "-99.999", "0.001", "0.0010", "000.000", "123456789012.5",
		"9999999999.99", "10000000000.00", "-0.01", "+7.70", "123456789.123456789",
		"999999999999999999", "-0.000000000000000001", "12345678901234567890.5",
		"1e3", "1.5E-2", "0x1p-2", "Inf", "NaN", "abc", "", ".", "-", "1.2.3", "1e999",
	}
	sizes := [][2]int{{12, 2}, {4, 2}, {2, 2}, {5, 0}, {1, 0}, {38, 10}, {2, 3}, {40, 20}}
	for _, v := range values {
		for _, size := range sizes {
			got, gotErr := DecimalArg(v, size[0], size[1])
			want, wantErr := decimalArgBefore(v, size[0], size[1])
			if fmt.Sprint(gotErr) != fmt.Sprint(wantErr) {
				t.Errorf("%q decimal(%d, %d): error %v, want %v", v, size[0], size[1], gotErr, wantErr)
			} else if want != nil && got.Cmp(want) != 0 {
				t.Errorf("%q decimal(%d, %d): value %v, want %v", v, size[0], size[1], got, want)
			}
		}
	}
}
