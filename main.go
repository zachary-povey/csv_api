package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	"github.com/zachary-povey/csv_api/internal/avro_writer"
	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
	"github.com/zachary-povey/csv_api/internal/options"
	"github.com/zachary-povey/csv_api/internal/parser"
	"github.com/zachary-povey/csv_api/internal/reader"
)

const (
	queueBuffer int = 10
)

func main() {

	app := &cli.App{
		Name: "csv-api",
		Commands: []*cli.Command{
			{
				Name:        "parse",
				Usage:       "Validates a csv file and extracts it into avro.",
				Description: "Validates a csv file and extracts it into avro.",
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
						Name:     "output-path",
						Aliases:  []string{"o"},
						Usage:    "output file path",
						Required: true,
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
						Usage: "where to report errors (only 'console' is supported)",
						Value: options.ConsoleErrorReport,
					},
				},
				Action: func(cCtx *cli.Context) error {
					opts := options.Options{
						ErrorOnDataFailures: cCtx.Bool("error-on-data-failures"),
						FailFast:            cCtx.Bool("fail-fast"),
						MaxErrors:           cCtx.Int("max-errors"),
						DataFailureReport:   cCtx.Path("data-failure-report"),
						ErrorReport:         cCtx.String("error-report"),
					}
					if err := opts.Validate(); err != nil {
						return err
					}
					return parse(cCtx.Path("config-path"), cCtx.Path("input-path"), cCtx.Path("output-path"), opts)
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
				},
				Action: func(cCtx *cli.Context) error {
					_, err := config.LoadConfig(cCtx.Path("config-path"))
					if err != nil {
						return fmt.Errorf("%w\n%s", err, color.RedString("✗ config file is not valid"))
					}
					color.Cyan("✔ config file is valid")
					return nil
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

func parse(configPath string, inputPath string, outputPath string, opts options.Options) error {
	config, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	errorTracker := error_tracking.NewErrorTracker(opts.FailFast, opts.MaxErrors)
	errorTracker.ErrorReport.DataFailuresDropped = !opts.ErrorOnDataFailures

	dataReader, executionErr, fileErr := reader.Open(inputPath, config)
	if executionErr != nil {
		return executionErr
	}
	if fileErr != nil {
		errorTracker.AddReportError(fileErr.Error(), error_tracking.File)
		return errors.New(errorTracker.ErrorReport.ConsoleFormat())
	}

	// outputs are written to temporary files and only moved into place if the
	// run succeeds, so a failed run never leaves output behind
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

	dataParser, err := parser.NewParser(config, opts, dataFailureWriter)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	outputFile, err := tempFileFor(outputPath)
	if err != nil {
		return fmt.Errorf("error creating output file '%s': %w", outputPath, err)
	}
	outputFile.Close()
	tempPaths = append(tempPaths, outputFile.Name())

	var waitGroup sync.WaitGroup
	waitGroup.Add(3)
	inputChan := make(chan reader.Record, queueBuffer)
	outputChan := make(chan map[string]any, queueBuffer)

	go dataReader.ReadRows(inputChan, &waitGroup, errorTracker)
	go dataParser.ParseData(inputChan, outputChan, &waitGroup, errorTracker)
	go avro_writer.WriteFile(outputFile.Name(), config, outputChan, &waitGroup, errorTracker)

	waitGroup.Wait()

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

	if err := os.Rename(outputFile.Name(), outputPath); err != nil {
		return fmt.Errorf("error writing output file: %w", err)
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
