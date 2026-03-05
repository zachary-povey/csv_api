package parser

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/zachary-povey/csv_api/internal/config"
)

func Convert(args map[string]any, logical_type config.LogicalTypeConfig) (any, error) {
	switch logical_type.(type) {
	case config.IntegerTypeConfig:
		return convert_int(args)
	case config.StringTypeConfig:
		return convert_string(args)
	case config.DecimalTypeConfig:
		return convert_decimal(args, logical_type.(config.DecimalTypeConfig))
	case config.EnumTypeConfig:
		return convert_enum(args, logical_type.(config.EnumTypeConfig))
	case config.DateTypeConfig:
		return convert_date(args)
	case config.TimeTypeConfig:
		return convert_time(args)
	case config.TimestampTypeConfig:
		return convert_timestamp(args)
	default:
		return nil, fmt.Errorf("unsupported logical type: %s", logical_type)

	}
}

func convert_int(args map[string]any) (int, error) {
	value, exists := args["value"]
	if !exists {
		return 0, fmt.Errorf("missing 'value' argument")
	}
	string_value, ok := value.(string)
	if !ok {
		return 0, fmt.Errorf("'value' in args is not of type string")
	}
	intValue, err := strconv.Atoi(string_value)
	if err != nil {
		return 0, fmt.Errorf("failed to convert string to int: %w", err)
	}
	return intValue, nil
}

func convert_string(args map[string]any) (string, error) {
	value, exists := args["value"]
	if !exists {
		return "", fmt.Errorf("missing 'value' argument")
	}
	string_value, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("'value' in args is not of type string")
	}
	return string_value, nil
}

func convert_decimal(args map[string]any, config config.DecimalTypeConfig) (any, error) {
	// Check if we have a single value or separate integer/decimal parts
	if value, exists := args["value"]; exists {
		// Single value parameter set
		string_value, ok := value.(string)
		if !ok {
			return 0, fmt.Errorf("'value' in args is not of type string")
		}

		floatValue, err := strconv.ParseFloat(string_value, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to convert string to decimal: %w", err)
		}

		if config.Args.AsFloat {
			return floatValue, nil
		} else {
			// Convert to Avro decimal bytes representation
			return decimalToRat(string_value)
		}

	} else if integer_part, has_int := args["integer_part"]; has_int {
		// Separate integer_part and decimal_part parameter set
		decimal_part, has_dec := args["decimal_part"]
		if !has_dec {
			return 0, fmt.Errorf("missing 'decimal_part' argument when 'integer_part' is provided")
		}

		// Convert integer part
		var intPart float64
		switch int_val := integer_part.(type) {
		case string:
			parsed_int, err := strconv.ParseFloat(int_val, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to convert integer_part to number: %w", err)
			}
			intPart = parsed_int
		case int:
			intPart = float64(int_val)
		default:
			return 0, fmt.Errorf("'integer_part' must be string or integer")
		}

		// Convert decimal part
		var decPart float64
		switch dec_val := decimal_part.(type) {
		case string:
			parsed_dec, err := strconv.ParseFloat(dec_val, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to convert decimal_part to number: %w", err)
			}
			decPart = parsed_dec
		case int:
			decPart = float64(dec_val)
		default:
			return 0, fmt.Errorf("'decimal_part' must be string or integer")
		}

		// Combine parts: determine the scale of decimal part
		scale := 1.0
		if decPart > 0 {
			// Count digits to determine appropriate scale
			temp := decPart
			for temp >= 1 {
				scale *= 10
				temp /= 10
			}
		}

		result := intPart + (decPart / scale)

		if config.Args.AsFloat {
			return result, nil
		} else {
			// Convert to Avro decimal bytes representation
			resultStr := fmt.Sprintf("%.10g", result)
			return decimalToRat(resultStr)
		}

	} else {
		return 0, fmt.Errorf("missing required arguments: need either 'value' or 'integer_part'+'decimal_part'")
	}
}

