package error_tracking

import (
	"fmt"
	"sync"
)

type ReportErrorType string

const (
	File ReportErrorType = "file"
	Row  ReportErrorType = "row"
)

// CellError is a data failure: a value that matched no representation of
// its field, or failed conversion.
type CellError struct {
	// 1-based line of the row in the file
	Row int
	// how the column is named in errors, e.g. "column 'id'"
	Column string
	// the field the column was read for
	Field string
	// the raw value from the file
	Value string
	// the technical reason, e.g. "value 'x' did not match any pattern"
	Reason string
	// the field's error_message from the config, if any
	Message string
}

// Text describes the failure without its row: column, reason and, when the
// field has one, its error_message.
func (cellError CellError) Text() string {
	return WithFieldMessage(fmt.Sprintf("%s: %s", cellError.Column, cellError.Reason), cellError.Message)
}

// WithFieldMessage adds a field's error_message, if it has one, after the
// technical description of an error about the field.
func WithFieldMessage(text string, message string) string {
	if message == "" {
		return text
	}
	return fmt.Sprintf("%s (note: %s)", text, message)
}

// String describes the failure with its row, as shown in the console report.
func (cellError CellError) String() string {
	return fmt.Sprintf("row %d, %s", cellError.Row, cellError.Text())
}

type ErrorReport struct {
	FileErrors []string
	RowErrors  []string
	CellErrors []CellError
	// set when collection stopped before the whole file was checked
	StoppedEarly bool
	// why processing was stopped, e.g. fail-fast
	StopReason string
	// set when invalid data values drop rows rather than being reported
	DataFailuresDropped bool
}

func (report ErrorReport) ContainsErrors() bool {
	return (len(report.FileErrors) + len(report.RowErrors) + len(report.CellErrors)) > 0
}

func (report ErrorReport) count() int {
	return len(report.FileErrors) + len(report.RowErrors) + len(report.CellErrors)
}

// ErrorTracker collects errors from the pipeline goroutines and closes KillCh
// once processing should stop: on any execution error, on the first report
// error when failing fast, or once maxErrors report errors have been collected.
type ErrorTracker struct {
	ExecutionErrors []error
	ErrorReport     ErrorReport
	KillCh          chan struct{}

	failFast  bool
	maxErrors int // 0 means no limit
	mu        sync.Mutex
	killOnce  sync.Once
}

func NewErrorTracker(failFast bool, maxErrors int) *ErrorTracker {
	return &ErrorTracker{
		KillCh:    make(chan struct{}),
		failFast:  failFast,
		maxErrors: maxErrors,
	}
}

func (tracker *ErrorTracker) kill() {
	tracker.killOnce.Do(func() { close(tracker.KillCh) })
}

// Killed reports whether processing has been told to stop.
func (tracker *ErrorTracker) Killed() bool {
	select {
	case <-tracker.KillCh:
		return true
	default:
		return false
	}
}

func (tracker *ErrorTracker) AddExecutionError(err error) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.ExecutionErrors = append(tracker.ExecutionErrors, err)
	tracker.kill()
}

// AddCellError records a data failure.
func (tracker *ErrorTracker) AddCellError(cellError CellError) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if !tracker.underLimit() {
		return
	}
	tracker.ErrorReport.CellErrors = append(tracker.ErrorReport.CellErrors, cellError)
	tracker.checkLimit()
}

// AddReportError records a file or row error.
func (tracker *ErrorTracker) AddReportError(err string, errType ReportErrorType) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if !tracker.underLimit() {
		return
	}

	switch errType {
	case File:
		tracker.ErrorReport.FileErrors = append(tracker.ErrorReport.FileErrors, err)
	case Row:
		tracker.ErrorReport.RowErrors = append(tracker.ErrorReport.RowErrors, err)
	}
	tracker.checkLimit()
}

// limit is the number of report errors to collect before stopping (0 means
// no limit).
func (tracker *ErrorTracker) limit() int {
	if tracker.failFast {
		return 1
	}
	return tracker.maxErrors
}

// underLimit reports whether another report error may be collected; if not,
// it was in flight when the limit was hit and is discarded. The caller holds mu.
func (tracker *ErrorTracker) underLimit() bool {
	limit := tracker.limit()
	if limit > 0 && tracker.ErrorReport.count() >= limit {
		tracker.ErrorReport.StoppedEarly = true
		return false
	}
	return true
}

// checkLimit stops processing once the limit is reached. The caller holds mu.
func (tracker *ErrorTracker) checkLimit() {
	limit := tracker.limit()
	if limit > 0 && tracker.ErrorReport.count() >= limit {
		if tracker.failFast {
			tracker.ErrorReport.StopReason = "fail-fast is on"
		} else {
			tracker.ErrorReport.StopReason = fmt.Sprintf("--max-errors (%d) was reached", tracker.maxErrors)
		}
		tracker.kill()
	}
}

// MarkStoppedEarly records that the file was not read to the end. The reason
// is only used if no reason has been recorded already.
func (tracker *ErrorTracker) MarkStoppedEarly(reason string) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.ErrorReport.StoppedEarly = true
	if tracker.ErrorReport.StopReason == "" {
		tracker.ErrorReport.StopReason = reason
	}
}
