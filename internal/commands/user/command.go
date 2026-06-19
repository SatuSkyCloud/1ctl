// Package user defines the "1ctl user" command tree — flag names,
// input structs, and CLI wiring.  Handler logic lives in handlers.go.
package user

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagName  = "name"
	flagEmail = "email"
)

// --- Input structs ------------------------------------------------------

type userUpdateInput struct {
	Name  string
	Email string
}

// --- Command tree -------------------------------------------------------

// Command returns the root user command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "Manage user account",
		Commands: []*cli.Command{
			userMeCommand(),
			userUpdateCommand(),
			userPasswordCommand(),
			userPermissionsCommand(),
			userSessionsCommand(),
		},
	}
}

func userMeCommand() *cli.Command {
	return &cli.Command{
		Name:    "me",
		Aliases: []string{"info"},
		Usage:   "Show current user profile",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleUserMe(ctx)
		},
	}
}

func userUpdateCommand() *cli.Command {
	var in userUpdateInput
	return &cli.Command{
		Name:  "update",
		Usage: "Update user profile",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagName,
				Usage:       "New display name",
				Destination: &in.Name,
			},
			&cli.StringFlag{
				Name:        flagEmail,
				Usage:       "New email address",
				Destination: &in.Email,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleUserUpdate(ctx, in)
		},
	}
}

func userPasswordCommand() *cli.Command {
	return &cli.Command{
		Name:   "password",
		Usage:  "Change password (interactive)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleUserPassword(ctx)
		},
	}
}

func userPermissionsCommand() *cli.Command {
	return &cli.Command{
		Name:   "permissions",
		Usage:  "Show current permissions",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleUserPermissions(ctx)
		},
	}
}

func userSessionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "sessions",
		Usage: "Manage sessions",
		Commands: []*cli.Command{
			{
				Name:   "revoke",
				Usage:  "Revoke all sessions",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return handleUserSessionsRevoke(ctx)
				},
			},
		},
	}
}
