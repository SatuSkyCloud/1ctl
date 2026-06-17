package commands

import (
	"context"
	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"
)

func AuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Authenticate and manage credentials for SatuSky Cloud",
		Commands: []*cli.Command{
			{
				Name:  "login",
				Usage: "Authenticate with Satusky",
				Description: `Authenticate using one of these methods:
   1. CLI flag:     1ctl auth login --token=<your-token>
   2. Environment:  export SATUSKY_API_KEY=<your-token> && 1ctl auth login`,
				Action: handleLogin,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "token",
						Usage:   "API token for authentication",
						Sources: cli.EnvVars("SATUSKY_API_KEY"),
					},
				},
			},
			{
				Name:   "logout",
				Usage:  "Remove stored authentication",
				Action: handleLogout,
			},
			{
				Name:   "status",
				Usage:  "View authentication status",
				Action: handleAuthStatus,
			},
		},
	}
}

func handleLogin(ctx context.Context, cmd *cli.Command) error {
	// Try to get token from flag first, then environment variable
	token := cmd.String("token")
	if token == "" {
		// check in context.json
		token = satuskyctx.GetToken()
		if token == "" {
			return utils.NewError("token is required. Use --token flag or set SATUSKY_API_KEY environment variable", nil)
		}
	}

	// Validate token with API first, before writing anything to disk
	result, err := api.LoginCLI(token)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to login: %s", err.Error()), nil)
	}

	// Fail-fast: a token unassociated with an organization is unusable. Writing
	// an empty namespace to context.json poisons every subsequent command with
	// cryptic "not found" errors. Surface the cause at login instead.
	if result.OrganizationID == "" || result.Namespace == "" {
		return utils.NewError("token is not associated with an organization — ensure the token has an organization scope", nil)
	}

	if err := satuskyctx.SaveLoginState(token, result.UserID, result.UserEmail, result.OrganizationID, result.OrganizationName, result.Namespace); err != nil {
		return utils.NewError(fmt.Sprintf("failed to store login state: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Logged in successfully to SatuSky 1ctl as %s!\n", result.UserEmail)
	return nil
}

func handleLogout(ctx context.Context, cmd *cli.Command) error {
	if err := satuskyctx.ClearAuthState(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to clear auth state: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Successfully logged out")
	return nil
}

func handleAuthStatus(ctx context.Context, cmd *cli.Command) error {
	token := satuskyctx.GetToken()
	if token == "" {
		return utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate", nil)
	}

	// Validate token with API
	result, err := api.LoginCLI(token)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to check token status: %s", err.Error()), nil)
	}

	if !result.IsActive {
		return utils.NewError("token is not active", nil)
	}

	if !result.Valid {
		return utils.NewError("token is invalid", nil)
	}

	daysUntilExpiry := time.Until(result.ExpiresAt).Hours() / 24

	utils.PrintSuccess("Authenticated with Satusky\n")
	utils.PrintStatusLine("User Email", result.UserEmail)
	utils.PrintStatusLine("Organization", result.OrganizationName)
	utils.PrintStatusLine("Organization ID", result.OrganizationID)
	utils.PrintStatusLine("Namespace", result.Namespace)
	utils.PrintStatusLine("Token expires", fmt.Sprintf("in %.0f days", daysUntilExpiry))
	return nil
}
