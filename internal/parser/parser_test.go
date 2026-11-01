package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/options"
	"github.com/zachary-povey/csv_api/internal/reader"
)

// benchmarkConfig is the config scripts/benchmark.py uses: one field of each
// common logical type plus a nullable string.
const benchmarkConfig = `name: com.example.benchmark
fields:
  - name: id
    logical_type:
      name: integer
    representations:
      - pattern: "^[0-9]+$"
  - name: name
    logical_type:
      name: string
    representations:
      - pattern: "^(.+)$"
  - name: amount
    logical_type:
      name: decimal
      args:
        precision: 12
        scale: 2
    representations:
      - pattern: "^(-?[0-9]+\\.[0-9]{2})$"
  - name: ratio
    logical_type:
      name: decimal
      args:
        as_float: true
    representations:
      - pattern: "^([0-9]+\\.[0-9]+)$"
  - name: status
    logical_type:
      name: enum
      args:
        permitted_values: [open, closed, pending]
    representations:
      - pattern: "^(open|closed|pending)$"
  - name: created_on
    logical_type:
      name: date
    representations:
      - pattern: "^(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})$"
  - name: updated_at
    logical_type:
      name: timestamp
    representations:
      - pattern: "^(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})T(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})Z$"
  - name: active
    logical_type:
      name: boolean
    representations:
      - pattern: "^(true|false)$"
  - name: note
    nullable: true
    logical_type:
      name: string
    representations:
      - pattern: "^$"
        is_null: true
      - pattern: "^(.+)$"
`

// a row as scripts/benchmark.py generates it
var benchmarkRow = []string{"1", "customer 1", "-463.01", "0.0001", "closed", "2001-02-02", "2024-02-02T01:01:07Z", "true", "note 1"}

func newBenchmarkParser(t testing.TB) (*Parser, reader.Record) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(benchmarkConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	conf, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	columnNames := make([]string, len(conf.Fields))
	for i, field := range conf.Fields {
		columnNames[i] = field.Name
	}
	p, err := NewParser(conf, options.Options{}, nil, columnNames)
	if err != nil {
		t.Fatal(err)
	}
	values := make([]*string, len(benchmarkRow))
	for i := range benchmarkRow {
		values[i] = &benchmarkRow[i]
	}
	return p, reader.Record{Line: 2, Raw: benchmarkRow, Values: values}
}

// maxAllocsPerRow is the TASK-20 budget for parsing one benchmark row: before
// TASK-20 the parser made 95 allocations per row, after it 40 (with Go 1.24),
// leaving a little headroom for other Go versions.
const maxAllocsPerRow = 45

// TASK-20 AC#3: a representative row of the benchmark's field types parses
// with at most maxAllocsPerRow allocations.
func TestParseRowAllocations(t *testing.T) {
	p, row := newBenchmarkParser(t)
	if _, failures := p.parseRow(row); len(failures) > 0 {
		t.Fatalf("unexpected failures: %v", failures)
	}
	allocs := testing.AllocsPerRun(1000, func() { p.parseRow(row) })
	t.Logf("%.0f allocations per row", allocs)
	if allocs > maxAllocsPerRow {
		t.Errorf("parsing a row made %.0f allocations, want at most %d", allocs, maxAllocsPerRow)
	}
}

// TASK-20 AC#1: parser throughput on a benchmark row (go test -bench
// ParseRow ./internal/parser). The end-to-end figure for AC#1 comes from
// scripts/benchmark.py and is recorded in README "Performance".
func BenchmarkParseRow(b *testing.B) {
	p, row := newBenchmarkParser(b)
	b.ReportAllocs()
	for b.Loop() {
		p.parseRow(row)
	}
}

// TASK-20 AC#2: matching resolves the same value args as before the reusable
// args map: static args win over groups of the same name, a group that didn't
// take part in the match gives "", and nothing is left over from an earlier
// match.
func TestRepresentationMatchArgs(t *testing.T) {
	rep, err := compileRepresentation(config.Representation{
		Pattern: `^(?P<year>[0-9]{4})-(?P<month>[0-9]{2})(?:-(?P<day>[0-9]{2}))?$`,
		Args:    map[string]any{"month": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		value string
		want  map[string]any
	}{
		{"2024-06-15", map[string]any{"year": "2024", "month": 1, "day": "15"}},
		{"2023-07", map[string]any{"year": "2023", "month": 1, "day": ""}},
	}
	for _, c := range cases {
		if !rep.match(c.value) {
			t.Fatalf("%q: no match", c.value)
		}
		if fmt.Sprint(rep.args) != fmt.Sprint(c.want) {
			t.Errorf("%q: args %v, want %v", c.value, rep.args, c.want)
		}
	}
	if rep.match("June") {
		t.Errorf("'June' matched")
	}
}
