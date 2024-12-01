package error_tracking

import (
	"errors"
	"fmt"
)

func (tracker *ErrorTracker) CombinedExecutionError() error {
	errMsg := ""
	for i, err := range tracker.ExecutionErrors {
		if i == 0 {
			errMsg += "Execution failed with the following errors:\n"
		}
		errMsg += fmt.Sprintf("%s \n", err)
	}

	return errors.New(errMsg)

}

func (report *ErrorReport) ConsoleFormat() string {
	errMsg := "\n\n🚨 The CSV Has Failed Validation 🚨\n\n"

	if len(report.FileErrors) > 0 {
		errMsg += "The following file-level errors were detected:\n"
		for _, fileErr := range report.FileErrors {
			errMsg += "  - " + fileErr + "\n"
		}
		errMsg += "\n\n"
	} else {
		errMsg += "The file was read from disk successfully and the header was valid and consistent with the declared schema.\n\n"
	}

	if len(report.RowErrors) > 0 {
		errMsg += "The following row level errors were detected:\n"
		for _, rowErr := range report.RowErrors {
			errMsg += "  - " + rowErr + "\n"
		}
		errMsg += "\n\n"
	} else {
		errMsg += "No malformed rows were found.\n\n"
	}

	if len(report.CellErrors) > 0 {
		errMsg += "The following cell level errors were detected:\n"
		for _, cellErr := range report.CellErrors {
			errMsg += "  - " + cellErr + "\n"
		}
	}

	return errMsg
}
