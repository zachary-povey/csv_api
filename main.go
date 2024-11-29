package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/urfave/cli/v2"

	"github.com/zachary-povey/csv_api/internal/avro_writer"
	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/reader"
)

const (
	queueBuffer int = 10
)

func main() {

	app := &cli.App{
		Name:        "csv_api",
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

			var waitGroup sync.WaitGroup
			waitGroup.Add(2)
			rowChan := make(chan []*string, queueBuffer)
			readErr := reader.ReadFile(cCtx.Path("data_path"), config, rowChan, &waitGroup)
			if readErr != nil {
				return readErr
			}
			avro_writer.WriteFile(cCtx.Path("output_path"), config, rowChan, &waitGroup)

			waitGroup.Wait()

			return nil
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
