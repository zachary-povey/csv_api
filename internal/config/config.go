package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Name   string        `yaml:"name"`
	Fields []FieldConfig `yaml:"fields" validate:"required,dive"`
	// AllowExtraFields lets the file have columns that map to no field (they
	// are ignored). Defaults to true; false makes each such column a
	// file-level error. A pointer so an unset value can be told apart.
	AllowExtraFields *bool `yaml:"allow_extra_fields"`
}

// ExtraFieldsAllowed reports whether columns that map to no field are
// ignored (the default, unless the config sets allow_extra_fields: false).
func (conf Config) ExtraFieldsAllowed() bool {
	return conf.AllowExtraFields == nil || *conf.AllowExtraFields
}

type FieldConfig struct {
	Name              string            `yaml:"name"`
	LogicalTypeConfig LogicalTypeConfig `yaml:"-"`
	Representations   []Representation  `yaml:"representations"`
	// Nullable fields may hold null (written to Avro as a union with null).
	// Only nullable fields may have is_null representations.
	Nullable bool `yaml:"nullable"`
	// Required fields must have a column in the file. Defaults to true; a
	// missing optional (required: false) column is null for every row, so
	// only nullable fields may be optional.
	Required *bool `yaml:"required"`
	// HeaderPatterns are regexes (not anchored) matched against the csv
	// header to find the field's column. When set, they replace the exact
	// match on Name.
	HeaderPatterns []string `yaml:"header_patterns"`

	headerRegexes []*regexp.Regexp

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
	Args DecimalTypeArgs `mapstructure:"args"`
}

type DecimalTypeArgs struct {
	AsFloat bool `mapstructure:"as_float" validate:"required_if=Precision 0"`

	Precision int `validate:"required_if=AsFloat false,excluded_if=AsFloat true"`
	Scale     int `validate:"required_if=AsFloat false,excluded_if=AsFloat true"`
}

func (DecimalTypeConfig) isLjc() {}

type EnumTypeConfig struct {
	Name LogicalType  `validate:"required"`
	Args EnumTypeArgs `mapstructure:"args"`
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
		return fmt.Errorf("field '%s': logical type name is not a string: %v", fc.Name, logicalTypeName)
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
		return fmt.Errorf("field '%s': unknown logical type in config: '%s'", fc.Name, logicalTypeName)
	}

	if narrowingError != nil {
		return fmt.Errorf("field '%s': invalid logical_type: %w", fc.Name, narrowingError)
	}
	return nil
}

