package config

import (
	"fmt"
	"math"
	"regexp"
	"regexp/syntax"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CompilePattern compiles a representation pattern the way the parser matches
// it: a single unnamed capture group is named 'value', and a pattern with no
// capture groups is wrapped so the whole match is 'value' (wholeMatch).
func CompilePattern(pattern string) (rgx *regexp.Regexp, wholeMatch bool, err error) {
	tree, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil, false, err
	}

	// collect capture groups; non-capturing groups and flags aren't captures
	var captures, unnamed []*syntax.Regexp
	var walk func(re *syntax.Regexp)
	walk = func(re *syntax.Regexp) {
		if re.Op == syntax.OpCapture {
			captures = append(captures, re)
			if re.Name == "" {
				unnamed = append(unnamed, re)
			}
		}
		for _, sub := range re.Sub {
			walk(sub)
		}
	}
	walk(tree)

	named := pattern
	if len(captures) == 0 {
		// no capture groups: the whole match is 'value'
		named = fmt.Sprintf("(?P<value>%s)", pattern)
		wholeMatch = true
	} else if len(unnamed) == 1 {
		unnamed[0].Name = "value"
		named = tree.String()
	}

	rgx, err = regexp.Compile(named)
	if err != nil {
		return nil, false, err
	}
	return rgx, wholeMatch, nil
}

// IntArg converts a value arg to an int. Capture groups give strings; static
// args decoded from YAML may be numbers. yaml.v2 decodes whole numbers as int
// (int64/uint64 when large) and others as float64; a float is only accepted
// when it has no fractional part.
func IntArg(val any, key string) (int, error) {
	switch v := val.(type) {
	case string:
		return strconv.Atoi(v)
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("'%s' value %d is out of range", key, v)
		}
		return int(v), nil
	case float64:
		if v != math.Trunc(v) || v < math.MinInt64 || v >= math.MaxInt64 {
			return 0, fmt.Errorf("'%s' value %v is not a whole number", key, v)
		}
		return int(v), nil
	default:
		return 0, fmt.Errorf("'%s' must be string or integer", key)
	}
}

// argSet is one combination of value args a logical type's converter accepts.
type argSet struct {
	required []string
	optional []string
}

var (
	timeParts = []string{"hour", "minute", "second", "millisecond", "microsecond"}

	// valueArgSets lists the value args each logical type accepts (see the
	// README's "value args each type accepts" table).
	valueArgSets = map[LogicalType][]argSet{
		String:  {{required: []string{"value"}}},
		Integer: {{required: []string{"value"}}},
		Enum:    {{required: []string{"value"}}},
		Decimal: {
			{required: []string{"value"}},
			{required: []string{"integer_part", "decimal_part"}},
		},
		Date: {{required: []string{"year", "month", "day"}}},
		Time: {{optional: timeParts}},
		Timestamp: {
			{required: []string{"value", "precision"}, optional: []string{"offset"}},
			{required: []string{"year", "month", "day"}, optional: append(slices.Clone(timeParts), "timezone")},
		},
	}
)

func (s argSet) allows(provided map[string]bool) bool {
	for _, name := range s.required {
		if !provided[name] {
			return false
		}
	}
	for name := range provided {
		if !slices.Contains(s.required, name) && !slices.Contains(s.optional, name) {
			return false
		}
	}
	return true
}

func (s argSet) String() string {
	desc := strings.Join(s.required, ", ")
	if len(s.optional) > 0 {
		if desc != "" {
			desc += ", "
		}
		desc += "optional " + strings.Join(s.optional, ", ")
	}
	return desc
}

func describeArgSets(sets []argSet) string {
	descs := []string{}
	for _, s := range sets {
		descs = append(descs, "("+s.String()+")")
	}
	return strings.Join(descs, " or ")
}

func sortedKeys(m map[string]bool) []string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// checkRepresentations checks that every representation of every field
// compiles and provides a valid set of value args for the field's logical
// type, and that its static args are usable values. It runs on the fully
// loaded config so config mistakes show up at load rather than per cell.
func (conf Config) checkRepresentations() error {
	for _, field := range conf.Fields {
		for _, rep := range field.Representations {
			if err := field.checkRepresentation(rep); err != nil {
				return fmt.Errorf("field '%s', pattern '%s': %w", field.Name, rep.Pattern, err)
			}
		}
	}
	return nil
}

