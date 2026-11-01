package reader

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
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

// quoteConfig makes every field quote sensitive: an unquoted empty cell is
// null, a quoted one is not.
const quoteConfig = `
name: test
quoted_empty_is_null: false
fields:
  - name: a
    nullable: true
    logical_type: {name: string}
    representations: [{pattern: "^(.*)$"}]
  - name: b
    nullable: true
    logical_type: {name: string}
    representations: [{pattern: "^(.*)$"}]
  - name: c
    nullable: true
    logical_type: {name: string}
    representations: [{pattern: "^(.*)$"}]
`

// readMarks reads data with conf and describes each row's values: "q" for a
// quoted empty cell, "u" for an unquoted one and "v" for a value.
func readMarks(t *testing.T, conf *config.Config, data string) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.csv")
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	r, execErr, fileErrs := Open(path, conf)
	if execErr != nil || len(fileErrs) > 0 {
		t.Fatalf("open: %v %v", execErr, fileErrs)
	}
	channel := make(chan Record, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	errTracker := error_tracking.NewErrorTracker(false, 0)
	go r.ReadRows(channel, &wg, errTracker)
	marks := []string{}
	for record := range channel {
		mark := ""
		for i, value := range record.Values {
			switch {
			case *value != "":
				mark += "v"
			case record.QuotedEmpty != nil && record.QuotedEmpty[i]:
				mark += "q"
			default:
				mark += "u"
			}
		}
		marks = append(marks, mark)
	}
	wg.Wait()
	if errTracker.ErrorReport.ContainsErrors() || len(errTracker.ExecutionErrors) > 0 {
		t.Fatalf("unexpected errors reading %q", data)
	}
	return marks
}

// TASK-23 AC#10: quoted (,"",) and unquoted (,,) empty cells are told apart
// in any position, including the last field of a row, with CRLF line endings,
// with no trailing newline (or a lone "\r") at end of file, after a quoted
// field containing a newline, and after skipped blank lines.
func TestQuotedEmptyDetection(t *testing.T) {
	conf := loadConfig(t, quoteConfig)
	cases := []struct {
		name string
		data string
		want []string
	}{
		{"LF", "a,b,c\n,\"\",\n\"\",\"\",\"\"\n,,\n", []string{"uqu", "qqq", "uuu"}},
		{"CRLF", "a,b,c\r\n,\"\",\r\n\"\",\"\",\"\"\r\n,,\r\n", []string{"uqu", "qqq", "uuu"}},
		{"no trailing newline, quoted last", "a,b,c\n,,\"\"", []string{"uuq"}},
		{"no trailing newline, unquoted last", "a,b,c\n\"\",\"\",", []string{"qqu"}},
		{"trailing CR at EOF, quoted last", "a,b,c\n,,\"\"\r", []string{"uuq"}},
		{"trailing CR at EOF, unquoted last", "a,b,c\n\"\",\"\",\r", []string{"qqu"}},
		{"after a quoted field with a newline", "a,b,c\n\"x\ny\",,\"\"\n\"x\ny\",\"\",\n,\"x\n\ny\",\"\"\n",
			[]string{"vuq", "vqu", "uvq"}},
		{"after a quoted field with a CRLF, CRLF endings", "a,b,c\r\n\"x\r\ny\",,\"\"\r\n\"x\r\ny\",\"\",\r\n",
			[]string{"vuq", "vqu"}},
		{"after a quoted field with a newline, no trailing newline", "a,b,c\n,\"x\ny\",\"\"", []string{"uvq"}},
		{"blank lines skipped", "a,b,c\n\n\n,\"\",\n\r\n\"\",,\"\"\n", []string{"uqu", "quq"}},
		{"values ending in quotes", "a,b,c\n\"\"\"\",,\"a\"\"\"\n\"\"\"\",\"\",\n", []string{"vuv", "vqu"}},
	}
	for _, c := range cases {
		got := readMarks(t, conf, c.data)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TASK-23 AC#10: detection keeps working past the tail buffer, with long
// values (longer than the buffer) and many rows.
func TestQuotedEmptyDetectionLongInput(t *testing.T) {
	conf := loadConfig(t, quoteConfig)
	long := strings.Repeat("x", tailBufferSize+100)
	var data strings.Builder
	data.WriteString("a,b,c\n")
	want := []string{}
	// a few rows longer than the buffer among many short ones
	for i := range 4000 {
		kind := i % 2
		if i%500 < 2 {
			kind = 2 + i%2
		}
		switch kind {
		case 0:
			data.WriteString("abc,,\"\"\n")
			want = append(want, "vuq")
		case 1:
			data.WriteString("\"x\ny\",\"\",\r\n")
			want = append(want, "vqu")
		case 2:
			data.WriteString(long + ",,\"\"\n")
			want = append(want, "vuq")
		case 3:
			data.WriteString("\"" + long + "\",\"\",\n")
			want = append(want, "vqu")
		}
	}
	got := readMarks(t, conf, data.String())
	if !slices.Equal(got, want) {
		t.Errorf("got %d rows, want %d (or marks differ)", len(got), len(want))
	}
}

// TASK-23: with the default flags (or any field's flags equal) quoting can't
// matter, so empty cells aren't classified and reads aren't tracked.
func TestQuotedEmptyNotTrackedByDefault(t *testing.T) {
	conf := loadConfig(t, strings.Replace(quoteConfig, "quoted_empty_is_null: false", "", 1))
	got := readMarks(t, conf, "a,b,c\n,\"\",\"\"\n")
	if !slices.Equal(got, []string{"uuu"}) {
		t.Errorf("got %v, want [uuu]", got)
	}
	path := filepath.Join(t.TempDir(), "data.csv")
	os.WriteFile(path, []byte("a,b,c\n"), 0644)
	r, _, _ := Open(path, conf)
	defer r.file.Close()
	if r.tail != nil {
		t.Errorf("reads are tracked although no field is quote sensitive")
	}
}