func convert_enum(args map[string]any, config config.EnumTypeConfig) (string, error) {
	value, exists := args["value"]
	if !exists {
		return "", fmt.Errorf("missing 'value' argument")
	}
	string_value, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("'value' in args is not of type string")
	}
	for _, permitted := range config.Args.PermittedValues {
		if string_value == permitted {
			return string_value, nil
		}
	}
	return "", fmt.Errorf("invalid enum value '%s', permitted values are: [%s]",
		string_value, strings.Join(config.Args.PermittedValues, ", "))
}

func argToInt(args map[string]any, key string) (int, error) {
	val, exists := args[key]
	if !exists {
		return 0, fmt.Errorf("missing '%s' argument", key)
	}
	switch v := val.(type) {
	case string:
		return strconv.Atoi(v)
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("'%s' must be string or integer", key)
	}
}

func convert_date(args map[string]any) (int32, error) {
	year, err := argToInt(args, "year")
	if err != nil {
		return 0, fmt.Errorf("failed to extract year: %w", err)
	}
	month, err := argToInt(args, "month")
	if err != nil {
		return 0, fmt.Errorf("failed to extract month: %w", err)
	}
	day, err := argToInt(args, "day")
	if err != nil {
		return 0, fmt.Errorf("failed to extract day: %w", err)
	}

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Year() != year || date.Month() != time.Month(month) || date.Day() != day {
		return 0, fmt.Errorf("invalid date: %04d-%02d-%02d", year, month, day)
	}

	epoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	days := int32(date.Sub(epoch).Hours() / 24)
	return days, nil
}

func argToIntDefault(args map[string]any, key string, defaultVal int) (int, error) {
	val, exists := args[key]
	if !exists {
		return defaultVal, nil
	}
	switch v := val.(type) {
	case string:
		if v == "" {
			return defaultVal, nil
		}
		return strconv.Atoi(v)
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("'%s' must be string or integer", key)
	}
}

func convert_time(args map[string]any) (int64, error) {
	hour, err := argToIntDefault(args, "hour", 0)
	if err != nil {
		return 0, fmt.Errorf("failed to extract hour: %w", err)
	}
	minute, err := argToIntDefault(args, "minute", 0)
	if err != nil {
		return 0, fmt.Errorf("failed to extract minute: %w", err)
	}
	second, err := argToIntDefault(args, "second", 0)
	if err != nil {
		return 0, fmt.Errorf("failed to extract second: %w", err)
	}
	millisecond, err := argToIntDefault(args, "millisecond", 0)
	if err != nil {
		return 0, fmt.Errorf("failed to extract millisecond: %w", err)
	}
	microsecond, err := argToIntDefault(args, "microsecond", 0)
	if err != nil {
		return 0, fmt.Errorf("failed to extract microsecond: %w", err)
	}

	if hour < 0 || hour > 23 {
		return 0, fmt.Errorf("hour must be between 0 and 23, got %d", hour)
	}
	if minute < 0 || minute > 59 {
		return 0, fmt.Errorf("minute must be between 0 and 59, got %d", minute)
	}
	if second < 0 || second > 59 {
		return 0, fmt.Errorf("second must be between 0 and 59, got %d", second)
	}
	if millisecond < 0 || millisecond > 999 {
		return 0, fmt.Errorf("millisecond must be between 0 and 999, got %d", millisecond)
	}
	if microsecond < 0 || microsecond > 999999 {
		return 0, fmt.Errorf("microsecond must be between 0 and 999999, got %d", microsecond)
	}

	micros := int64(hour)*3600000000 + int64(minute)*60000000 + int64(second)*1000000 + int64(millisecond)*1000 + int64(microsecond)
	return micros, nil
}

