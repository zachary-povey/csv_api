package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Define the custom type
type LogicalType string

const (
	Integer  LogicalType = "integer"
	String   LogicalType = "string"
	Date     LogicalType = "date"
	Datetime LogicalType = "datetime"
	Time     LogicalType = "time"
)

type Config struct {
	Fields []FieldConfig `yaml:"fields"`
}

type FieldConfig struct {
	LogicalType LogicalType `yaml:"logical_type"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config Config
	err = yaml.UnmarshalStrict(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	return &config, nil
}
