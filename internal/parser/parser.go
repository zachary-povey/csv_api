package parser

import (
	"encoding/csv"
	"fmt"
	"maps"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
	"github.com/zachary-povey/csv_api/internal/options"
	"github.com/zachary-povey/csv_api/internal/reader"
)

// DataFailureErrorColumn is the column added to the data failure report
// holding the reasons a row was dropped.
const DataFailureErrorColumn = "__csv_api_error__"

// batchSize is the most rows handed to a worker at once: enough to make the
// hand-over cheap per row, few enough to keep little in flight.
const batchSize = 256

type compiledRepresentation struct {
	regex *regexp.Regexp
	// the named capture groups that give value args, resolved once so
	// matching a cell doesn't look them up again
	groups []captureGroup
	// the value args of the latest match. The map is reused for every match
	// to save allocating one per cell (so a Parser is used by one goroutine):
	// its keys are always the same, the static args are set once and a match
	// only overwrites the capture group values.
	args   map[string]any
	isNull bool
}

type captureGroup struct {
	name  string
	index int
}

func compileRepresentation(rep config.Representation) (compiledRepresentation, error) {
	rgx, _, err := config.CompilePattern(rep.Pattern)
	if err != nil {
		return compiledRepresentation{}, err
	}
	// static args win over capture groups of the same name
	args := maps.Clone(rep.Args)
	if args == nil {
		args = map[string]any{}
	}
	var groups []captureGroup
	for _, name := range rgx.SubexpNames()[1:] {
		// unnamed groups give no args; a repeated name takes the leftmost group
		if _, exists := args[name]; name != "" && !exists {
			groups = append(groups, captureGroup{name, rgx.SubexpIndex(name)})
			args[name] = ""
		}
	}
	return compiledRepresentation{rgx, groups, args, rep.IsNull}, nil
}

// match matches a cell against the representation and, on a match, sets
// rep.args to its value args: the named capture groups merged with the static
// args. A group that didn't take part in the match gives "".
func (rep compiledRepresentation) match(value string) bool {
	loc := rep.regex.FindStringSubmatchIndex(value)
	if loc == nil {
		return false
	}
	for _, group := range rep.groups {
		start, end := loc[2*group.index], loc[2*group.index+1]
		if start < 0 {
			rep.args[group.name] = ""
		} else {
			rep.args[group.name] = value[start:end]
		}
	}
	return true
}

// Parser validates and converts rows from the reader.
type Parser struct {
	config          *config.Config
	opts            options.Options
	representations [][]compiledRepresentation
	// how each field's column is named in errors, in config order
	columnLabels []string
	// the header of each field's column, in config order ("" if missing)
	columnNames []string
	// whether an empty cell is null before representations are tried, per
	// field in config order, for unquoted (,,) and quoted (,"",) cells
	emptyIsNull       []bool
	quotedEmptyIsNull []bool
	// optional destination for rows dropped for invalid data
	dataFailureWriter *csv.Writer

	// the number of goroutines validating and converting rows in parallel
	// (runtime.GOMAXPROCS by default)
	Workers int

	// rows dropped for invalid data, set once ParseData returns
	DroppedRows int
}

// NewParser compiles every representation pattern in the config up front.
// LoadConfig has already checked the patterns, so errors here are unexpected.
// columnNames are the headers of the columns the fields were read from, in
// config order (see reader.ColumnNames); errors name columns by them.
func NewParser(conf *config.Config, opts options.Options, dataFailureWriter *csv.Writer, columnNames []string) (*Parser, error) {
	representations := make([][]compiledRepresentation, len(conf.Fields))
	columnLabels := make([]string, len(conf.Fields))
	emptyIsNull := make([]bool, len(conf.Fields))
	quotedEmptyIsNull := make([]bool, len(conf.Fields))
	for i, field := range conf.Fields {
		columnLabels[i] = columnLabel(field.Name, columnNames[i])
		emptyIsNull[i] = conf.EmptyCellIsNull(field, false)
		quotedEmptyIsNull[i] = conf.EmptyCellIsNull(field, true)
		for _, rep := range field.Representations {
			compiled, err := compileRepresentation(rep)
			if err != nil {
				return nil, fmt.Errorf("error parsing regex for field '%s': %w", field.Name, err)
			}
			representations[i] = append(representations[i], compiled)
		}
	}
	return &Parser{
		config:            conf,
		opts:              opts,
		representations:   representations,
		columnLabels:      columnLabels,
		columnNames:       columnNames,
		emptyIsNull:       emptyIsNull,
		quotedEmptyIsNull: quotedEmptyIsNull,
		dataFailureWriter: dataFailureWriter,
		Workers:           runtime.GOMAXPROCS(0),
	}, nil
}

