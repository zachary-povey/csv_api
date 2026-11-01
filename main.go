package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"

	"github.com/zachary-povey/csv_api/internal/avro_writer"
	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
	"github.com/zachary-povey/csv_api/internal/options"
	"github.com/zachary-povey/csv_api/internal/parquet_writer"
	"github.com/zachary-povey/csv_api/internal/parser"
	"github.com/zachary-povey/csv_api/internal/reader"
)

const (
	queueBuffer int = 10
)

func main() {

	app := &cli.App{
		Name: "csv-api",
		// help and usage errors go to stderr: stdout carries only output data
		Writer: os.Stderr,
		Commands: []*cli.Command{
			{
				Name:        "parse",
				Usage:       "Validates a csv file and converts it to avro or parquet.",
				Description: "Validates a csv file and converts it to avro or parquet.",
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:     "config-path",
						Aliases:  []string{"c"},
						Usage:    "config file path",
						Required: true,
					},

					&cli.PathFlag{
						Name:     "input-path",
						Aliases:  []string{"i"},
						Usage:    "data file path",
						Required: true,
					},

					&cli.PathFlag{
						Name:    "output-path",
						Aliases: []string{"o"},
						Usage:   "output file path; with --stream it can be left out to write to stdout",
					},

					&cli.StringFlag{
						Name:  "output-format",
						Usage: "the output file format: 'avro' or 'parquet'",
						Value: options.AvroOutputFormat,
					},

					&cli.BoolFlag{
						Name:  "stream",
						Usage: "write output as rows are validated instead of only on success; a failed run leaves incomplete output, signalled by the non-zero exit code",
					},

					&cli.BoolFlag{
						Name:  "error-on-data-failures",
						Usage: "fail the run on invalid data values; if false, rows with invalid values are dropped",
						Value: true,
					},

					&cli.BoolFlag{
						Name:  "fail-fast",
						Usage: "stop at the first error; if false, collect errors before failing",
						Value: true,
					},

					&cli.IntFlag{
						Name:  "max-errors",
						Usage: "with --fail-fast=false, stop after collecting this many errors (default: no limit)",
					},

					&cli.PathFlag{
						Name:  "data-failure-report",
						Usage: "with --error-on-data-failures=false, write the dropped rows and the reasons to this csv file",
					},

					&cli.StringFlag{
						Name:  "error-report",
						Usage: "where to report errors: 'console', or 'json' to also write a json report to --error-report-path",
						Value: options.ConsoleErrorReport,
					},

					&cli.PathFlag{
						Name:  "error-report-path",
						Usage: "with --error-report=json, the file the json error report is written to",
					},
				},
				Action: func(cCtx *cli.Context) error {
					opts := options.Options{
						OutputPath:          cCtx.Path("output-path"),
						OutputFormat:        cCtx.String("output-format"),
						Stream:              cCtx.Bool("stream"),
						ErrorOnDataFailures: cCtx.Bool("error-on-data-failures"),
						FailFast:            cCtx.Bool("fail-fast"),
						MaxErrors:           cCtx.Int("max-errors"),
						DataFailureReport:   cCtx.Path("data-failure-report"),
						ErrorReport:         cCtx.String("error-report"),
						ErrorReportPath:     cCtx.Path("error-report-path"),
					}
					if err := opts.Validate(); err != nil {
						return err
					}
					errorTracker := error_tracking.NewErrorTracker(opts.FailFast, opts.MaxErrors)
					errorTracker.ErrorReport.DataFailuresDropped = !opts.ErrorOnDataFailures
					err := parse(cCtx.Path("config-path"), cCtx.Path("input-path"), opts, errorTracker)
					if err != nil && opts.Stream {
						err = fmt.Errorf("%w\n%s", err, color.RedString("✗ --stream: the output written to %s is incomplete and must not be used", opts.OutputName()))
					}
					if opts.ErrorReport == options.JSONErrorReport {
						if reportErr := writeJSONReport(opts.ErrorReportPath, errorTracker, err); reportErr != nil {
							return errors.Join(err, reportErr)
						}
					}
					return err
				},
			},
			{
				Name:        "validate_config",
				Usage:       "Validates a csv_api config file.",
				Description: "Validates a csv_api config file.",
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:     "config-path",
						Aliases:  []string{"c"},
						Usage:    "config file path",
						Required: true,
					},
					&cli.BoolFlag{
						Name:  "print-merged",
						Usage: "print the config, merged with any configs it extends, as YAML to stdout",
					},
				},
				Action: func(cCtx *cli.Context) error {
					conf, err := config.LoadConfig(cCtx.Path("config-path"))
					if err != nil {
						return fmt.Errorf("%w\n%s", err, color.RedString("✗ config file is not valid"))
					}
					if !cCtx.Bool("print-merged") {
						color.Cyan("✔ config file is valid")
						return nil
					}
					// stdout carries only the merged config
					merged, err := yaml.Marshal(conf)
					if err != nil {
						return fmt.Errorf("failed to print merged config: %w", err)
					}
					fmt.Fprintln(os.Stderr, color.CyanString("✔ config file is valid"))
					_, err = os.Stdout.Write(merged)
					return err
				},
			},
		},
	}

	app.ExitErrHandler = func(c *cli.Context, err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// tempFileFor creates an empty temporary file next to path, so it can be
// renamed into place once the run has succeeded.
func tempFileFor(path string) (*os.File, error) {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return nil, err
	}
	// CreateTemp makes the file private; use the usual permissions for output
	if err := file.Chmod(0644); err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, err
	}
	return file, nil
}

