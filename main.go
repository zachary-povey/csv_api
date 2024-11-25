package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {

	app := &cli.App{
		Name:        "csv_api",
		Description: "Validates a csv file and extracts it into parquet.",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "config_path",
				Usage:    "config file path",
				Required: true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			fmt.Println("Hola", cCtx.String("config_path"))
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