func convert_timestamp(args map[string]any) (int64, error) {
	// Parameter set 1: epoch-based (value + precision + optional offset)
	if value, exists := args["value"]; exists {
		var epochVal int64
		switch v := value.(type) {
		case string:
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to convert timestamp value to int: %w", err)
			}
			epochVal = parsed
		case int:
			epochVal = int64(v)
		default:
			return 0, fmt.Errorf("'value' must be string or integer")
		}

		precision, exists := args["precision"]
		if !exists {
			return 0, fmt.Errorf("missing 'precision' argument for epoch-based timestamp")
		}
		precisionStr, ok := precision.(string)
		if !ok {
			return 0, fmt.Errorf("'precision' must be a string")
		}

		offset, err := argToIntDefault(args, "offset", 0)
		if err != nil {
			return 0, fmt.Errorf("failed to extract offset: %w", err)
		}

		var micros int64
		switch precisionStr {
		case "seconds":
			micros = (epochVal + int64(offset)) * 1000000
		case "milliseconds":
			micros = (epochVal + int64(offset)*1000) * 1000
		case "microseconds":
			micros = epochVal + int64(offset)*1000000
		default:
			return 0, fmt.Errorf("unsupported precision '%s', must be 'seconds', 'milliseconds', or 'microseconds'", precisionStr)
		}
		return micros, nil
	}

	// Parameter set 2: component-based
	year, err := argToInt(args, "year")
	if err != nil {
		return 0, fmt.Errorf("failed to extract year: %w", err)
	}
	month, err := argToInt(args, "month")
	if err != nil {
		return 0, fmt.Errorf("failed to extract month: %w", err)
	}
	day, err := argToInt(args, "day")
	if err != nil {
		return 0, fmt.Errorf("failed to extract day: %w", err)
	}
	hour, err := argToIntDefault(args, "hour", 0)
	if err != nil {
		return 0, fmt.Errorf("failed to extract hour: %w", err)
	}
	minute, err := argToIntDefault(args, "minute", 0)
	if err != nil {
		return 0, fmt.Errorf("failed to extract minute: %w", err)
	}
	second, err := argToIntDefault(args, "second", 0)
	if err != nil {
		return 0, fmt.Errorf("failed to extract second: %w", err)
	}
	millisecond, err := argToIntDefault(args, "millisecond", 0)
	if err != nil {
		return 0, fmt.Errorf("failed to extract millisecond: %w", err)
	}
	microsecond, err := argToIntDefault(args, "microsecond", 0)
	if err != nil {
		return 0, fmt.Errorf("failed to extract microsecond: %w", err)
	}

	// Parse timezone if present
	loc := time.UTC
	if tz, exists := args["timezone"]; exists {
		tzStr, ok := tz.(string)
		if !ok {
			return 0, fmt.Errorf("'timezone' must be a string")
		}
		if tzStr != "" {
			parsed, err := time.Parse("-07:00", tzStr)
			if err != nil {
				return 0, fmt.Errorf("failed to parse timezone '%s': %w", tzStr, err)
			}
			_, offset := parsed.Zone()
			loc = time.FixedZone("", offset)
		}
	}

	// Validate date components
	date := time.Date(year, time.Month(month), day, hour, minute, second, 0, loc)
	if date.Year() != year || date.Month() != time.Month(month) || date.Day() != day {
		return 0, fmt.Errorf("invalid date in timestamp: %04d-%02d-%02d", year, month, day)
	}
	if hour < 0 || hour > 23 {
		return 0, fmt.Errorf("hour must be between 0 and 23, got %d", hour)
	}
	if minute < 0 || minute > 59 {
		return 0, fmt.Errorf("minute must be between 0 and 59, got %d", minute)
	}
	if second < 0 || second > 59 {
		return 0, fmt.Errorf("second must be between 0 and 59, got %d", second)
	}

	// Convert to UTC microseconds since epoch
	utcTime := date.UTC()
	epoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	micros := utcTime.Sub(epoch).Microseconds() + int64(millisecond)*1000 + int64(microsecond)
	return micros, nil
}

// decimalToRat converts a decimal string to a *big.Rat for goavro's decimal logical type.
func decimalToRat(decimalStr string) (*big.Rat, error) {
	rat := new(big.Rat)
	_, ok := rat.SetString(decimalStr)
	if !ok {
		return nil, fmt.Errorf("invalid decimal string: %s", decimalStr)
	}
	return rat, nil
}
