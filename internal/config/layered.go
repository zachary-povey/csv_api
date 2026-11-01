package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// overrideConfig is the content of a config that extends another one. It
// holds only input-side settings, plus the schema-side keys (name, nullable,
// logical_type) which are decoded only so they can be rejected with a clear
// error. It is decoded strictly, so any other key fails the load.
type overrideConfig struct {
	Extends           string          `yaml:"extends"`
	Name              *string         `yaml:"name"`
	AllowExtraFields  *bool           `yaml:"allow_extra_fields"`
	EmptyIsNull       *bool           `yaml:"empty_is_null"`
	QuotedEmptyIsNull *bool           `yaml:"quoted_empty_is_null"`
	Fields            []fieldOverride `yaml:"fields"`
}

// fieldOverride changes the input-side settings of one base field, matched by
// name. Pointers (and nil slices) tell an unset setting from an empty one.
type fieldOverride struct {
	Name                   string            `yaml:"name"`
	Representations        *[]Representation `yaml:"representations"`
	PrependRepresentations []Representation  `yaml:"prepend_representations"`
	AppendRepresentations  []Representation  `yaml:"append_representations"`
	HeaderPatterns         *[]string         `yaml:"header_patterns"`
	ErrorMessage           *string           `yaml:"error_message"`
	Required               *bool             `yaml:"required"`
	EmptyIsNull            *bool             `yaml:"empty_is_null"`
	QuotedEmptyIsNull      *bool             `yaml:"quoted_empty_is_null"`

	// schema-side: rejected if present
	Nullable    *bool `yaml:"nullable"`
	LogicalType any   `yaml:"logical_type"`
}

// configFile is one file of an extends chain.
type configFile struct {
	path string
	data []byte
}

// readChain reads the config at path and every config it extends, returning
// them leaf first and root last. An extends path is resolved relative to the
// file containing it.
func readChain(path string) ([]configFile, error) {
	var chain []configFile
	seen := map[string]bool{}
	for {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("config '%s': %w", path, err)
		}
		if seen[abs] {
			names := make([]string, 0, len(chain)+1)
			for _, file := range chain {
				names = append(names, file.path)
			}
			names = append(names, path)
			return nil, fmt.Errorf("extends cycle: %s", strings.Join(names, " -> "))
		}
		seen[abs] = true

		data, err := os.ReadFile(path)
		if err != nil {
			if len(chain) > 0 {
				return nil, fmt.Errorf("config '%s' extends '%s', which can't be read: %w",
					chain[len(chain)-1].path, path, err)
			}
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		chain = append(chain, configFile{path: path, data: data})

		// only the extends key is read here; each file is decoded strictly later
		var header struct {
			Extends string `yaml:"extends"`
		}
		err = yaml.Unmarshal(data, &header)
		if err != nil {
			return nil, fmt.Errorf("config '%s': failed to unmarshal yaml: %w", path, err)
		}
		if header.Extends == "" {
			return chain, nil
		}
		next := header.Extends
		if !filepath.IsAbs(next) {
			next = filepath.Join(filepath.Dir(path), next)
		}
		path = next
	}
}

// loadMerged reads the extends chain starting at path and merges it into one
// Config, root first and later files winning. The result is not yet checked.
// extended reports whether path extends another config.
func loadMerged(path string) (config Config, extended bool, err error) {
	chain, err := readChain(path)
	if err != nil {
		return Config{}, false, err
	}
	extended = len(chain) > 1

	root := chain[len(chain)-1]
	err = yaml.UnmarshalStrict(root.data, &config)
	if err != nil {
		if len(chain) > 1 {
			return Config{}, false, fmt.Errorf("config '%s': failed to unmarshal yaml: %w", root.path, err)
		}
		return Config{}, false, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	for i := len(chain) - 2; i >= 0; i-- {
		file := chain[i]
		var override overrideConfig
		err = yaml.UnmarshalStrict(file.data, &override)
		if err != nil {
			return Config{}, false, fmt.Errorf("config '%s': failed to unmarshal yaml: %w", file.path, err)
		}
		err = config.applyOverride(override)
		if err != nil {
			return Config{}, false, fmt.Errorf("config '%s' (extends '%s'): %w", file.path, override.Extends, err)
		}
	}
	return config, extended, nil
}

const inputSideOnly = "a config that extends another may only change input-side settings " +
	"(representations, prepend_representations, append_representations, header_patterns, " +
	"error_message, required, empty_is_null, quoted_empty_is_null, allow_extra_fields), so the output schema stays the root's"

// applyOverride merges an extending file's settings onto the config.
func (conf *Config) applyOverride(override overrideConfig) error {
	if override.Name != nil {
		return fmt.Errorf("can't set the schema 'name': %s", inputSideOnly)
	}
	if override.AllowExtraFields != nil {
		conf.AllowExtraFields = override.AllowExtraFields
	}
	if override.EmptyIsNull != nil {
		conf.EmptyIsNull = override.EmptyIsNull
	}
	if override.QuotedEmptyIsNull != nil {
		conf.QuotedEmptyIsNull = override.QuotedEmptyIsNull
	}

	// copy the fields so changes never reach slices shared with the base
	fields := make([]FieldConfig, len(conf.Fields))
	copy(fields, conf.Fields)
	seen := map[string]bool{}
	for _, fo := range override.Fields {
		if fo.Name == "" {
			return fmt.Errorf("a field is missing 'name': fields are matched to the base by name")
		}
		if seen[fo.Name] {
			return fmt.Errorf("field '%s' is listed more than once", fo.Name)
		}
		seen[fo.Name] = true

		index := -1
		for i := range fields {
			if fields[i].Name == fo.Name {
				index = i
				break
			}
		}
		if index < 0 {
			return fmt.Errorf("field '%s' is not in the base config: %s",
				fo.Name, inputSideOnly)
		}
		if fo.Nullable != nil {
			return fmt.Errorf("field '%s': can't set 'nullable': %s", fo.Name, inputSideOnly)
		}
		if fo.LogicalType != nil {
			return fmt.Errorf("field '%s': can't set 'logical_type': %s", fo.Name, inputSideOnly)
		}

		field := &fields[index]
		if fo.Representations != nil {
			field.Representations = *fo.Representations
		}
		if fo.PrependRepresentations != nil || fo.AppendRepresentations != nil {
			reps := make([]Representation, 0,
				len(fo.PrependRepresentations)+len(field.Representations)+len(fo.AppendRepresentations))
			reps = append(reps, fo.PrependRepresentations...)
			reps = append(reps, field.Representations...)
			reps = append(reps, fo.AppendRepresentations...)
			field.Representations = reps
		}
		if fo.HeaderPatterns != nil {
			field.HeaderPatterns = *fo.HeaderPatterns
		}
		if fo.ErrorMessage != nil {
			field.ErrorMessage = *fo.ErrorMessage
		}
		if fo.Required != nil {
			field.Required = fo.Required
		}
		if fo.EmptyIsNull != nil {
			field.EmptyIsNull = fo.EmptyIsNull
		}
		if fo.QuotedEmptyIsNull != nil {
			field.QuotedEmptyIsNull = fo.QuotedEmptyIsNull
		}
	}
	conf.Fields = fields
	return nil
}
