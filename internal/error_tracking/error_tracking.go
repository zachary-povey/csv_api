package error_tracking

const (
	exErrBuffer  int = 5
	repErrBuffer int = 5
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
	exErrQ          chan error
	repErrQ         chan ReportError
}

func NewErrorTracker() ErrorTracker {
	return ErrorTracker{
		exErrQ:  make(chan error, exErrBuffer),
		repErrQ: make(chan ReportError, repErrBuffer),
	}
}

func (tracker *ErrorTracker) AddExecutionError(err error) {
	tracker.exErrQ <- err
}

func (tracker *ErrorTracker) AddReportError(err string, errType ReportErrorType) {
	tracker.repErrQ <- ReportError{err, errType}
}

func (tracker *ErrorTracker) Start() {
	go func() {
		for exErr := range tracker.exErrQ {
			tracker.ExecutionErrors = append(tracker.ExecutionErrors, exErr)
		}
	}()

	go func() {
		for repErr := range tracker.repErrQ {
			switch repErr.Type {
			case File:
				tracker.ErrorReport.FileErrors = append(tracker.ErrorReport.FileErrors, repErr.Value)
			case Row:
				tracker.ErrorReport.RowErrors = append(tracker.ErrorReport.RowErrors, repErr.Value)
			case Cell:
				tracker.ErrorReport.CellErrors = append(tracker.ErrorReport.CellErrors, repErr.Value)
			}
		}
	}()
}

func (tracker *ErrorTracker) ShouldTerminate() bool {
	return len(tracker.ExecutionErrors) > 
}

func (tracker *ErrorTracker) Stop() {
	close(tracker.exErrQ)
	close(tracker.repErrQ)
}
