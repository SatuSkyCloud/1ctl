// Package token defines the "1ctl token" command tree — flag names,
// input structs, and CLI wiring.  Handler logic lives in handlers.go.
package token

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagTokenName = "name"
	flagExpires   = "expires"
	flagID        = "id"
	flagYes       = "yes"
)

// --- Input structs ------------------------------------------------------

type tokenCreateInput struct {
	Name    string
	Expires int
}

type tokenIDInput struct {
	TokenID string
}

type tokenDeleteInput struct {
	TokenID string
	Yes     bool
}

// --- Command tree -------------------------------------------------------

// Command returns the root token command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "token",
		
		Usage:   "Manage API tokens",
		Commands: []*cli.Command{
			tokenListCommand(),
			tokenCreateCommand(),
			tokenGetCommand(),
			tokenEnableCommand(),
			tokenDisableCommand(),
			tokenDeleteCommand(),
		},
	}
}

func tokenListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List API tokens",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleTokenList(ctx)
		},
	}
}

func tokenCreateCommand() *cli.Command {
	var in tokenCreateInput
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new API token",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        flagExpires,
				Usage:       "Token expiry in days (0 for no expiry)",
				Value:       0,
				Destination: &in.Expires,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Name = cmd.Args().First()
			return handleTokenCreate(ctx, in)
		},
	}
}

func tokenGetCommand() *cli.Command {
	var in tokenIDInput
	return &cli.Command{
		Name:  "get",
		Usage: "Get token details",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagID,
				Usage:       "Token ID",
				Required:    true,
				Destination: &in.TokenID,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleTokenGet(ctx, in)
		},
	}
}

func tokenEnableCommand() *cli.Command {
	var in tokenIDInput
	return &cli.Command{
		Name:  "enable",
		Usage: "Enable a token",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagID,
				Usage:       "Token ID",
				Required:    true,
				Destination: &in.TokenID,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleTokenEnable(ctx, in)
		},
	}
}

func tokenDisableCommand() *cli.Command {
	var in tokenIDInput
	return &cli.Command{
		Name:  "disable",
		Usage: "Disable a token",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagID,
				Usage:       "Token ID",
				Required:    true,
				Destination: &in.TokenID,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleTokenDisable(ctx, in)
		},
	}
}

func tokenDeleteCommand() *cli.Command {
	var in tokenDeleteInput
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a token",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagID,
				Usage:       "Token ID",
				Required:    true,
				Destination: &in.TokenID,
			},
			&cli.BoolFlag{
				Name:        flagYes,
				Aliases:     []string{"y"},
				Usage:       "Skip confirmation prompt",
				Destination: &in.Yes,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleTokenDelete(ctx, in)
		},
	}
}
