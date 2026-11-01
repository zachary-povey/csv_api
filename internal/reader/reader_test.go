package reader

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/zachary-povey/csv_api/internal/config"
)

func loadConfig(t *testing.T, yaml string) *config.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	conf, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	return conf
}

const testConfig = `
name: test
fields:
  - name: id
    logical_type: {name: integer}
    representations: [{pattern: "^[0-9]+$"}]
  - name: first_name
    header_patterns: ["(?i)^first ?name$", "^given$"]
    logical_type: {name: string}
    representations: [{pattern: "^(.+)$"}]
  - name: notes
    required: false
    nullable: true
    header_patterns: ["^notes$"]
    logical_type: {name: string}
    representations: [{pattern: "^(.*)$"}]
`

func positions(p []*int) []int {
	out := []int{}
	for _, pos := range p {
		if pos == nil {
			out = append(out, -1)
		} else {
			out = append(out, *pos)
		}
	}
	return out
}

// TASK-5 AC#1, AC#2: header_patterns map columns to fields; fields without
// them use the exact name; header_patterns replace the name match.
func TestGetFieldPositions(t *testing.T) {
	conf := loadConfig(t, testConfig)
	cases := []struct {
		header []string
		want   []int
	}{
		{[]string{"id", "First Name"}, []int{0, 1, -1}},
		{[]string{"x", "given", "notes", "id"}, []int{3, 1, 2}},
		{[]string{"FIRSTNAME", "ID", "id"}, []int{2, 0, -1}},
	}
	for _, c := range cases {
		got, errs := getFieldPositions(conf, c.header)
		if len(errs) > 0 {
			t.Errorf("%v: unexpected errors: %v", c.header, errs)
			continue
		}
		if g := positions(got); !slices.Equal(g, c.want) {
			t.Errorf("%v: got positions %v, want %v", c.header, g, c.want)
		}
	}
}

// TASK-5 AC#1, AC#3: unmatched required fields, and ambiguous matches in
// either direction, are file-level errors naming the headers and fields.
func TestGetFieldPositionsErrors(t *testing.T) {
	conf := loadConfig(t, testConfig)
	cases := []struct {
		header []string
		want   []string
	}{
		{[]string{"id", "first_name"}, []string{
			"input data is missing some required fields: [first_name (no header matches header_patterns '(?i)^first ?name$', '^given$')]"}},
		{[]string{"id", "given", "First Name"}, []string{
			"field 'first_name' matches more than one header: 'given' (column 2), 'First Name' (column 3)"}},
		{[]string{"id", "given", "id"}, []string{
			"field 'id' matches more than one header: 'id' (column 1), 'id' (column 3)"}},
	}
	for _, c := range cases {
		_, errs := getFieldPositions(conf, c.header)
		got := []string{}
		for _, err := range errs {
			got = append(got, err.Error())
		}
		if strings.Join(got, "\n") != strings.Join(c.want, "\n") {
			t.Errorf("%v:\n got %q\nwant %q", c.header, got, c.want)
		}
	}

	ambiguous := loadConfig(t, `
name: test
fields:
  - name: a
    header_patterns: ["x"]
    logical_type: {name: string}
    representations: [{pattern: "^(.*)$"}]
  - name: b
    header_patterns: ["y"]
    logical_type: {name: string}
    representations: [{pattern: "^(.*)$"}]
`)
	_, errs := getFieldPositions(ambiguous, []string{"xy"})
	want := []string{
		"header 'xy' (column 1) matches more than one field: 'a', 'b'",
	}
	if len(errs) != 1 || errs[0].Error() != want[0] {
		t.Errorf("got %v, want %v", errs, want)
	}
}
