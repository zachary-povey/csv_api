package options

import (
	"errors"
	"fmt"
)

const (
	ConsoleErrorReport = "console"
	JSONErrorReport    = "json"

	AvroOutputFormat    = "avro"
	ParquetOutputFormat = "parquet"
)

// Options control how the parse command writes output and treats errors.
// They are run-time choices of the consumer, so they come from command line
// flags rather than the config file.
type Options struct {
	// where the output is written; with Stream, empty means stdout
	OutputPath string
	// the output file format: avro or parquet
	OutputFormat string
	// write output as rows are validated rather than all or nothing; a failed
	// run leaves incomplete output
	Stream bool
	// if false, rows with invalid data values are dropped instead of failing the run
	ErrorOnDataFailures bool
	// stop at the first error rather than collecting errors
	FailFast bool
	// maximum number of errors to collect when not failing fast, 0 means no limit
	MaxErrors int
	// optional path for a csv of the rows dropped for invalid data
	DataFailureReport string
	// where collected errors are reported: console, or json (as well as console)
	ErrorReport string
	// with the json error report, the file it is written to
	ErrorReportPath string
}

func (opts Options) Validate() error {
	var errs []error
	if opts.OutputPath == "" && !opts.Stream {
		errs = append(errs, errors.New("--output-path is required (it can only be left out with --stream, to write to stdout)"))
	}
	switch opts.OutputFormat {
	case AvroOutputFormat:
	case ParquetOutputFormat:
	default:
		errs = append(errs, fmt.Errorf("unsupported --output-format '%s', supported values are '%s' and '%s'", opts.OutputFormat, AvroOutputFormat, ParquetOutputFormat))
	}
	if opts.MaxErrors < 0 {
		errs = append(errs, errors.New("--max-errors must be a positive number"))
	}
	if opts.MaxErrors > 0 && opts.FailFast {
		errs = append(errs, errors.New("--max-errors can only be set with --fail-fast=false"))
	}
	if opts.DataFailureReport != "" && opts.ErrorOnDataFailures {
		errs = append(errs, errors.New("--data-failure-report can only be set with --error-on-data-failures=false"))
	}
	switch opts.ErrorReport {
	case ConsoleErrorReport:
		if opts.ErrorReportPath != "" {
			errs = append(errs, errors.New("--error-report-path can only be set with --error-report=json"))
		}
	case JSONErrorReport:
		if opts.ErrorReportPath == "" {
			errs = append(errs, errors.New("--error-report=json requires --error-report-path"))
		}
	default:
		errs = append(errs, fmt.Errorf("unsupported --error-report '%s', supported values are '%s' and '%s'", opts.ErrorReport, ConsoleErrorReport, JSONErrorReport))
	}
	return errors.Join(errs...)
}

// OutputName describes where the output goes, for messages.
func (opts Options) OutputName() string {
	if opts.OutputPath == "" {
		return "stdout"
	}
	return fmt.Sprintf("'%s'", opts.OutputPath)
}
