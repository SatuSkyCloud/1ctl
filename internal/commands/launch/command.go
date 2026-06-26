// Package launch defines the "1ctl launch" command tree.
package launch

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const flagNonInteractive = "non-interactive"

// --- Input structs ------------------------------------------------------

type launchInput struct {
	NonInteractive bool
}

// --- Command tree -------------------------------------------------------

// Command returns the launch command.
func Command() *cli.Command {
	var in launchInput
	return &cli.Command{
		Name:  "launch",
		Usage: "Interactive wizard: detect runtime, write satusky.toml, ready to deploy",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        flagNonInteractive,
				Usage:       "Skip prompts and accept all detected defaults",
				Destination: &in.NonInteractive,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleLaunch(ctx, in)
		},
	}
}
