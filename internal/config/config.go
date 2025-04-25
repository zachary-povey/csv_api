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

type Representation struct {
	Pattern string         `yaml:"pattern"`
	IsNull  bool           `yaml:"is_null"`
	Args    map[string]any `yaml:"args"`
}

type LogicalTypeConfig struct {
	Name LogicalType    `yaml:"name"`
	Args map[string]any `yaml:"args"`
}

type FieldConfig struct {
	Name              string            `yaml:"name"`
	LogicalTypeConfig LogicalTypeConfig `yaml:"logical_type"`
	Representations   []Representation  `yaml:"representations"`
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

func (conf Config) AllFieldNames() []string {
	fieldNames := []string{}
	for _, field := range conf.Fields {
		fieldNames = append(fieldNames, field.Name)
	}

	return fieldNames
}

func (conf Config) RequiredFieldNames() []string {
	fieldNames := []string{}
	for _, field := range conf.Fields {
		fieldNames = append(fieldNames, field.Name)
	}

	return fieldNames
}

func (conf Config) FieldMap() map[string]FieldConfig {
	fieldMap := map[string]FieldConfig{}
	for _, field := range conf.Fields {
		fieldMap[field.Name] = field
	}
	return fieldMap
}
