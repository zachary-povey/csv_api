package error_tracking

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// seconds the passphrase is cached after the last use
const defaultCacheTTL = 600 // e.g. 10 minutes

// maximum lifetime even if repeatedly used
const maxCacheTTL = 3600 // e.g. 1 hour

func FormatBulletList(items []string) string {
	result := ""
	for i, item := range items {
		msg := strings.ReplaceAll(item, "\n", "\n    ")
		result += "  - " + msg
		if i < len(items)-1 {
			result += "\n"
		}
	}
	return result
}

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
	sections := []string{}

	if len(report.FileErrors) > 0 {
		// no rows are read when the file itself is invalid
		sections = append(sections,
			color.RedString("The following file-level errors were detected:\n")+FormatBulletList(report.FileErrors))
	} else {
		sections = append(sections,
			color.CyanString("The file was read from disk successfully and the header was valid and consistent with the declared schema."))

		if len(report.RowErrors) > 0 {
			sections = append(sections,
				color.RedString("The following malformed rows were detected:\n")+FormatBulletList(report.RowErrors))
		} else if !report.StoppedEarly {
			sections = append(sections, color.CyanString("No malformed rows were found."))
		}

		if len(report.CellErrors) > 0 {
			sections = append(sections,
				color.RedString("The following invalid data values were detected:\n")+FormatBulletList(report.CellErrors))
		} else if report.DataFailuresDropped {
			sections = append(sections, color.CyanString("Rows with invalid data values are dropped rather than reported (--error-on-data-failures=false)."))
		} else if !report.StoppedEarly {
			sections = append(sections, color.CyanString("No invalid data values were detected."))
		}
	}

	if report.StoppedEarly {
		sections = append(sections,
			color.YellowString("Processing stopped after %d error(s) because %s; the rest of the file was not checked.", report.count(), report.StopReason))
	}

	sections = append(sections, color.RedString("✗ The CSV has failed validation"))

	return strings.Join(sections, "\n\n")
}
