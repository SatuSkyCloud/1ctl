// Package profile defines the "1ctl profile" command tree.
package profile

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagURL = "url"
)

// --- Input structs ------------------------------------------------------

type profileCreateInput struct {
	Name string
	URL  string
}

type profileNameInput struct {
	Name string
}

// --- Command tree -------------------------------------------------------

// Command returns the root profile command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "Manage named configuration profiles (API endpoints + credentials)",
		Commands: []*cli.Command{
			profileListCommand(),
			profileCreateCommand(),
			profileUseCommand(),
			profileCurrentCommand(),
			profileDeleteCommand(),
		},
	}
}

func profileListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all profiles",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleProfileList(ctx)
		},
	}
}

func profileCreateCommand() *cli.Command {
	var in profileCreateInput
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new profile",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagURL,
				Usage:       "API URL for this profile (e.g. http://localhost:8080/v1/cli)",
				Destination: &in.URL,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Name = cmd.Args().First()
			return handleProfileCreate(ctx, in)
		},
	}
}

func profileUseCommand() *cli.Command {
	var in profileNameInput
	return &cli.Command{
		Name:      "use",
		Usage:     "Switch to a profile",
		ArgsUsage: "<name>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Name = cmd.Args().First()
			return handleProfileUse(ctx, in)
		},
	}
}

func profileCurrentCommand() *cli.Command {
	return &cli.Command{
		Name:  "current",
		Usage: "Show the active profile",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleProfileCurrent(ctx)
		},
	}
}

func profileDeleteCommand() *cli.Command {
	var in profileNameInput
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a profile",
		ArgsUsage: "<name>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Name = cmd.Args().First()
			return handleProfileDelete(ctx, in)
		},
	}
}
