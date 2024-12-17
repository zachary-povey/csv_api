package error_tracking

import (
	"sync"
)

const (
	exErrBuffer  int = 5
	repErrBuffer int = 5
	killChBuffer int = 5
	maxErrors    int = 10
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
}

type ReportError struct {
	Value string
	Type  ReportErrorType
}

type ErrorTracker struct {
	ExecutionErrors []error
	ErrorReport     ErrorReport
	KillCh          chan struct{}
	waitGroup       *sync.WaitGroup
	exErrQ          chan error
	repErrQ         chan ReportError
}

func NewErrorTracker() ErrorTracker {
	tracker := ErrorTracker{
		exErrQ:  make(chan error, exErrBuffer),
		repErrQ: make(chan ReportError, repErrBuffer),
		KillCh:  make(chan struct{}, killChBuffer),
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	tracker.waitGroup = &waitGroup

	return tracker
}

func (tracker *ErrorTracker) AddExecutionError(err error) {
	tracker.exErrQ <- err
}

func (tracker *ErrorTracker) AddReportError(err string, errType ReportErrorType) {
	tracker.repErrQ <- ReportError{err, errType}
}

func (tracker *ErrorTracker) Start() {
	killOnce := sync.OnceFunc(func() {
		close(tracker.KillCh)
	})

	go func() {
		for exErr := range tracker.exErrQ {
			tracker.ExecutionErrors = append(tracker.ExecutionErrors, exErr)
			killOnce()
		}
		tracker.waitGroup.Done()
	}()

	go func() {
		count := 0
		for repErr := range tracker.repErrQ {
			switch repErr.Type {
			case File:
				tracker.ErrorReport.FileErrors = append(tracker.ErrorReport.FileErrors, repErr.Value)
			case Row:
				tracker.ErrorReport.RowErrors = append(tracker.ErrorReport.RowErrors, repErr.Value)
			case Cell:
				tracker.ErrorReport.CellErrors = append(tracker.ErrorReport.CellErrors, repErr.Value)
			}

			count += 1
			if count > maxErrors {
				killOnce()
				break
			}
		}
		tracker.waitGroup.Done()
	}()
}

func (tracker *ErrorTracker) Stop() {
	close(tracker.exErrQ)
	close(tracker.repErrQ)
	tracker.waitGroup.Wait()
}
