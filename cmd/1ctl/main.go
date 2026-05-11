package main

import (
	"os"

	"1ctl/internal/commands"
	"1ctl/internal/config"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"1ctl/internal/version"

	"github.com/urfave/cli/v2"
)

// Make run function accessible to tests
func run() error {
	app := createApp()
	return app.Run(os.Args)
}

// Make createApp function accessible to tests
func createApp() *cli.App {
	app := &cli.App{
		Name:    "1ctl",
		Usage:   "Deploy and manage applications on SatuSky Cloud",
		Version: version.GetVersionInfo(),
		Description: `1ctl is the command-line interface for SatuSky Cloud.

Quick start:
   1ctl profile create --url https://api.satusky.com/v1/cli prod
   1ctl profile use prod
   1ctl auth login --token <your-api-token>
   1ctl deploy --port 8080

Build & deploy:
   Images are built in the cloud via Kaniko — no local Docker required.
   Run 'satusky.toml' or use --dockerfile to control the build.
   Use --image <ref> to skip the build step with a pre-built image.

Profiles (multi-environment):
   1ctl profile create --url http://localhost:8080/v1/cli local
   1ctl profile use local
   1ctl --profile local deploy --port 8080   # one-shot override

Docs:   https://docs.satusky.com/cli
Tokens: https://cloud.satusky.com/<org-id>/token`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "profile",
				Aliases: []string{"p"},
				Usage:   "Profile to use for this command (e.g. --profile local)",
				EnvVars: []string{"SATUSKY_PROFILE"},
			},
			&cli.StringFlag{
				Name:  "api-url",
				Usage: "API URL override for this command (highest priority, overrides profile and env var)",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: table (default) or json",
				Value:   "table",
			},
		},
		Commands: []*cli.Command{
			commands.AuthCommand(),
			commands.ProfileCommand(),
			commands.OrgCommand(),
			commands.InitCommand(),
			commands.DeployCommand(),
			commands.ServiceCommand(),
			commands.SecretCommand(),
			commands.IngressCommand(),
			commands.IssuerCommand(),
			commands.DomainsCommand(),
			commands.EnvironmentCommand(),
			commands.MachineCommand(),
			commands.CompletionCommand(),
			// Phase 1: Credits, Logs
			commands.CreditsCommand(),
			commands.LogsCommand(),
			// Phase 2: Notifications
			commands.NotificationsCommand(),
			// Phase 3: User, Token
			commands.UserCommand(),
			commands.TokenCommand(),
			// Phase 5: Marketplace, Audit
			commands.MarketplaceCommand(),
			commands.AuditCommand(),
			// Phase 3+4: Pricing
			commands.PricingCommand(),
			commands.ClusterCommand(),
		},
		Before: func(c *cli.Context) error {
			// Apply --profile flag: sets profile for this process invocation only (not persisted)
			if profile := c.String("profile"); profile != "" {
				context.SetProfileOverride(profile)
			}

			// Apply --api-url flag: highest-priority URL override, set via env var so GetConfig picks it up
			if apiURL := c.String("api-url"); apiURL != "" {
				if err := os.Setenv("SATUSKY_API_URL", apiURL); err != nil {
					return utils.NewError("failed to set API URL", err)
				}
			}

			// Apply --output flag: sets global output format for this process invocation
			if format := c.String("output"); format != "" {
				utils.SetOutputFormat(format)
			}

			// Get the command or first argument
			cmdName := c.Args().First()

			// Skip token validation for these cases
			if cmdName == "auth" ||
				cmdName == "profile" ||
				cmdName == "org" ||
				cmdName == "init" ||
				cmdName == "completion" ||
				cmdName == "help" ||
				c.Bool("help") ||
				c.Bool("h") ||
				c.Bool("version") ||
				c.Bool("v") ||
				len(c.Args().Slice()) == 0 {
				return nil
			}

			// Check if command exists in our registered commands
			commandExists := false
			for _, cmd := range c.App.Commands {
				if cmd.Name == cmdName {
					commandExists = true
					break
				}
			}

			// If command doesn't exist, show help and return error
			if !commandExists {
				if err := cli.ShowAppHelp(c); err != nil {
					return utils.NewError("failed to show help", err)
				}
				return utils.NewError("command %q not found\nRun '1ctl --help' for usage", nil)
			}

			// Only validate environment for existing commands
			return config.ValidateEnvironment()
		},
		Action: func(c *cli.Context) error {
			return utils.NewError("No command specified, use --help for usage", nil)
		},
	}
	return app
}

func main() {
	if err := run(); err != nil {
		_ = utils.HandleError(err) //nolint:errcheck
		os.Exit(1)
	}
}