func narrowType[T LogicalTypeConfig](fc FieldConfig) (T, error) {
	// ErrorUnused rejects keys (under logical_type or its args) that don't
	// match a field of the type's config struct, e.g. a typo in an arg name.
	// Decoding via a wrapper gives errors a 'logical_type' prefix.
	var wrapper struct {
		LogicalType T `mapstructure:"logical_type"`
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      &wrapper,
	})
	if err != nil {
		return wrapper.LogicalType, err
	}
	err = decoder.Decode(map[string]any{"logical_type": fc.RawLogicalTypeConfig})
	typedConfig := wrapper.LogicalType
	if err != nil {
		return typedConfig, err
	}

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
	err = CheckAvroName(config.Name)
	if err != nil {
		return nil, err
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	err = validate.Struct(config)
	if err != nil {
		return nil, err
	}

	err = config.checkFieldNames()
	if err != nil {
		return nil, err
	}

	for i := range config.Fields {
		err = config.Fields[i].UnmarshalTypeConfigs()
		if err != nil {
			return nil, err
		}
	}

	err = config.compileHeaderPatterns()
	if err != nil {
		return nil, err
	}

	err = config.checkRequired()
	if err != nil {
		return nil, err
	}

	err = config.checkEnumNames()
	if err != nil {
		return nil, err
	}

	err = config.checkRepresentations()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

var avroNameSegment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// CheckAvroName checks the config's top-level name, which becomes the Avro
// record name. It is required, and must be a valid Avro fullname: one or more
// dot-separated segments, each matching [A-Za-z_][A-Za-z0-9_]*. Segments
// before the last one form the Avro namespace.
func CheckAvroName(name string) error {
	if name == "" {
		return fmt.Errorf("config 'name' is required: it is used as the Avro schema name")
	}
	for _, segment := range strings.Split(name, ".") {
		if !avroNameSegment.MatchString(segment) {
			return fmt.Errorf("config name '%s' is not a valid Avro name: "+
				"each dot-separated part must start with a letter or underscore "+
				"and contain only letters, digits and underscores", name)
		}
	}
	return nil
}

// CheckFieldName checks a field's name, which becomes its Avro field name
// (and, for an enum field, its Avro enum type name). It must be a valid Avro
// name, [A-Za-z_][A-Za-z0-9_]*; unlike the config name it can't be dotted.
func CheckFieldName(name string) error {
	if name == "" {
		return fmt.Errorf("a field is missing 'name': it is used as the Avro field name")
	}
	if !avroNameSegment.MatchString(name) {
		return fmt.Errorf("field '%s': name is not a valid Avro name: it must start with a letter "+
			"or underscore and contain only letters, digits and underscores "+
			"(to match a csv header like '%s', give the field a valid name and use header_patterns)",
			name, name)
	}
	return nil
}

// checkFieldNames rejects field names that are not valid Avro names, and
// duplicate field names (an Avro record's field names must be unique).
func (conf Config) checkFieldNames() error {
	seen := map[string]bool{}
	for _, field := range conf.Fields {
		err := CheckFieldName(field.Name)
		if err != nil {
			return err
		}
		if seen[field.Name] {
			return fmt.Errorf("field '%s': duplicate field name: field names must be unique", field.Name)
		}
		seen[field.Name] = true
	}
	return nil
}

// checkEnumNames rejects an enum field named the same as the record. Enum
// types are named after their field and share the record's namespace, so the
// clash would redefine the record's named type and readers reject the file.
func (conf Config) checkEnumNames() error {
	parts := strings.Split(conf.Name, ".")
	recordName := parts[len(parts)-1]
	for _, field := range conf.Fields {
		if _, isEnum := field.LogicalTypeConfig.(EnumTypeConfig); isEnum && field.Name == recordName {
			return fmt.Errorf("field '%s': an enum field can't have the same name as the config "+
				"(its Avro enum type would clash with the record type '%s')", field.Name, conf.Name)
		}
	}
	return nil
}

// compileHeaderPatterns compiles every field's header_patterns, so an invalid
// pattern fails the config load rather than the run.
func (conf *Config) compileHeaderPatterns() error {
	for i := range conf.Fields {
		field := &conf.Fields[i]
		field.headerRegexes = nil
		for _, pattern := range field.HeaderPatterns {
			rgx, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("field '%s', header pattern '%s': %w", field.Name, pattern, err)
			}
			field.headerRegexes = append(field.headerRegexes, rgx)
		}
	}
	return nil
}

// MatchesHeader reports whether a csv column header belongs to the field: it
// matches any of the field's header_patterns or, when the field has none, it
// is exactly the field's name.
func (fc FieldConfig) MatchesHeader(header string) bool {
	if len(fc.HeaderPatterns) == 0 {
		return header == fc.Name
	}
	for _, rgx := range fc.headerRegexes {
		if rgx.MatchString(header) {
			return true
		}
	}
	return false
}

// IsRequired reports whether the field's column must be in the file
// (the default, unless the config sets required: false).
func (fc FieldConfig) IsRequired() bool {
	return fc.Required == nil || *fc.Required
}

// checkRequired rejects required: false on a field that is not nullable: a
// missing column is null for every row, so the field must allow nulls.
func (conf Config) checkRequired() error {
	for _, field := range conf.Fields {
		if !field.IsRequired() && !field.Nullable {
			return fmt.Errorf("field '%s': required: false is only allowed on a nullable field "+
				"(a missing column is null; add 'nullable: true' to the field)", field.Name)
		}
	}
	return nil
}