// worker returns a copy of the parser for one worker goroutine to call
// parseRow on: the compiled regexes are safe to share, but each
// representation's value-args map is reused for every match, so each worker
// needs its own.
func (p *Parser) worker() *Parser {
	w := *p
	w.representations = make([][]compiledRepresentation, len(p.representations))
	for i, reps := range p.representations {
		w.representations[i] = make([]compiledRepresentation, len(reps))
		for j, rep := range reps {
			rep.args = maps.Clone(rep.args)
			w.representations[i][j] = rep
		}
	}
	return &w
}

// columnLabel names a column in errors by its header text, which is what the
// producer sees in the file, adding the field name when header_patterns mapped
// a header that differs from it.
func columnLabel(fieldName string, header string) string {
	if header == "" || header == fieldName {
		return fmt.Sprintf("column '%s'", fieldName)
	}
	return fmt.Sprintf("column '%s' (field '%s')", header, fieldName)
}

// parseValue resolves a single cell against its field's representations and
// converts it to the field's logical type. A nil result with no error is a
// null (only possible on nullable fields, checked at config load). The error
// is the technical reason the value failed.
func (p *Parser) parseValue(fieldIndex int, value string) (any, error) {
	field := p.config.Fields[fieldIndex]

	for _, rep := range p.representations[fieldIndex] {
		if !rep.match(value) {
			continue
		}
		if rep.isNull {
			return nil, nil
		}

		parsed_value, err := Convert(rep.args, field.LogicalTypeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to convert '%s' to %v: %s (resolved args: %v)", value, field.RawLogicalTypeConfig["name"], err, rep.args)
		}
		return parsed_value, nil
	}
	return nil, fmt.Errorf("value '%s' did not match any pattern", value)
}

// parseRow resolves every cell of a row. It checks every cell so a dropped
// row reports all of its problems; the row is only usable without failures.
func (p *Parser) parseRow(row reader.Record) (map[string]any, []error_tracking.CellError) {
	resolvedRow := make(map[string]any, len(row.Values))
	var failures []error_tracking.CellError
	for i, value := range row.Values {
		if value == nil {
			// a missing column (only non-required fields may be
			// missing): the field is null
			resolvedRow[p.config.Fields[i].Name] = nil
			continue
		}
		if *value == "" && p.emptyCellIsNull(i, row) {
			// empty_is_null / quoted_empty_is_null (nullable fields only)
			resolvedRow[p.config.Fields[i].Name] = nil
			continue
		}
		parsed_value, err := p.parseValue(i, *value)
		if err != nil {
			failures = append(failures, error_tracking.CellError{
				Row:     row.Line,
				Column:  p.columnLabels[i],
				Header:  p.columnNames[i],
				Field:   p.config.Fields[i].Name,
				Value:   *value,
				Reason:  err.Error(),
				Message: p.config.Fields[i].ErrorMessage,
			})
			continue
		}
		resolvedRow[p.config.Fields[i].Name] = parsed_value
	}
	return resolvedRow, failures
}

