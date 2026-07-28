// Package init defines the "1ctl init" command tree.
package init

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const flagConfig = "config"

// --- Input structs ------------------------------------------------------

type initInput struct {
	Config string
}

// --- Command tree -------------------------------------------------------

// Command returns the init command.
func Command() *cli.Command {
	var in initInput
	return &cli.Command{
		Name:  "init",
		Usage: "Create a satusky.toml config file in the current directory",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagConfig,
				Usage:       "Config name (e.g. staging \u2192 creates satusky.staging.toml)",
				Destination: &in.Config,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if err := validateInitArgs(cmd.Args().Len()); err != nil {
				return err
			}
			return handleInit(ctx, in)
		},
	}
}

func validateInitArgs(count int) error {
	if count != 0 {
		return fmt.Errorf("init does not accept positional arguments")
	}
	return nil
}
