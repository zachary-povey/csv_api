package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/urfave/cli/v2"

	"github.com/zachary-povey/csv_api/internal/avro_writer"
	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
	"github.com/zachary-povey/csv_api/internal/parser"
	"github.com/zachary-povey/csv_api/internal/reader"
)

const (
	queueBuffer int = 10
)

func main() {

	app := &cli.App{
		Name: "csv_api",
		Commands: []*cli.Command{
			{
				Name:        "parse",
				Usage:       "Validates a csv file and extracts it into avro.",
				Description: "Validates a csv file and extracts it into avro.",
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:     "config_path",
						Usage:    "config file path",
						Required: true,
					},

					&cli.PathFlag{
						Name:     "data_path",
						Usage:    "data file path",
						Required: true,
					},

					&cli.PathFlag{
						Name:     "output_path",
						Usage:    "output file path",
						Required: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					config, err := config.LoadConfig(cCtx.Path("config_path"))
					if err != nil {
						return fmt.Errorf("failed to read config: %w", err)
					}

					errorTracker := error_tracking.NewErrorTracker()
					errorTracker.Start()

					var waitGroup sync.WaitGroup
					waitGroup.Add(3)
					inputChan := make(chan []*string, queueBuffer)
					outputChan := make(chan map[string]any, queueBuffer)

					go reader.ReadFile(cCtx.Path("data_path"), config, inputChan, &waitGroup, errorTracker)
					go parser.ParseData(config, inputChan, outputChan, &waitGroup, errorTracker)
					go avro_writer.WriteFile(cCtx.Path("output_path"), config, outputChan, &waitGroup, errorTracker)

					waitGroup.Wait()
					errorTracker.Stop()

					if len(errorTracker.ExecutionErrors) > 0 {
						return errorTracker.CombinedExecutionError()
					} else if errorTracker.ErrorReport.ContainsErrors() {
						return errors.New(errorTracker.ErrorReport.ConsoleFormat())
					} else {
						return nil
					}
				},
			},
			{
				Name:        "validate_config",
				Usage:       "Validates a csv_api config file.",
				Description: "Validates a csv_api config file.",
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:     "config_path",
						Usage:    "config file path",
						Required: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					_, err := config.LoadConfig(cCtx.Path("config_path"))
					if err != nil {
						return fmt.Errorf("failed to read config:\n%w", err)
					}
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// load config
// validate config
// validate file (for example, if specific args + named groups of regex don't provide all args - error)
// validate basic csv structure (file errors): e.g. column names, commas etc
// open stream to csv
// for each row:
//  check for row errors (column count etc)
// 	for each cell:
//		run regex
//      if fails: add cell error
//      else:
//        for first matched regex instantiate logical type using output from regex + args provided by
//   on success: write row to parquet buffer
