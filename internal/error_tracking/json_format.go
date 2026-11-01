package error_tracking

import (
	"encoding/json"
)

// JSONReport is the --error-report json layout. Every key is always present;
// values that don't apply are null.
type JSONReport struct {
	Valid        bool        `json:"valid"`
	StoppedEarly bool        `json:"stopped_early"`
	StopReason   *string     `json:"stop_reason"`
	DroppedRows  int         `json:"dropped_rows"`
	Errors       []JSONError `json:"errors"`
}

type JSONError struct {
	// file, row, data or execution
	Group   string  `json:"group"`
	Row     *int    `json:"row"`
	Column  *string `json:"column"`
	Field   *string `json:"field"`
	Value   *string `json:"value"`
	Reason  string  `json:"reason"`
	Message *string `json:"message"`
}

// JSONFormat gives the report as JSON. valid is whether the run succeeded.
func (tracker *ErrorTracker) JSONFormat(valid bool) ([]byte, error) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	report := tracker.ErrorReport

	out := JSONReport{
		Valid:        valid,
		StoppedEarly: report.StoppedEarly,
		DroppedRows:  report.DroppedRows,
		Errors:       []JSONError{},
	}
	stopReason := report.StopReason

	for _, err := range tracker.ExecutionErrors {
		out.Errors = append(out.Errors, JSONError{Group: "execution", Reason: err.Error()})
	}
	for _, fileError := range report.FileErrors {
		out.Errors = append(out.Errors, JSONError{Group: "file", Reason: fileError})
	}
	for _, rowError := range report.RowErrors {
		out.Errors = append(out.Errors, JSONError{Group: "row", Row: nonZero(rowError.Row), Reason: rowError.Reason})
	}
	for _, cellError := range report.CellErrors {
		value := cellError.Value
		out.Errors = append(out.Errors, JSONError{
			Group:   "data",
			Row:     nonZero(cellError.Row),
			Column:  nonEmpty(cellError.Header),
			Field:   nonEmpty(cellError.Field),
			Value:   &value,
			Reason:  cellError.Reason,
			Message: nonEmpty(cellError.Message),
		})
	}

	// the console report says these without a stop reason; the json report
	// states them so consumers know the file was not fully checked
	if len(tracker.ExecutionErrors) > 0 {
		out.StoppedEarly = true
		if stopReason == "" {
			stopReason = "an execution error stopped the run"
		}
	} else if len(report.FileErrors) > 0 {
		out.StoppedEarly = true
		stopReason = "the header is not valid, so no rows were checked"
	}
	if out.StoppedEarly {
		out.StopReason = nonEmpty(stopReason)
	}

	return json.MarshalIndent(out, "", "  ")
}

func nonZero(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
