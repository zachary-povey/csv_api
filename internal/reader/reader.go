package reader

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
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
	fieldPositions []*int
}

// Open opens the data file, reads the header and checks it against the
// config. The returned execution error is set when the file cannot be read at
// all; the file errors are set when the header is not valid for the config.
func Open(filepath string, config *config.Config) (reader *Reader, executionErr error, fileErrs []error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error opening data file: %w", err), nil
	}

	csvReader := csv.NewReader(file)
	header, headerErr := csvReader.Read()
	if headerErr != nil {
		file.Close()
		return nil, nil, []error{fmt.Errorf("error reading CSV header: %s", headerErr)}
	}

	fieldPositions, fieldPosErrs := getFieldPositions(config, header)
	if len(fieldPosErrs) > 0 {
		file.Close()
		return nil, nil, fieldPosErrs
	}

	return &Reader{
		Header:         header,
		file:           file,
		csvReader:      csvReader,
		config:         config,
		fieldPositions: fieldPositions,
	}, nil, nil
}

// ColumnNames gives, in config order, the header of the column each field was
// read from ("" for a missing optional field).
func (r *Reader) ColumnNames() []string {
	names := make([]string, len(r.fieldPositions))
	for i, position := range r.fieldPositions {
		if position != nil {
			names[i] = r.Header[*position]
		}
	}
	return names
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
		for _, fieldPosition := range r.fieldPositions {
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

// getFieldPositions finds each configured field's column, in config order (nil
// for a missing optional field). It returns every header problem found: a
// header matching more than one field, a field matching more than one header,
// and missing required fields.
func getFieldPositions(conf *config.Config, header []string) ([]*int, []error) {
	errs := []error{}
	matches := make([][]int, len(conf.Fields))

	for position, columnName := range header {
		matchedFields := []string{}
		for i, field := range conf.Fields {
			if field.MatchesHeader(columnName) {
				matches[i] = append(matches[i], position)
				matchedFields = append(matchedFields, quoted(field.Name))
			}
		}
		if len(matchedFields) > 1 {
			errs = append(errs, fmt.Errorf("header %s matches more than one field: %s",
				describeColumn(header, position), strings.Join(matchedFields, ", ")))
		}
	}

	fieldPositions := make([]*int, len(conf.Fields))
	missingRequired := []string{}
	for i, field := range conf.Fields {
		switch len(matches[i]) {
		case 0:
			if field.IsRequired() {
				missingRequired = append(missingRequired, describeMissing(field))
			}
		case 1:
			fieldPositions[i] = &matches[i][0]
		default:
			columns := []string{}
			for _, position := range matches[i] {
				columns = append(columns, describeColumn(header, position))
			}
			errs = append(errs, fmt.Errorf("field '%s' matches more than one header: %s",
				field.Name, strings.Join(columns, ", ")))
		}
	}

	if len(missingRequired) > 0 {
		errs = append(errs, fmt.Errorf("input data is missing some required fields: [%s]",
			strings.Join(missingRequired, ", ")))
	}
	return fieldPositions, errs
}

// describeColumn names a column by its header text and 1-based position.
func describeColumn(header []string, position int) string {
	return fmt.Sprintf("%s (column %d)", quoted(header[position]), position+1)
}

// describeMissing names a missing field, with the header_patterns it was
// looked up by when it has them.
func describeMissing(field config.FieldConfig) string {
	if len(field.HeaderPatterns) == 0 {
		return field.Name
	}
	patterns := []string{}
	for _, pattern := range field.HeaderPatterns {
		patterns = append(patterns, quoted(pattern))
	}
	return fmt.Sprintf("%s (no header matches header_patterns %s)", field.Name, strings.Join(patterns, ", "))
}

func quoted(s string) string {
	return "'" + s + "'"
}
