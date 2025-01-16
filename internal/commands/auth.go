package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

func AuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Display commands for authentication",
		Subcommands: []*cli.Command{
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
						EnvVars: []string{"SATUSKY_API_KEY"},
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

func handleLogin(c *cli.Context) error {
	// Try to get token from flag first, then environment variable
	token := c.String("token")
	if token == "" {
		// check in context.json
		token = context.GetToken()
		if token == "" {
			return utils.NewError("token is required. Use --token flag or set SATUSKY_API_KEY environment variable", nil)
		}
	}

	// Store token in context.json
	if err := context.SetToken(token); err != nil {
		return utils.NewError("failed to store token: %w", err)
	}

	// Validate token with API
	result, err := api.LoginCLI(token)
	if err != nil {
		return utils.NewError("failed to login: %w", err)
	}

	// Store user ID in context.json
	if err := context.SetUserID(result.UserID.String()); err != nil {
		return utils.NewError("failed to store user ID: %w", err)
	}

	// Store namespace in context.json
	if err := context.SetCurrentNamespace(result.OrganizationName); err != nil {
		return utils.NewError("failed to store namespace: %w", err)
	}

	// Store user config key in context.json
	if err := context.SetUserConfigKey(result.UserConfigKey); err != nil {
		return utils.NewError("failed to store user config key: %w", err)
	}

	utils.PrintSuccess("Logged in successfully to SatuSky 1ctl as %s!\n", result.UserEmail)
	return nil
}

func handleLogout(c *cli.Context) error {
	// Clear token from context
	if err := context.SetToken(""); err != nil {
		return utils.NewError("failed to clear token: %w", err)
	}

	// Clear namespace from context
	if err := context.SetCurrentNamespace(""); err != nil {
		return utils.NewError("failed to clear namespace: %w", err)
	}

	// Clear user config key from context
	if err := context.SetUserConfigKey(""); err != nil {
		return utils.NewError("failed to clear user config key: %w", err)
	}

	// Clear user ID from context
	if err := context.SetUserID(""); err != nil {
		return utils.NewError("failed to clear user ID: %w", err)
	}

	utils.PrintSuccess("Successfully logged out")
	return nil
}

func handleAuthStatus(c *cli.Context) error {
	token := context.GetToken()
	if token == "" {
		return utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate", nil)
	}

	// Validate token with API
	result, err := api.LoginCLI(token)
	if err != nil {
		return utils.NewError("failed to check token status: %w", err)
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
	utils.PrintStatusLine("Token expires", fmt.Sprintf("in %.0f days", daysUntilExpiry))
	return nil
}
