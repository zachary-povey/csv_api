package parser

import (
	"fmt"
	"math/big"
	"strconv"

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

// decimalToRat converts a decimal string to a *big.Rat for goavro's decimal logical type.
func decimalToRat(decimalStr string) (*big.Rat, error) {
	rat := new(big.Rat)
	_, ok := rat.SetString(decimalStr)
	if !ok {
		return nil, fmt.Errorf("invalid decimal string: %s", decimalStr)
	}
	return rat, nil
}
