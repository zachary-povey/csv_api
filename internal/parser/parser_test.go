package parser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
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
	p, err := NewParser(loadBenchmarkConfig(t), options.Options{}, nil, benchmarkColumns)
	if err != nil {
		t.Fatal(err)
	}
	return p, record(2, benchmarkRow)
}

// the header of the benchmark csv: the field names, in config order
var benchmarkColumns = []string{"id", "name", "amount", "ratio", "status", "created_on", "updated_at", "active", "note"}

func record(line int, raw []string) reader.Record {
	values := make([]*string, len(raw))
	for i := range raw {
		values[i] = &raw[i]
	}
	return reader.Record{Line: line, Raw: raw, Values: values}
}

func loadBenchmarkConfig(t testing.TB) *config.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(benchmarkConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	conf, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	return conf
}

// maxAllocsPerRow is the TASK-20 budget for parsing one benchmark row: before
// TASK-20 the parser made 95 allocations per row, after it 40 (with Go 1.24),
// leaving a little headroom for other Go versions.
const maxAllocsPerRow = 45

// TASK-20 AC#3: a representative row of the benchmark's field types parses
// with at most maxAllocsPerRow allocations.
func TestParseRowAllocations(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector adds allocations")
	}
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

// interleavedRows are benchmark rows, some with one or two invalid values,
// spread so that failures fall in every batch and across batch boundaries.
func interleavedRows(n int) []reader.Record {
	rows := make([]reader.Record, n)
	for i := range n {
		raw := []string{fmt.Sprint(i), fmt.Sprintf("customer %d", i), fmt.Sprintf("%d.%02d", i, i%100), "0.5", "open", "2024-01-02", "2024-01-02T03:04:05Z", "true", ""}
		if i%7 == 3 {
			raw[0] = fmt.Sprintf("x%d", i)
		}
		if i%11 == 5 {
			raw[4] = "unknown"
			raw[5] = "2024-13-45"
		}
		rows[i] = record(i+2, raw)
	}
	return rows
}

type parseResult struct {
	output  []string
	report  string
	errors  []error_tracking.CellError
	dropped int
	stop    string
}

// parseWithWorkers runs ParseData over rows with the given number of
// workers, feeding the input the way the reader does (stopping on KillCh).
func parseWithWorkers(t *testing.T, conf *config.Config, rows []reader.Record, workers int, opts options.Options, failFast bool, maxErrors int) parseResult {
	t.Helper()
	var report bytes.Buffer
	reportWriter := csv.NewWriter(&report)
	p, err := NewParser(conf, opts, reportWriter, benchmarkColumns)
	if err != nil {
		t.Fatal(err)
	}
	p.Workers = workers
	tracker := error_tracking.NewErrorTracker(failFast, maxErrors)
	input := make(chan reader.Record, 100)
	output := make(chan map[string]any, 100)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer close(input)
		for _, row := range rows {
			select {
			case <-tracker.KillCh:
				return
			case input <- row:
			}
		}
	}()
	go p.ParseData(input, output, &wg, tracker)
	var result parseResult
	for row := range output {
		result.output = append(result.output, fmt.Sprint(row))
	}
	wg.Wait()
	reportWriter.Flush()
	result.report = report.String()
	result.errors = tracker.ErrorReport.CellErrors
	result.dropped = p.DroppedRows
	result.stop = tracker.ErrorReport.StopReason
	return result
}

// TASK-22 AC#3: with several workers, rows with data failures interleaved
// with valid rows give exactly the single-worker result: output rows in input
// order, the same errors in the same order with the same row numbers, the
// same data failure report and dropped-row count, and --max-errors/fail-fast
// stopping at the same errors.
func TestWorkersGiveSingleWorkerResults(t *testing.T) {
	conf := loadBenchmarkConfig(t)
	rows := interleavedRows(5000)
	cases := []struct {
		name      string
		opts      options.Options
		failFast  bool
		maxErrors int
		// a stopped run may send a few more rows before noticing KillCh (as
		// with one parser goroutine), so its output isn't compared
		stops bool
	}{
		{name: "dropped rows with a data failure report", opts: options.Options{ErrorOnDataFailures: false}},
		{name: "all errors collected", opts: options.Options{ErrorOnDataFailures: true}},
		{name: "max errors", opts: options.Options{ErrorOnDataFailures: true}, maxErrors: 300, stops: true},
		{name: "fail fast", opts: options.Options{ErrorOnDataFailures: true}, failFast: true, stops: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := parseWithWorkers(t, conf, rows, 1, c.opts, c.failFast, c.maxErrors)
			if !c.stops && len(want.output)+want.dropped+len(want.errors) == 0 {
				t.Fatal("single-worker run gave no results")
			}
			for _, workers := range []int{2, 4, 8} {
				got := parseWithWorkers(t, conf, rows, workers, c.opts, c.failFast, c.maxErrors)
				if fmt.Sprint(got.errors) != fmt.Sprint(want.errors) {
					t.Errorf("%d workers: errors differ:\n got %v\nwant %v", workers, got.errors, want.errors)
				}
				if got.stop != want.stop {
					t.Errorf("%d workers: stop reason %q, want %q", workers, got.stop, want.stop)
				}
				if c.stops {
					continue
				}
				if strings.Join(got.output, "\n") != strings.Join(want.output, "\n") {
					t.Errorf("%d workers: output rows differ from one worker's", workers)
				}
				if got.report != want.report || got.dropped != want.dropped {
					t.Errorf("%d workers: data failure report (%d dropped) differs from one worker's (%d dropped)", workers, got.dropped, want.dropped)
				}
			}
		})
	}
	// the cases above really have failures in every batch
	all := parseWithWorkers(t, conf, rows, 4, options.Options{ErrorOnDataFailures: true}, false, 0)
	if len(all.errors) < 1000 || len(all.output) < 3000 {
		t.Fatalf("expected many errors and valid rows, got %d and %d", len(all.errors), len(all.output))
	}
}

// TASK-22 AC#2: when the pipeline is killed, ParseData still drains its input
// and returns even if nothing reads its output any more.
func TestWorkersDrainInputWhenKilled(t *testing.T) {
	conf := loadBenchmarkConfig(t)
	rows := interleavedRows(20000)
	p, err := NewParser(conf, options.Options{ErrorOnDataFailures: true}, nil, benchmarkColumns)
	if err != nil {
		t.Fatal(err)
	}
	p.Workers = 4
	tracker := error_tracking.NewErrorTracker(true, 0)
	input := make(chan reader.Record, 100)
	output := make(chan map[string]any) // never read
	var wg sync.WaitGroup
	wg.Add(1)
	go p.ParseData(input, output, &wg, tracker)
	go func() {
		// the first row fails, stopping the run; the reader stops on KillCh
		defer close(input)
		for _, row := range rows[3:] {
			select {
			case <-tracker.KillCh:
				return
			case input <- row:
			}
		}
	}()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("ParseData did not return after the pipeline was killed")
	}
	if len(tracker.ErrorReport.CellErrors) != 1 || tracker.ErrorReport.CellErrors[0].Row != 5 {
		t.Errorf("errors %v, want only row 5's", tracker.ErrorReport.CellErrors)
	}
}
