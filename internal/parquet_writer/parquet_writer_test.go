package parquet_writer

import (
	"bytes"
	"sync"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
)

// recordingWriter is a non-seekable io.Writer (like stdout or a pipe) that
// keeps what was written, safe to read while Write runs.
type recordingWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *recordingWriter) bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return bytes.Clone(w.buf.Bytes())
}

var testConfig = &config.Config{
	Name:   "test",
	Fields: []config.FieldConfig{{Name: "id", LogicalTypeConfig: config.IntegerTypeConfig{}}},
}

// startWrite runs Write with row groups of rowGroupSize rows on an
// unbuffered channel, so a send returns only once Write has finished with
// the previous row.
func startWrite(t *testing.T, rowGroupSize int, tracker *error_tracking.ErrorTracker) (*recordingWriter, chan map[string]any, *sync.WaitGroup) {
	previous := rowsPerRowGroup
	rowsPerRowGroup = rowGroupSize
	t.Cleanup(func() { rowsPerRowGroup = previous })
	output := &recordingWriter{}
	channel := make(chan map[string]any)
	var wg sync.WaitGroup
	wg.Add(1)
	go Write(output, testConfig, channel, &wg, tracker)
	return output, channel, &wg
}

func TestRowGroupsReachOutputBeforeInputEnds(t *testing.T) {
	// TASK-24 AC#4: with an input larger than one row group, each completed
	// row group is written to the (non-seekable) output before the input has
	// finished, rather than the whole file being written at the end.
	tracker := error_tracking.NewErrorTracker(true, 0)
	output, channel, wg := startWrite(t, 2, tracker)
	for i := 1; i <= 3; i++ {
		channel <- map[string]any{"id": i}
	}
	// the third row was received, so the first row group (rows 1-2) has been
	// flushed; the channel is still open
	beforeEnd := output.bytes()
	close(channel)
	wg.Wait()
	if tracker.Failed() {
		t.Fatalf("unexpected errors: %v", tracker.ExecutionErrors)
	}

	file := output.bytes()
	if !bytes.HasPrefix(file, beforeEnd) {
		t.Fatal("bytes written before the input ended are not a prefix of the file")
	}
	parquetFile, err := parquet.OpenFile(bytes.NewReader(file), int64(len(file)))
	if err != nil {
		t.Fatalf("output is not a readable Parquet file: %v", err)
	}
	rowGroups := parquetFile.Metadata().RowGroups
	if len(rowGroups) != 2 || rowGroups[0].NumRows != 2 || rowGroups[1].NumRows != 1 {
		t.Fatalf("expected row groups of 2 and 1 rows, got %+v", rowGroups)
	}
	chunk := rowGroups[0].Columns[0].MetaData
	firstGroupEnd := chunk.DataPageOffset + chunk.TotalCompressedSize
	if chunk.DictionaryPageOffset > 0 && chunk.DictionaryPageOffset < chunk.DataPageOffset {
		firstGroupEnd = chunk.DictionaryPageOffset + chunk.TotalCompressedSize
	}
	if int64(len(beforeEnd)) < firstGroupEnd {
		t.Fatalf("only %d bytes were written before the input ended, the first row group ends at %d", len(beforeEnd), firstGroupEnd)
	}
}

func TestFailedRunWritesNoFooter(t *testing.T) {
	// TASK-24 AC#2: when the run has failed by the time the input ends, no
	// footer is written: the row groups already written stay incomplete
	// output that Parquet readers reject.
	tracker := error_tracking.NewErrorTracker(true, 0)
	output, channel, wg := startWrite(t, 2, tracker)
	for i := 1; i <= 3; i++ {
		channel <- map[string]any{"id": i}
	}
	tracker.AddCellError(error_tracking.CellError{Row: 5, Column: "id", Field: "id", Value: "x", Reason: "invalid"})
	close(channel)
	wg.Wait()

	file := output.bytes()
	if !bytes.HasPrefix(file, []byte("PAR1")) || len(file) <= 4 {
		t.Fatalf("expected the first row group to have been written, got %d bytes", len(file))
	}
	if bytes.HasSuffix(file, []byte("PAR1")) {
		t.Fatal("a footer was written for a failed run")
	}
	if _, err := parquet.OpenFile(bytes.NewReader(file), int64(len(file))); err == nil {
		t.Fatal("the output of a failed run is readable as Parquet")
	}
}
