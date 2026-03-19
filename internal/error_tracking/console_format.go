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
	errMsg := ""

	if len(report.FileErrors) > 0 {
		errMsg += color.RedString("The following file-level errors were detected:\n")
		errMsg += FormatBulletList(report.FileErrors) + "\n\n"
	} else {
		errMsg += color.CyanString("The file was read from disk successfully and the header was valid and consistent with the declared schema.\n")
	}

	if len(report.RowErrors) > 0 {
		errMsg += color.RedString("The following row level errors were detected:\n")
		errMsg += FormatBulletList(report.RowErrors) + "\n\n"
	} else {
		errMsg += color.CyanString("No malformed rows were found.\n")
	}

	if len(report.CellErrors) > 0 {
		errMsg += color.RedString("The following cell level errors were detected:\n")
		errMsg += FormatBulletList(report.CellErrors) + "\n"
	} else {
		errMsg += color.CyanString("No cell level errors were detected.")
	}

	errMsg += color.RedString("\n✗ The CSV has failed validation")

	return errMsg
}
