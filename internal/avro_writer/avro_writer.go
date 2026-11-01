package avro_writer

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/linkedin/goavro/v2"
	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
)

const tmpl string = `
{
	"name": "{{.Name}}",
	"type": "record",
	"fields": [
	{{- $total := len .Fields }}
	{{- range $i, $value := .Fields }}
		{"name": "{{$value.Name}}", "type": {{field_type_json $value}}{{if $value.Nullable}}, "default": null{{end}}}{{if ne $i (subtract $total 1)}},{{end}}
	{{- end }}
	]
}`

var type_mappings = map[config.LogicalType]string{
	config.Integer: "long",
	config.String:  "string",
	config.Decimal: "double", // Use double for decimals (both float and precise)
}

var funcMap template.FuncMap = template.FuncMap{
	"subtract": func(a, b int) int {
		return a - b
	},
	"map_type": func(logical_type config.LogicalType) string {
		avro_type := type_mappings[logical_type]
		if avro_type == "" {
			panic("Unknown logical type: " + logical_type)
		}
		return avro_type
	},
	"field_type_json": func(field config.FieldConfig) string {
		if field.Nullable {
			return fmt.Sprintf(`["null", %s]`, map_type_json(field))
		}
		return map_type_json(field)
	},
}

func map_type_json(field config.FieldConfig) string {
	switch ltc := field.LogicalTypeConfig.(type) {
	case config.IntegerTypeConfig:
		return `"long"`
	case config.StringTypeConfig:
		return `"string"`
	case config.DecimalTypeConfig:
		if ltc.Args.AsFloat {
			return `"double"`
		} else {
			return fmt.Sprintf(`{"type":"bytes","logicalType":"decimal","precision":%d,"scale":%d}`,
				ltc.Args.Precision, ltc.Args.Scale)
		}
	case config.EnumTypeConfig:
		symbols := make([]string, len(ltc.Args.PermittedValues))
		for i, v := range ltc.Args.PermittedValues {
			symbols[i] = fmt.Sprintf(`"%s"`, v)
		}
		return fmt.Sprintf(`{"type":"enum","name":"%s","symbols":[%s]}`,
			field.Name, strings.Join(symbols, ","))
	case config.DateTypeConfig:
		return `{"type":"int","logicalType":"date"}`
	case config.TimeTypeConfig:
		return `{"type":"long","logicalType":"time-micros"}`
	case config.TimestampTypeConfig:
		return `{"type":"long","logicalType":"timestamp-micros"}`
	case config.BooleanTypeConfig:
		return `"boolean"`
	default:
		panic(fmt.Sprintf("Unknown logical type config: %T", field.LogicalTypeConfig))
	}
}

// unionBranch returns the name goavro uses for a field's non-null branch in
// a ["null", <type>] union. goavro's native form of a non-null union value is
// map[string]any{<branch>: value}. namespace is the record's namespace, which
// enum types (named after their field) inherit.
func unionBranch(field config.FieldConfig, namespace string) string {
	switch ltc := field.LogicalTypeConfig.(type) {
	case config.IntegerTypeConfig:
		return "long"
	case config.StringTypeConfig:
		return "string"
	case config.DecimalTypeConfig:
		if ltc.Args.AsFloat {
			return "double"
		}
		return "bytes.decimal"
	case config.EnumTypeConfig:
		if namespace == "" {
			return field.Name
		}
		return namespace + "." + field.Name
	case config.DateTypeConfig:
		return "int.date"
	case config.TimeTypeConfig:
		return "long.time-micros"
	case config.TimestampTypeConfig:
		return "long.timestamp-micros"
	case config.BooleanTypeConfig:
		return "boolean"
	default:
		panic(fmt.Sprintf("Unknown logical type config: %T", field.LogicalTypeConfig))
	}
}

// nullableBranches maps each nullable field's name to its union branch name.
func nullableBranches(conf *config.Config) map[string]string {
	namespace := ""
	if i := strings.LastIndex(conf.Name, "."); i >= 0 {
		namespace = conf.Name[:i]
	}
	branches := map[string]string{}
	for _, field := range conf.Fields {
		if field.Nullable {
			branches[field.Name] = unionBranch(field, namespace)
		}
	}
	return branches
}

func generateSchema(config *config.Config) string {
	t, err := template.New("schemaTemplate").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		panic(err)
	}
	var builder strings.Builder
	if err := t.Execute(&builder, config); err != nil {
		panic(err)
	}

	result := builder.String()
	return result
}

func WriteFile(filepath string, config *config.Config, channel chan map[string]any, wg *sync.WaitGroup, errTracker *error_tracking.ErrorTracker) {
	defer wg.Done()

	avroSchema := generateSchema(config)

	output, err := os.Create(filepath)
	if err != nil {
		errTracker.AddExecutionError(fmt.Errorf("error opening output file: %s", err))
		return
	}
	defer output.Close()

	writer, writerErr := goavro.NewOCFWriter(
		goavro.OCFConfig{
			W:      output,
			Schema: avroSchema,
		},
	)
	if writerErr != nil {
		errTracker.AddExecutionError(fmt.Errorf("error setting up avro writer: %s", writerErr))
		return
	}

	branches := nullableBranches(config)
	for row := range channel {
		for name, branch := range branches {
			if value := row[name]; value != nil {
				row[name] = map[string]any{branch: value}
			}
		}
		if err := writer.Append([]map[string]interface{}{row}); err != nil {
			errTracker.AddExecutionError(fmt.Errorf("error writing avro record: %s", err))
			return
		}
	}
}
