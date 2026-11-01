package parser

import (
	"encoding/csv"
	"fmt"
	"maps"
	"regexp"
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
	// optional destination for rows dropped for invalid data
	dataFailureWriter *csv.Writer

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
	for i, field := range conf.Fields {
		columnLabels[i] = columnLabel(field.Name, columnNames[i])
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
		dataFailureWriter: dataFailureWriter,
	}, nil
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

func (p *Parser) ParseData(input_channel chan reader.Record, output_channel chan map[string]any, wg *sync.WaitGroup, errTracker *error_tracking.ErrorTracker) {
	defer wg.Done()
	defer close(output_channel)

	// always drain the input channel so the reader never blocks; the reader
	// stops and closes it when processing is killed
	for row := range input_channel {
		resolvedRow, failures := p.parseRow(row)

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
			continue
		}

		select {
		case <-errTracker.KillCh:
			// keep draining input until the reader closes it
		case output_channel <- resolvedRow:
		}
	}
}
