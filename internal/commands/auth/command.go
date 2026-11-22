// Package auth defines the "1ctl auth" command tree.
package auth

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const flagToken = "token"

// --- Input structs ------------------------------------------------------

type authLoginInput struct {
	Token string
}

// --- Command tree -------------------------------------------------------

// Command returns the root auth command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Authenticate and manage credentials for SatuSky Cloud",
		Commands: []*cli.Command{
			authLoginCommand(),
			authLogoutCommand(),
			authStatusCommand(),
		},
	}
}

func authLoginCommand() *cli.Command {
	var in authLoginInput
	return &cli.Command{
		Name:  "login",
		Usage: "Authenticate with Satusky",
		Description: `1ctl authenticates with an API token. Create one in the SatuSky console
under API Tokens (https://cloud.satusky.com/<org-id>/token), then use one of:
   1. CLI flag:     1ctl auth login --token=<your-token>
   2. Environment:  export SATUSKY_API_KEY=<your-token> && 1ctl auth login

With no token, 1ctl re-checks the token saved by an earlier login.
Browser and email/password login are not available in the CLI yet.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagToken,
				Usage:       "API token for authentication",
				Sources:     cli.EnvVars("SATUSKY_API_KEY"),
				Destination: &in.Token,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleLogin(ctx, in)
		},
	}
}

func authLogoutCommand() *cli.Command {
	return &cli.Command{
		Name:  "logout",
		Usage: "Remove stored authentication",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleLogout(ctx)
		},
	}
}

func authStatusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "View authentication status",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleAuthStatus(ctx)
		},
	}
}