// writeJSONReport writes the json error report for a run that ended with
// runErr (nil on success). It is written whether or not the run succeeded,
// atomically, so consumers never see a partial report.
func writeJSONReport(path string, errorTracker *error_tracking.ErrorTracker, runErr error) error {
	if runErr != nil && len(errorTracker.ExecutionErrors) == 0 && !errorTracker.ErrorReport.ContainsErrors() {
		// a failure before or after the pipeline (e.g. the config or input
		// can't be read, the output can't be moved into place) isn't tracked
		errorTracker.ExecutionErrors = append(errorTracker.ExecutionErrors, runErr)
	}
	content, err := errorTracker.JSONFormat(runErr == nil)
	if err != nil {
		return fmt.Errorf("error writing error report: %w", err)
	}
	file, err := tempFileFor(path)
	if err != nil {
		return fmt.Errorf("error creating error report '%s': %w", path, err)
	}
	defer os.Remove(file.Name()) // no-op once renamed
	_, err = file.Write(append(content, '\n'))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(file.Name(), path)
	}
	if err != nil {
		return fmt.Errorf("error writing error report '%s': %w", path, err)
	}
	return nil
}

// openOutput opens the output. Without --stream it is a temporary file,
// moved into place by commit once the run has succeeded. With --stream it is
// the output itself, written as rows are validated: stdout, or the output
// path written in place; commit does nothing.
func openOutput(opts options.Options) (output io.Writer, closeOutput func() error, commit func() error, tempPath string, err error) {
	noop := func() error { return nil }
	if opts.Stream && opts.OutputPath == "" {
		// hide stdout's *os.File type from goavro, which would otherwise try to
		// append to an existing OCF when stdout is redirected to a non-empty
		// file. Writes go straight through, unbuffered.
		return struct{ io.Writer }{os.Stdout}, noop, noop, "", nil
	}
	if opts.Stream {
		file, err := os.Create(opts.OutputPath)
		if err != nil {
			return nil, nil, nil, "", fmt.Errorf("error creating output file '%s': %w", opts.OutputPath, err)
		}
		return file, file.Close, noop, "", nil
	}
	file, err := tempFileFor(opts.OutputPath)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("error creating output file '%s': %w", opts.OutputPath, err)
	}
	commit = func() error {
		if err := os.Rename(file.Name(), opts.OutputPath); err != nil {
			return fmt.Errorf("error writing output file: %w", err)
		}
		return nil
	}
	return file, file.Close, commit, file.Name(), nil
}

