package parser

import (
	"fmt"
	"strconv"

	"github.com/zachary-povey/csv_api/internal/config"
)

func Convert(args map[string]any, logical_type config.LogicalTypeConfig) (any, error) {
	switch logical_type.(type) {
	case config.IntegerTypeConfig:
		return convert_int(args)
	case config.StringTypeConfig:
		return convert_string(args)
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