// emptyCellIsNull reports whether the empty cell of field i in row is null
// from the field's empty_is_null or quoted_empty_is_null.
func (p *Parser) emptyCellIsNull(i int, row reader.Record) bool {
	if row.QuotedEmpty != nil && row.QuotedEmpty[i] {
		return p.quotedEmptyIsNull[i]
	}
	return p.emptyIsNull[i]
}

// batch is a run of consecutive rows parsed by one worker. done is closed
// once resolved and failures are filled in.
type batch struct {
	rows     []reader.Record
	resolved []map[string]any
	failures [][]error_tracking.CellError
	done     chan struct{}
}

// ParseData validates and converts the rows from the reader, sending valid
// rows on in input order. Rows are parsed by p.Workers goroutines in batches;
// only the pure per-row work (parseRow) runs in parallel. Everything with side
// effects (error tracking, dropping rows, the data failure report, sending
// rows to the writer) happens here, one row at a time in input order, so the
// result is the same whatever the number of workers.
func (p *Parser) ParseData(input_channel chan reader.Record, output_channel chan map[string]any, wg *sync.WaitGroup, errTracker *error_tracking.ErrorTracker) {
	defer wg.Done()
	defer close(output_channel)

	workers := max(p.Workers, 1)
	jobs := make(chan *batch, workers)
	// batches in input order, a few ahead so workers needn't wait for a slow one
	ordered := make(chan *batch, 2*workers)
	go dispatchBatches(input_channel, jobs, ordered)
	for range workers {
		go func(w *Parser) {
			for b := range jobs {
				for i, row := range b.rows {
					b.resolved[i], b.failures[i] = w.parseRow(row)
				}
				close(b.done)
			}
		}(p.worker())
	}

	// workers never wait on this loop, so it always gets through every batch
	// and dispatchBatches always drains the input: the reader never blocks
	for b := range ordered {
		<-b.done
		for i, row := range b.rows {
			p.handleRow(row, b.resolved[i], b.failures[i], output_channel, errTracker)
		}
	}
}

// dispatchBatches groups the input rows into batches, sending each to the
// workers and, in input order, to ParseData. A batch is sent as soon as no
// further row is ready, so rows aren't held back waiting for a full batch
// (with --stream they reach the output as they are validated). It always
// drains the input until the reader closes it.
func dispatchBatches(input_channel chan reader.Record, jobs chan<- *batch, ordered chan<- *batch) {
	defer close(jobs)
	defer close(ordered)
	for row := range input_channel {
		rows := make([]reader.Record, 1, batchSize)
		rows[0] = row
	fill:
		for len(rows) < batchSize {
			select {
			case row, ok := <-input_channel:
				if !ok {
					break fill
				}
				rows = append(rows, row)
			default:
				break fill
			}
		}
		b := &batch{
			rows:     rows,
			resolved: make([]map[string]any, len(rows)),
			failures: make([][]error_tracking.CellError, len(rows)),
			done:     make(chan struct{}),
		}
		ordered <- b
		jobs <- b
	}
}

// handleRow acts on one parsed row: a valid row goes to the writer, one with
// failures is reported or dropped.
func (p *Parser) handleRow(row reader.Record, resolvedRow map[string]any, failures []error_tracking.CellError, output_channel chan map[string]any, errTracker *error_tracking.ErrorTracker) {
	if len(failures) > 0 {
		if p.opts.ErrorOnDataFailures {
			for _, failure := range failures {
				errTracker.AddCellError(failure)
			}
		} else {
			p.DroppedRows++
			if p.dataFailureWriter != nil {
				texts := make([]string, len(failures))
				for j, failure := range failures {
					texts[j] = failure.Text()
				}
				reportRow := append(append([]string{}, row.Raw...), strings.Join(texts, "; "))
				if err := p.dataFailureWriter.Write(reportRow); err != nil {
					errTracker.AddExecutionError(fmt.Errorf("error writing data failure report: %w", err))
				}
			}
		}
		return
	}

	select {
	case <-errTracker.KillCh:
		// keep draining input until the reader closes it
	case output_channel <- resolvedRow:
	}
}
