package parser

import "testing"

// TASK-2 AC#1, AC#2: convert_int accepts numeric static args and capture group strings.
func TestConvertIntAcceptsNumbersAndStrings(t *testing.T) {
	cases := []struct {
		value any
		want  int
	}{
		{"42", 42},
		{"-7", -7},
		{0, 0},
		{int64(5), 5},
		{uint64(6), 6},
		{12.0, 12},
	}
	for _, c := range cases {
		got, err := convert_int(map[string]any{"value": c.value})
		if err != nil {
			t.Errorf("convert_int(%#v) error: %v", c.value, err)
		} else if got != c.want {
			t.Errorf("convert_int(%#v) = %d, want %d", c.value, got, c.want)
		}
	}
}

// TASK-2: non-integer static args are still rejected.
func TestConvertIntRejectsNonIntegers(t *testing.T) {
	for _, value := range []any{1.5, "abc", true} {
		if _, err := convert_int(map[string]any{"value": value}); err == nil {
			t.Errorf("convert_int(%#v) expected error", value)
		}
	}
}
