package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/fatih/color"
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
				},
				Action: func(cCtx *cli.Context) error {
					config, err := config.LoadConfig(cCtx.Path("config-path"))
					if err != nil {
						return fmt.Errorf("failed to read config: %w", err)
					}

					errorTracker := error_tracking.NewErrorTracker()
					errorTracker.Start()

					var waitGroup sync.WaitGroup
					waitGroup.Add(3)
					inputChan := make(chan []*string, queueBuffer)
					outputChan := make(chan map[string]any, queueBuffer)

					go reader.ReadFile(cCtx.Path("input-path"), config, inputChan, &waitGroup, errorTracker)
					go parser.ParseData(config, inputChan, outputChan, &waitGroup, errorTracker)
					go avro_writer.WriteFile(cCtx.Path("output-path"), config, outputChan, &waitGroup, errorTracker)

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
						Name:     "config-path",
						Aliases:  []string{"c"},
						Usage:    "config file path",
						Required: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					_, err := config.LoadConfig(cCtx.Path("config_path"))
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
