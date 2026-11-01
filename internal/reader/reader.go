package reader

import (
	"bufio"
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
	// QuotedEmpty marks, in config order, the empty values that were quoted
	// in the file (,"",). It is nil when the row has none, and only set for
	// fields whose empty_is_null and quoted_empty_is_null differ (for the
	// others quoting doesn't change anything).
	QuotedEmpty []bool
}

type Reader struct {
	Header []string

	file           *os.File
	csvReader      *csv.Reader
	config         *config.Config
	fieldPositions []*int
	// quoteSensitive marks, in config order, the fields whose empty cells
	// must be classified as quoted or not
	quoteSensitive []bool
	// the bytes most recently read from the file, set only when some field
	// is quote sensitive
	tail *tailReader
}

// Open opens the data file, reads the header and checks it against the
// config. The returned execution error is set when the file cannot be read at
// all; the file errors are set when the header is not valid for the config.
func Open(filepath string, config *config.Config) (reader *Reader, executionErr error, fileErrs []error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error opening data file: %w", err), nil
	}

	quoteSensitive := make([]bool, len(config.Fields))
	anyQuoteSensitive := false
	for i, field := range config.Fields {
		quoteSensitive[i] = config.EmptyCellIsNull(field, false) != config.EmptyCellIsNull(field, true)
		anyQuoteSensitive = anyQuoteSensitive || quoteSensitive[i]
	}
	var tail *tailReader
	var csvReader *csv.Reader
	if anyQuoteSensitive {
		tail = &tailReader{r: file}
		csvReader = csv.NewReader(bufio.NewReaderSize(tail, tailBufferSize))
	} else {
		csvReader = csv.NewReader(file)
	}
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
		quoteSensitive: quoteSensitive,
		tail:           tail,
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
			errTracker.AddRowError(error_tracking.RowError{
				Row:    line,
				Reason: fmt.Sprintf("row %d has %d values but the header has %d", line, len(input_record), len(r.Header)),
			})
			if errTracker.Killed() {
				errTracker.MarkStoppedEarly("")
				return
			}
			continue
		} else if err != nil {
			// e.g. a broken quote: later rows can't be located reliably
			rowError := error_tracking.RowError{Reason: err.Error()}
			var parseErr *csv.ParseError
			if errors.As(err, &parseErr) {
				rowError.Row = parseErr.StartLine
			}
			errTracker.AddRowError(rowError)
			errTracker.MarkStoppedEarly("the file could not be parsed past this point")
			return
		}

		line, _ := r.csvReader.FieldPos(0)
		output_record := []*string{}
		var quotedEmpty []bool
		for i, fieldPosition := range r.fieldPositions {
			if fieldPosition == nil {
				// aka a missing, non-required, field
				output_record = append(output_record, nil)
				continue
			}
			output_record = append(output_record, &input_record[*fieldPosition])
			if r.quoteSensitive[i] && input_record[*fieldPosition] == "" &&
				r.emptyFieldIsQuoted(*fieldPosition, len(input_record)) {
				if quotedEmpty == nil {
					quotedEmpty = make([]bool, len(r.fieldPositions))
				}
				quotedEmpty[i] = true
			}
		}

		select {
		case <-errTracker.KillCh:
			errTracker.MarkStoppedEarly("")
			return
		case channel <- Record{Line: line, Raw: input_record, Values: output_record, QuotedEmpty: quotedEmpty}:
			// message added
		}
	}
}

// emptyFieldIsQuoted reports whether the empty field at index i of the record
// just read was written quoted ("") rather than as nothing. encoding/csv gives
// "" for both, but the raw field is 2 bytes wide when quoted and 0 otherwise.
func (r *Reader) emptyFieldIsQuoted(i int, fieldCount int) bool {
	if i < fieldCount-1 {
		// an empty field never spans lines, so the next field starts on the
		// same line, after the raw field and the comma (columns are bytes)
		_, start := r.csvReader.FieldPos(i)
		_, next := r.csvReader.FieldPos(i + 1)
		return next-start == 3
	}
	// the last field ends at the record's line terminator ("\n", "\r\n", or
	// nothing / a dropped "\r" at end of file): a quoted field ends with a
	// quote, an unquoted empty one right after the comma
	end := r.csvReader.InputOffset()
	last, ok := r.tail.byteAt(end - 1)
	if ok && last == '\n' {
		end--
		last, ok = r.tail.byteAt(end - 1)
	}
	if ok && last == '\r' {
		last, ok = r.tail.byteAt(end - 2)
	}
	return ok && last == '"'
}

// tailBufferSize is the read buffer between the file and the csv reader when
// quoting is tracked: it bounds how far reading runs ahead of the csv reader.
const tailBufferSize = 64 * 1024

// tailKeep is how many bytes tailReader keeps: the csv reader is at most
// tailBufferSize (unread buffered bytes) behind the file, and the end of a
// record is looked up up to 3 bytes before the csv reader's position.
const tailKeep = tailBufferSize + 4

// tailReader passes reads through, keeping at least the last tailKeep bytes
// read so the raw bytes at the end of the record the csv reader just read can
// be looked up by offset.
type tailReader struct {
	r io.Reader
	// the last bytes read, ending at offset
	tail   []byte
	offset int64
}

func (t *tailReader) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	if len(t.tail)+n > 2*tailKeep {
		// drop the oldest bytes, keeping tailKeep before this read
		t.tail = append(t.tail[:0], t.tail[max(len(t.tail)-tailKeep, 0):]...)
	}
	t.tail = append(t.tail, p[:n]...)
	t.offset += int64(n)
	return n, err
}

// byteAt gives the byte at offset in the file, if it is still kept.
func (t *tailReader) byteAt(offset int64) (byte, bool) {
	i := int64(len(t.tail)) - (t.offset - offset)
	if offset < 0 || i < 0 || i >= int64(len(t.tail)) {
		return 0, false
	}
	return t.tail[i], true
}

// getFieldPositions finds each configured field's column, in config order (nil
// for a missing optional field). It returns every header problem found: a
// header matching more than one field, a field matching more than one header,
// missing required fields and, with allow_extra_fields: false, columns that
// match no field.
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
		if len(matchedFields) == 0 && !conf.ExtraFieldsAllowed() {
			errs = append(errs, fmt.Errorf("header %s matches no field (allow_extra_fields is false)",
				describeColumn(header, position)))
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
// looked up by when it has them, and its error_message.
func describeMissing(field config.FieldConfig) string {
	description := field.Name
	if len(field.HeaderPatterns) > 0 {
		patterns := []string{}
		for _, pattern := range field.HeaderPatterns {
			patterns = append(patterns, quoted(pattern))
		}
		description = fmt.Sprintf("%s (no header matches header_patterns %s)", field.Name, strings.Join(patterns, ", "))
	}
	return error_tracking.WithFieldMessage(description, field.ErrorMessage)
}

func quoted(s string) string {
	return "'" + s + "'"
}
