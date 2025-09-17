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
	"name": "test_schema",
	"type": "record",
	"fields": [
	{{- $total := len . }}
	{{- range $i, $value := . }}
		{"name": "{{$value.Name}}", "type": {{map_type_json $value.LogicalTypeConfig}}}{{if ne $i (subtract $total 1)}},{{end}}
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
	"map_type_json": func(logical_type_config config.LogicalTypeConfig) string {
		switch ltc := logical_type_config.(type) {
		case config.IntegerTypeConfig:
			return `"long"`
		case config.StringTypeConfig:
			return `"string"`
		case config.DecimalTypeConfig:
			if ltc.Args.AsFloat {
				// Use double for float decimals
				return `"double"`
			} else {
				// Use Avro logical decimal type with precision and scale
				return fmt.Sprintf(`{"type":"bytes","logicalType":"decimal","precision":%d,"scale":%d}`,
					ltc.Args.Precision, ltc.Args.Scale)
			}
		default:
			panic(fmt.Sprintf("Unknown logical type config: %T", logical_type_config))
		}
	},
}

func generateSchema(config *config.Config) string {
	t, err := template.New("schemaTemplate").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		panic(err)
	}
	var builder strings.Builder
	if err := t.Execute(&builder, config.Fields); err != nil {
		panic(err)
	}

	result := builder.String()
	return result
}

func WriteFile(filepath string, config *config.Config, channel chan map[string]any, wg *sync.WaitGroup, errTracker error_tracking.ErrorTracker) {
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
		errTracker.AddExecutionError(fmt.Errorf("error setting up avro writer: %s", err))
		return
	}

	for row := range channel {
		writer.Append([]map[string]interface{}{row})
	}

}
