package parser

import (
	"fmt"
	"regexp"
	"sync"

	"maps"

	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
)

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

func ParseData(config *config.Config, input_channel chan []*string, output_channel chan map[string]any, wg *sync.WaitGroup, errTracker error_tracking.ErrorTracker) {
	defer wg.Done()
	defer close(output_channel)

	for row := range input_channel {
		resolvedRow := map[string]any{}

		for i, value := range row {
			field := config.Fields[i]

			any_matched := false
			field_args := make(map[string]any)
			for _, rep := range field.Representations {
				pattern, err := EnsureValueName(rep.Pattern)
				if err != nil {
					errTracker.AddExecutionError(fmt.Errorf("failed to add 'value' name to named capture group: %s", rep.Pattern))
					return
				}

				rgx, err := regexp.Compile(pattern)
				if err != nil {
					errTracker.AddExecutionError(fmt.Errorf("error parsing regex: %s", err))
					return
				}
				matched, args := extract_args(rgx, *value)
				if matched {
					any_matched = true

					// merge args from regex with static ones on representation
					// values from regex match have priority if there is overlap
					maps.Copy(field_args, rep.Args)
					for k, v := range args {
						field_args[k] = v
					}
					break
				}
			}
			if !any_matched {
				errTracker.AddReportError(fmt.Sprintf("value '%s' did not match any pattern in column '%s'", *value, field.Name), "cell")
			}
			parsed_value, err := Convert(field_args, field.LogicalType)
			if err != nil {
				errTracker.AddReportError(fmt.Sprintf("Failed to convert '%s' to type '%s'\nResolved args: %v\nException:\n %s", *value, field.LogicalType, field_args, err), "cell")
				return
			}
			fmt.Println("!!!", parsed_value)
			resolvedRow[field.Name] = parsed_value
		}
		output_channel <- resolvedRow
	}
	// pull off messages from q
	// validate against regex
}
