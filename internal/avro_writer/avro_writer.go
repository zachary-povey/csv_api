package avro_writer

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/linkedin/goavro/v2"
	"github.com/zachary-povey/csv_api/internal/config"
)

const tmpl string = `
{
	"name": "test_schema",
	"type": "record",
	"fields": [
	{{- $total := len . }}
	{{- range $i, $value := . }}
		{"name": "{{$value.Name}}", "type": "string"}{{if ne $i (subtract $total 1)}},{{end}}
	{{- end }}
	]
}`

var funcMap template.FuncMap = template.FuncMap{
	"subtract": func(a, b int) int {
		return a - b
	},
}

func generateSchema(config *config.Config) string {
	t, err := template.New("schemaTemplate").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		panic(err)
	}

	fmt.Println(config.Fields)

	// Use a strings.Builder to get the output as a string
	var builder strings.Builder
	if err := t.Execute(&builder, config.Fields); err != nil {
		panic(err)
	}

	// Get the resulting string
	result := builder.String()
	return result
}

func WriteFile(filepath string, config *config.Config, channel chan []*string, wg *sync.WaitGroup) error {
	defer wg.Done()

	avroSchema := generateSchema(config)
	fmt.Println(avroSchema)

	output, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("error opening output file: %s", err)
	}
	defer output.Close()

	writer, writerErr := goavro.NewOCFWriter(goavro.OCFConfig{
		W:      output,
		Schema: avroSchema,
	})
	if writerErr != nil {
		panic(writerErr)
	}

	for row := range channel {
		resolvedRow := map[string]interface{}{}
		for i, value := range row {
			field := config.Fields[i]
			resolvedRow[field.Name] = *value
		}
		fmt.Println(resolvedRow)
		writer.Append([]map[string]interface{}{resolvedRow})
	}

	return nil
}
