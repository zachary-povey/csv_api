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

func extract_args(rgx *regexp.Regexp, value string) (bool, map[string]string) {
	args := make(map[string]string)

	group_names := rgx.SubexpNames()
	if len(group_names) == 1 {
		group_names = make([]string, 0)
	} else {
		group_names = group_names[1:]
	}
	groups := rgx.FindStringSubmatch(value)
	if groups == nil {
		return false, args
	}

	for _, group_name := range group_names {
		group_index := rgx.SubexpIndex(group_name)
		args[group_name] = groups[group_index]
	}
	return true, args
}

type compiledRepresentation struct {
	regex *regexp.Regexp
	args  map[string]any
}

// Parser validates and converts rows from the reader.
type Parser struct {
	config          *config.Config
	opts            options.Options
	representations [][]compiledRepresentation
	// optional destination for rows dropped for invalid data
	dataFailureWriter *csv.Writer

	// rows dropped for invalid data, set once ParseData returns
	DroppedRows int
}

// NewParser compiles every representation pattern in the config up front.
// LoadConfig has already checked the patterns, so errors here are unexpected.
func NewParser(conf *config.Config, opts options.Options, dataFailureWriter *csv.Writer) (*Parser, error) {
	representations := make([][]compiledRepresentation, len(conf.Fields))
	for i, field := range conf.Fields {
		for _, rep := range field.Representations {
			rgx, _, err := config.CompilePattern(rep.Pattern)
			if err != nil {
				return nil, fmt.Errorf("error parsing regex for field '%s': %w", field.Name, err)
			}
			representations[i] = append(representations[i], compiledRepresentation{rgx, rep.Args})
		}
	}
	return &Parser{
		config:            conf,
		opts:              opts,
		representations:   representations,
		dataFailureWriter: dataFailureWriter,
	}, nil
}

// parseValue resolves a single cell against its field's representations and
// converts it to the field's logical type.
func (p *Parser) parseValue(fieldIndex int, value string) (any, error) {
	field := p.config.Fields[fieldIndex]

	for _, rep := range p.representations[fieldIndex] {
		matched, args := extract_args(rep.regex, value)
		if !matched {
			continue
		}

		// merge args from regex with static ones on representation
		// values from config match have priority if there is overlap
		field_args := make(map[string]any)
		for k, v := range args {
			field_args[k] = v
		}
		maps.Copy(field_args, rep.args)

		parsed_value, err := Convert(field_args, field.LogicalTypeConfig)
		if err != nil {
			return nil, fmt.Errorf("column '%s': failed to convert '%s' to %v: %s (resolved args: %v)", field.Name, value, field.RawLogicalTypeConfig["name"], err, field_args)
		}
		return parsed_value, nil
	}
	return nil, fmt.Errorf("column '%s': value '%s' did not match any pattern", field.Name, value)
}

func (p *Parser) ParseData(input_channel chan reader.Record, output_channel chan map[string]any, wg *sync.WaitGroup, errTracker *error_tracking.ErrorTracker) {
	defer wg.Done()
	defer close(output_channel)

	// always drain the input channel so the reader never blocks; the reader
	// stops and closes it when processing is killed
	for row := range input_channel {
		resolvedRow := map[string]any{}
		failures := []string{}

		// check every cell so a dropped row reports all of its problems
		for i, value := range row.Values {
			parsed_value, err := p.parseValue(i, *value)
			if err != nil {
				failures = append(failures, err.Error())
				continue
			}
			resolvedRow[p.config.Fields[i].Name] = parsed_value
		}

		if len(failures) > 0 {
			if p.opts.ErrorOnDataFailures {
				for _, failure := range failures {
					errTracker.AddReportError(fmt.Sprintf("row %d, %s", row.Line, failure), error_tracking.Cell)
				}
			} else {
				p.DroppedRows++
				if p.dataFailureWriter != nil {
					reportRow := append(append([]string{}, row.Raw...), strings.Join(failures, "; "))
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