func (fc FieldConfig) checkRepresentation(rep Representation) error {
	rgx, wholeMatch, err := CompilePattern(rep.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}
	// an is_null representation resolves to null and provides no value args
	if rep.IsNull {
		return nil
	}

	typeName := LogicalType(fc.RawLogicalTypeConfig["name"].(string))
	sets := valueArgSets[typeName]

	provided := map[string]bool{}
	for _, name := range rgx.SubexpNames()[1:] {
		if name != "" {
			provided[name] = true
		}
	}
	// The whole match of a pattern with no capture groups is passed as
	// 'value'. Types without a 'value' arg ignore it, so a fixed pattern like
	// "^start$" with static date args is fine.
	if wholeMatch && !slices.ContainsFunc(sets, func(s argSet) bool {
		return slices.Contains(s.required, "value") || slices.Contains(s.optional, "value")
	}) {
		delete(provided, "value")
	}
	for name := range rep.Args {
		provided[name] = true
	}

	unknown := map[string]bool{}
	for name := range provided {
		known := slices.ContainsFunc(sets, func(s argSet) bool {
			return slices.Contains(s.required, name) || slices.Contains(s.optional, name)
		})
		if !known {
			unknown[name] = true
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown value args [%s] for %s, which accepts %s",
			strings.Join(sortedKeys(unknown), ", "), typeName, describeArgSets(sets))
	}
	if !slices.ContainsFunc(sets, func(s argSet) bool { return s.allows(provided) }) {
		return fmt.Errorf("value args [%s] (from capture groups and static args) are not a valid set for %s, which needs %s",
			strings.Join(sortedKeys(provided), ", "), typeName, describeArgSets(sets))
	}

	for _, name := range sortedKeys(provided) {
		val, static := rep.Args[name]
		if !static {
			continue
		}
		if err := checkStaticArg(fc.LogicalTypeConfig, name, val); err != nil {
			return fmt.Errorf("static arg '%s': %w", name, err)
		}
	}
	return nil
}

// checkStaticArg checks that a static arg's value is usable by the type's
// converter, so a bad value fails config load rather than every matching
// cell. Checks needing several args together (e.g. a real calendar date) are
// left to the converter.
func checkStaticArg(typeConfig LogicalTypeConfig, name string, val any) error {
	mustBeInt := func() error {
		if _, err := IntArg(val, name); err != nil {
			return fmt.Errorf("must be an integer, got %v", val)
		}
		return nil
	}
	mustBeString := func() (string, error) {
		s, ok := val.(string)
		if !ok {
			return "", fmt.Errorf("must be a string, got %v", val)
		}
		return s, nil
	}

	switch tc := typeConfig.(type) {
	case StringTypeConfig:
		_, err := mustBeString()
		return err
	case EnumTypeConfig:
		s, err := mustBeString()
		if err != nil {
			return err
		}
		if !slices.Contains(tc.Args.PermittedValues, s) {
			return fmt.Errorf("'%s' is not in permitted_values [%s]", s, strings.Join(tc.Args.PermittedValues, ", "))
		}
	case DecimalTypeConfig:
		if name == "value" {
			s, err := mustBeString()
			if err != nil {
				return err
			}
			if _, err := strconv.ParseFloat(s, 64); err != nil {
				return fmt.Errorf("'%s' is not a decimal", s)
			}
		}
	case IntegerTypeConfig, DateTypeConfig, TimeTypeConfig:
		return mustBeInt()
	case TimestampTypeConfig:
		switch name {
		case "precision":
			s, err := mustBeString()
			if err != nil {
				return err
			}
			if !slices.Contains([]string{"seconds", "milliseconds", "microseconds"}, s) {
				return fmt.Errorf("'%s' must be 'seconds', 'milliseconds' or 'microseconds'", s)
			}
		case "timezone":
			s, err := mustBeString()
			if err != nil {
				return err
			}
			if s != "" {
				if _, err := time.Parse("-07:00", s); err != nil {
					return fmt.Errorf("'%s' is not a timezone offset like +01:00", s)
				}
			}
		default:
			return mustBeInt()
		}
	}
	return nil
}
