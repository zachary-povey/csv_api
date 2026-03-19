package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Fields []FieldConfig `yaml:"fields" validate:"required,dive"`
}

type FieldConfig struct {
	Name              string            `yaml:"name"`
	LogicalTypeConfig LogicalTypeConfig `yaml:"-"`
	Representations   []Representation  `yaml:"representations"`

	RawLogicalTypeConfig map[string]any `yaml:"logical_type" validate:"required"`
}

type Representation struct {
	Pattern string         `yaml:"pattern"`
	IsNull  bool           `yaml:"is_null"`
	Args    map[string]any `yaml:"args"`
}

type LogicalType string

const (
	String    LogicalType = "string"
	Integer   LogicalType = "integer"
	Decimal   LogicalType = "decimal"
	Enum      LogicalType = "enum"
	Timestamp LogicalType = "timestamp"
	Date      LogicalType = "date"
	Time      LogicalType = "time"
)

type LogicalTypeConfig interface {
	isLjc()
}

type IntegerTypeConfig struct {
	Name LogicalType `validate:"required"`
}

func (IntegerTypeConfig) isLjc() {}

type StringTypeConfig struct {
	Name LogicalType `validate:"required"`
}

func (StringTypeConfig) isLjc() {}

type DecimalTypeConfig struct {
	Name LogicalType     `validate:"required"`
	Args DecimalTypeArgs `yaml:"args"`
}

type DecimalTypeArgs struct {
	AsFloat bool `mapstructure:"as_float" validate:"required_if=Precision 0"`

	Precision int `validate:"required_if=AsFloat false,excluded_if=AsFloat true"`
	Scale     int `validate:"required_if=AsFloat false,excluded_if=AsFloat true"`
}

func (DecimalTypeConfig) isLjc() {}

type EnumTypeConfig struct {
	Name LogicalType  `validate:"required"`
	Args EnumTypeArgs `yaml:"args"`
}

type EnumTypeArgs struct {
	PermittedValues []string `mapstructure:"permitted_values" validate:"required"`
}

func (EnumTypeConfig) isLjc() {}

type TimestampTypeConfig struct {
	Name LogicalType `validate:"required"`
}

func (TimestampTypeConfig) isLjc() {}

type TimeTypeConfig struct {
	Name LogicalType `validate:"required"`
}

func (TimeTypeConfig) isLjc() {}

type DateTypeConfig struct {
	Name LogicalType `validate:"required"`
}

func (DateTypeConfig) isLjc() {}

func (fc *FieldConfig) UnmarshalTypeConfigs() error {
	logicalTypeName := fc.RawLogicalTypeConfig["name"]
	logicalTypeNameStr, ok := logicalTypeName.(string)
	if !ok {
		return fmt.Errorf("logical type name is not a string: %v", logicalTypeName)
	}
	logicalTypeName = LogicalType(logicalTypeNameStr)

	var narrowingError error
	switch logicalTypeName {
	case Integer:
		fc.LogicalTypeConfig, narrowingError = narrowType[IntegerTypeConfig](*fc)
	case String:
		fc.LogicalTypeConfig, narrowingError = narrowType[StringTypeConfig](*fc)
	case Decimal:
		fc.LogicalTypeConfig, narrowingError = narrowType[DecimalTypeConfig](*fc)
	case Enum:
		fc.LogicalTypeConfig, narrowingError = narrowType[EnumTypeConfig](*fc)
	case Timestamp:
		fc.LogicalTypeConfig, narrowingError = narrowType[TimestampTypeConfig](*fc)
	case Time:
		fc.LogicalTypeConfig, narrowingError = narrowType[TimeTypeConfig](*fc)
	case Date:
		fc.LogicalTypeConfig, narrowingError = narrowType[DateTypeConfig](*fc)
	default:
		return fmt.Errorf("unknown logical type in config: '%s'", logicalTypeName)
	}

	return narrowingError
}

func narrowType[T LogicalTypeConfig](fc FieldConfig) (T, error) {
	var typedConfig T
	err := mapstructure.Decode(fc.RawLogicalTypeConfig, &typedConfig)
	if err != nil {
		return typedConfig, err
	}

	// todo: validate no extras
	validate := validator.New(validator.WithRequiredStructEnabled())
	err = validate.Struct(typedConfig)
	if err != nil {
		return typedConfig, err
	}

	return typedConfig, nil
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
	validate := validator.New(validator.WithRequiredStructEnabled())
	err = validate.Struct(config)
	if err != nil {
		return nil, err
	}

	for i := range config.Fields {
		err = config.Fields[i].UnmarshalTypeConfigs()
		if err != nil {
			return nil, err
		}
		validate.Struct(config.Fields[i].LogicalTypeConfig)
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
