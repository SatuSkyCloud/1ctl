package main

import (
	"context"
	"os"

	"1ctl/internal/cliapp"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v3"
)

// Make run function accessible to tests
func run() error {
	cmd := createCommand()
	return cmd.Run(context.Background(), os.Args)
}

// Make createCommand function accessible to tests
func createCommand() *cli.Command {
	return cliapp.New()
}

func main() {
	if err := run(); err != nil {
		_ = utils.HandleError(err) //nolint:errcheck
		os.Exit(1)
	}
}
