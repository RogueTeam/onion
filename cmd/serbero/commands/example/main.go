package example

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

const (
	NodeConfigFlag = "node-config"
)

var Command = cli.Command{
	Name:        "example",
	Description: "Utility for printing different kind of examples",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: NodeConfigFlag, Usage: "Node configuration file. Use '-' for writing to stdout", Value: "config.yaml"},
	},
	Action: func(ctx context.Context, cmd *cli.Command) (err error) {
		filename := cmd.String(NodeConfigFlag)
		if filename == "-" {
			err = yaml.NewEncoder(os.Stdout).Encode(&ExampleConfig)
			if err != nil {
				return fmt.Errorf("failed to write to stdout: %w", err)
			}
			return nil
		}

		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create example file: %w", err)
		}
		defer file.Close()

		err = yaml.NewEncoder(file).Encode(&ExampleConfig)
		if err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
		return nil
	},
}
