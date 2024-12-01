package reader

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
)

func ReadFile(filepath string, config *config.Config, channel chan []*string, wg *sync.WaitGroup, errTracker error_tracking.ErrorTracker) {
	defer wg.Done()
	defer close(channel)

	file, err := os.Open(filepath)
	if err != nil {
		errTracker.AddExecutionError(fmt.Errorf("error opening file: %s", err))
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)

	header, headerErr := reader.Read()

	if headerErr != nil {
		errTracker.AddExecutionError(fmt.Errorf("error reading CSV header: %s", err))
		return
	}

	fieldPositions, fieldPosErr := getFieldPositions(config, header)
	if fieldPosErr != nil {
		errTracker.AddExecutionError(fieldPosErr)
		return
	}

	for {
		input_record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			errTracker.AddExecutionError(fmt.Errorf("error reading CSV: %s", err))
			return
		}
		output_record := []*string{}

		for _, fieldName := range config.AllFieldNames() {
			fieldPosition := fieldPositions[fieldName]
			if fieldPosition == nil {
				// aka a missing, non-required, field
				output_record = append(output_record, nil)
			} else {
				output_record = append(output_record, &input_record[*fieldPosition])
			}
		}

		channel <- output_record
	}

}

func getFieldPositions(config *config.Config, header []string) (map[string]*int, error) {
	fieldsFound := []string{}
	fieldPositions := map[string]*int{}
	fieldMap := config.FieldMap()

	for position, column_name := range header {
		if _, exists := fieldMap[column_name]; exists {
			fieldsFound = append(fieldsFound, column_name)
			fieldPositions[column_name] = &position
		}
	}

	missingRequiredFields := missingEntries(fieldsFound, config.RequiredFieldNames())
	if len(missingRequiredFields) > 0 {
		return nil, fmt.Errorf("input data is missing some required fields: %s", missingRequiredFields)
	}

	return fieldPositions, nil
}

func missingEntries(s []string, entries []string) []string {
	entrySet := make(map[string]struct{}, len(s))
	for _, entry := range s {
		entrySet[entry] = struct{}{} // used as a dummy value
	}

	missingEntries := []string{}
	for _, entry := range entries {
		if _, found := entrySet[entry]; !found {
			missingEntries = append(missingEntries, entry)
		}
	}
	return missingEntries
}