func parse(configPath string, inputPath string, opts options.Options, errorTracker *error_tracking.ErrorTracker) error {
	config, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	dataReader, executionErr, fileErrs := reader.Open(inputPath, config)
	if executionErr != nil {
		return executionErr
	}
	if len(fileErrs) > 0 {
		// the header is checked in one go, so report every problem with it
		// (even with fail-fast on): no rows have been read yet
		for _, fileErr := range fileErrs {
			errorTracker.ErrorReport.FileErrors = append(errorTracker.ErrorReport.FileErrors, fileErr.Error())
		}
		return errors.New(errorTracker.ErrorReport.ConsoleFormat())
	}

	// outputs are written to temporary files and only moved into place if the
	// run succeeds, so a failed run never leaves output behind (except the
	// output with --stream, see openOutput)
	tempPaths := []string{}
	committed := false
	defer func() {
		if !committed {
			for _, path := range tempPaths {
				os.Remove(path)
			}
		}
	}()

	var dataFailureFile *os.File
	var dataFailureWriter *csv.Writer
	if opts.DataFailureReport != "" {
		dataFailureFile, err = tempFileFor(opts.DataFailureReport)
		if err != nil {
			return fmt.Errorf("error creating data failure report '%s': %w", opts.DataFailureReport, err)
		}
		defer dataFailureFile.Close()
		tempPaths = append(tempPaths, dataFailureFile.Name())
		dataFailureWriter = csv.NewWriter(dataFailureFile)
		if err := dataFailureWriter.Write(append(append([]string{}, dataReader.Header...), parser.DataFailureErrorColumn)); err != nil {
			return fmt.Errorf("error writing data failure report: %w", err)
		}
	}

	dataParser, err := parser.NewParser(config, opts, dataFailureWriter, dataReader.ColumnNames())
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	output, closeOutput, commitOutput, outputTempPath, err := openOutput(opts)
	if err != nil {
		return err
	}
	if outputTempPath != "" {
		tempPaths = append(tempPaths, outputTempPath)
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(3)
	inputChan := make(chan reader.Record, queueBuffer)
	outputChan := make(chan map[string]any, queueBuffer)

	go dataReader.ReadRows(inputChan, &waitGroup, errorTracker)
	go dataParser.ParseData(inputChan, outputChan, &waitGroup, errorTracker)
	writeOutput := avro_writer.Write
	if opts.OutputFormat == options.ParquetOutputFormat {
		writeOutput = parquet_writer.Write
	}
	go writeOutput(output, config, outputChan, &waitGroup, errorTracker)

	waitGroup.Wait()
	if err := closeOutput(); err != nil {
		errorTracker.AddExecutionError(fmt.Errorf("error writing output file: %w", err))
	}
	errorTracker.ErrorReport.DroppedRows = dataParser.DroppedRows

	if dataFailureWriter != nil {
		dataFailureWriter.Flush()
		if err := dataFailureWriter.Error(); err != nil {
			errorTracker.AddExecutionError(fmt.Errorf("error writing data failure report: %w", err))
		}
	}

	if len(errorTracker.ExecutionErrors) > 0 {
		return errorTracker.CombinedExecutionError()
	} else if errorTracker.ErrorReport.ContainsErrors() {
		return errors.New(errorTracker.ErrorReport.ConsoleFormat())
	}

	if err := commitOutput(); err != nil {
		return err
	}
	if dataFailureFile != nil {
		if err := os.Rename(dataFailureFile.Name(), opts.DataFailureReport); err != nil {
			return fmt.Errorf("error writing data failure report: %w", err)
		}
	}
	committed = true

	if dataParser.DroppedRows > 0 {
		msg := fmt.Sprintf("warning: dropped %d row(s) with invalid data", dataParser.DroppedRows)
		if opts.DataFailureReport != "" {
			msg += fmt.Sprintf(", see %s", opts.DataFailureReport)
		}
		fmt.Fprintln(os.Stderr, color.YellowString(msg))
	}
	return nil
}
