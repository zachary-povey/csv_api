package error_tracking

import (
	"fmt"
	"sync"
)

type ReportErrorType string

const (
	File ReportErrorType = "file"
	Row  ReportErrorType = "row"
	Cell ReportErrorType = "cell"
)

type ErrorReport struct {
	FileErrors []string
	RowErrors  []string
	CellErrors []string
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

func (tracker *ErrorTracker) AddReportError(err string, errType ReportErrorType) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	limit := tracker.maxErrors
	if tracker.failFast {
		limit = 1
	}
	if limit > 0 && tracker.ErrorReport.count() >= limit {
		// rows already in flight when the limit was hit
		tracker.ErrorReport.StoppedEarly = true
		return
	}

	switch errType {
	case File:
		tracker.ErrorReport.FileErrors = append(tracker.ErrorReport.FileErrors, err)
	case Row:
		tracker.ErrorReport.RowErrors = append(tracker.ErrorReport.RowErrors, err)
	case Cell:
		tracker.ErrorReport.CellErrors = append(tracker.ErrorReport.CellErrors, err)
	}

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
