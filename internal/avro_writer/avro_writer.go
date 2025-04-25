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
		{"name": "{{$value.Name}}", "type": "{{map_type $value.LogicalTypeConfig.Name}}"}{{if ne $i (subtract $total 1)}},{{end}}
	{{- end }}
	]
}`

var type_mappings = map[config.LogicalType]string{
	config.Integer: "long",
	config.String:  "string",
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
