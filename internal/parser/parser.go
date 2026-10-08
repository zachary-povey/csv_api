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

// EnsureValueName adds ?P<value> to the single *unnamed* capture
// group in pattern. If the pattern has zero or more than one
// unnamed capture, it is returned untouched.
func EnsureValueName(pattern string) (string, error) {
	var (
		inClass      bool // inside [...]
		escaped      bool // previous byte was '\'
		unnamed_caps int  // unnamed capture count
		total_caps   int  // total capture count
		firstCapPos  = -1 // index of that '('
	)

	for i := 0; i < len(pattern); i++ {
		b := pattern[i]

		// honour escapes everywhere
		if escaped {
			escaped = false
			continue
		}
		if b == '\\' {
			escaped = true
			continue
		}

		// track character classes so we ignore ( ... ) in [...]
		if b == '[' {
			inClass = true
			continue
		}
		if b == ']' && inClass {
			inClass = false
			continue
		}
		if inClass {
			continue
		}

		// an opening '(' outside a class?
		if b == '(' {
			total_caps++
			// if the next rune is '?', it's already named or non-capturing
			if i+1 < len(pattern) && pattern[i+1] == '?' {
				continue
			}
			unnamed_caps++
			if unnamed_caps == 1 {
				firstCapPos = i
			}
		}
	}

	// No capture groups? wrap entire pattern in a single capture group
	if total_caps == 0 {
		return fmt.Sprintf("(?P<value>%s)", pattern), nil
		// only one unnamed capture?  rename it
	} else if unnamed_caps == 1 {
		named := pattern[:firstCapPos] + "(?P<value>" + pattern[firstCapPos+1:]
		// make sure we didn’t break the regex
		if _, err := regexp.Compile(named); err != nil {
			return "", fmt.Errorf("after naming: %w", err)
		}
		return named, nil
	}
	return pattern, nil
}

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

// NewParser compiles every representation pattern in the config up front so
// a bad pattern is reported once rather than per cell.
func NewParser(config *config.Config, opts options.Options, dataFailureWriter *csv.Writer) (*Parser, error) {
	representations := make([][]compiledRepresentation, len(config.Fields))
	for i, field := range config.Fields {
		for _, rep := range field.Representations {
			pattern, err := EnsureValueName(rep.Pattern)
			if err != nil {
				return nil, fmt.Errorf("failed to add 'value' name to capture group in pattern '%s' for field '%s': %w", rep.Pattern, field.Name, err)
			}
			rgx, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("error parsing regex for field '%s': %w", field.Name, err)
			}
			representations[i] = append(representations[i], compiledRepresentation{rgx, rep.Args})
		}
	}
	return &Parser{
		config:            config,
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
