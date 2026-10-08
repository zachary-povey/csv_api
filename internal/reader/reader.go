package reader

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
)

// Record is one data row of the input file.
type Record struct {
	// line number in the input file where the row starts
	Line int
	// all values in the row, as read from the file
	Raw []string
	// values for each configured field, in config order
	Values []*string
}

type Reader struct {
	Header []string

	file           *os.File
	csvReader      *csv.Reader
	config         *config.Config
	fieldPositions map[string]*int
}

// Open opens the data file, reads the header and checks it against the
// config. The returned execution error is set when the file cannot be read at
// all; the file error is set when the header is not valid for the config.
func Open(filepath string, config *config.Config) (reader *Reader, executionErr error, fileErr error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error opening data file: %w", err), nil
	}

	csvReader := csv.NewReader(file)
	header, headerErr := csvReader.Read()
	if headerErr != nil {
		file.Close()
		return nil, nil, fmt.Errorf("error reading CSV header: %s", headerErr)
	}

	fieldPositions, fieldPosErr := getFieldPositions(config, header)
	if fieldPosErr != nil {
		file.Close()
		return nil, nil, fieldPosErr
	}

	return &Reader{
		Header:         header,
		file:           file,
		csvReader:      csvReader,
		config:         config,
		fieldPositions: fieldPositions,
	}, nil, nil
}

func (r *Reader) ReadRows(channel chan Record, wg *sync.WaitGroup, errTracker *error_tracking.ErrorTracker) {
	defer wg.Done()
	defer close(channel)
	defer r.file.Close()

	for {
		input_record, err := r.csvReader.Read()
		if err == io.EOF {
			return
		}
		if errors.Is(err, csv.ErrFieldCount) {
			// the rest of the file can still be read reliably
			line, _ := r.csvReader.FieldPos(0)
			errTracker.AddReportError(fmt.Sprintf("row %d has %d values but the header has %d", line, len(input_record), len(r.Header)), error_tracking.Row)
			if errTracker.Killed() {
				errTracker.MarkStoppedEarly("")
				return
			}
			continue
		} else if err != nil {
			// e.g. a broken quote: later rows can't be located reliably
			errTracker.AddReportError(err.Error(), error_tracking.Row)
			errTracker.MarkStoppedEarly("the file could not be parsed past this point")
			return
		}

		line, _ := r.csvReader.FieldPos(0)
		output_record := []*string{}
		for _, fieldName := range r.config.AllFieldNames() {
			fieldPosition := r.fieldPositions[fieldName]
			if fieldPosition == nil {
				// aka a missing, non-required, field
				output_record = append(output_record, nil)
			} else {
				output_record = append(output_record, &input_record[*fieldPosition])
			}
		}

		select {
		case <-errTracker.KillCh:
			errTracker.MarkStoppedEarly("")
			return
		case channel <- Record{Line: line, Raw: input_record, Values: output_record}:
			// message added
		}
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
