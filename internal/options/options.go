package options

import (
	"errors"
	"fmt"
)

const ConsoleErrorReport = "console"

// Options control how the parse command treats errors. They are run-time
// choices of the consumer, so they come from command line flags rather than
// the config file.
type Options struct {
	// if false, rows with invalid data values are dropped instead of failing the run
	ErrorOnDataFailures bool
	// stop at the first error rather than collecting errors
	FailFast bool
	// maximum number of errors to collect when not failing fast, 0 means no limit
	MaxErrors int
	// optional path for a csv of the rows dropped for invalid data
	DataFailureReport string
	// where collected errors are reported
	ErrorReport string
}

func (opts Options) Validate() error {
	var errs []error
	if opts.MaxErrors < 0 {
		errs = append(errs, errors.New("--max-errors must be a positive number"))
	}
	if opts.MaxErrors > 0 && opts.FailFast {
		errs = append(errs, errors.New("--max-errors can only be set with --fail-fast=false"))
	}
	if opts.DataFailureReport != "" && opts.ErrorOnDataFailures {
		errs = append(errs, errors.New("--data-failure-report can only be set with --error-on-data-failures=false"))
	}
	if opts.ErrorReport != ConsoleErrorReport {
		errs = append(errs, fmt.Errorf("unsupported --error-report '%s', the only supported value is '%s'", opts.ErrorReport, ConsoleErrorReport))
	}
	return errors.Join(errs...)
}
