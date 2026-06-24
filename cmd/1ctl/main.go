package main

import (
	"context"
	"fmt"
	"os"

	"1ctl/internal/commands"
	"1ctl/internal/config"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"1ctl/internal/version"

	"github.com/urfave/cli/v3"
)

// Make run function accessible to tests
func run() error {
	cmd := createCommand()
	return cmd.Run(context.Background(), os.Args)
}

// Make createCommand function accessible to tests
func createCommand() *cli.Command {
	cmd := &cli.Command{
		EnableShellCompletion: true,
		Name:                  "1ctl",
		Usage:   "Deploy and manage applications on SatuSky Cloud",
		Version: version.GetVersionInfo(),
		Description: `1ctl is the command-line interface for SatuSky Cloud.

Quick start:
   1ctl profile create --url https://api.satusky.com/v1/cli prod
   1ctl profile use prod
   1ctl auth login --token <your-api-token>
   1ctl deploy --port 8080

Build & deploy:
   Images are built in the cloud — no local Docker required.
   Run 'satusky.toml' or use --dockerfile to control the build.
   Use --fast to request the accelerated cloud builder.
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
				Sources: cli.EnvVars("SATUSKY_PROFILE"),
			},
			&cli.StringFlag{
				Name:  "api-url",
				Usage: "API URL override for this command (highest priority, overrides profile and env var)",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: table or json (default: table)",
				Value:   "table",
			},
		},
		Commands: []*cli.Command{
			commands.AuthCommand(),
			commands.ProfileCommand(),
			commands.OrgCommand(),
			commands.InitCommand(),
			commands.LaunchCommand(),
			commands.DeployCommand(),
			commands.DoctorCommand(),
			commands.ServiceCommand(),
			commands.SecretCommand(),
			commands.IngressCommand(),
			commands.IssuerCommand(),
			commands.DomainsCommand(),
			commands.VolumesCommand(),
			commands.PostgresCommand(),
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
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// Apply --profile flag: sets profile for this process invocation only (not persisted)
			if profile := cmd.String("profile"); profile != "" {
				satuskyctx.SetProfileOverride(profile)
			}

			// Apply --api-url flag: highest-priority URL override, set via env var so GetConfig picks it up
			if apiURL := cmd.String("api-url"); apiURL != "" {
				if err := os.Setenv("SATUSKY_API_URL", apiURL); err != nil {
					return ctx, utils.NewError("failed to set API URL", err)
				}
			}

			// Apply --output flag: sets global output format for this process invocation
			if format := cmd.String("output"); format != "" {
				utils.SetOutputFormat(format)
			}

			// Get the command or first argument
			cmdName := cmd.Args().First()

			// Skip token validation for these cases
			if cmdName == "auth" ||
				cmdName == "profile" ||
				cmdName == "org" ||
				cmdName == "init" ||
				cmdName == "completion" ||
				cmdName == "help" ||
				cmd.Bool("help") ||
				cmd.Bool("h") ||
				cmd.Bool("version") ||
				cmd.Bool("v") ||
				len(cmd.Args().Slice()) == 0 {
				return ctx, nil
			}

			// Check if command exists in our registered commands
			commandExists := false
			for _, subCmd := range cmd.Commands {
				if subCmd.Name == cmdName || containsString(subCmd.Aliases, cmdName) {
					commandExists = true
					break
				}
			}

			// If command doesn't exist, show help and return error
			if !commandExists {
				if err := cli.ShowAppHelp(cmd); err != nil {
					return ctx, utils.NewError("failed to show help", err)
				}
				msg := fmt.Sprintf("command %q not found\nRun '1ctl --help' for usage", cmdName)
				if cmdName == "storage" {
					msg += "\nPersistent volumes are managed with '1ctl volumes'."
				}
				if cmdName == "volume" {
					msg += "\nPersistent volumes are managed with '1ctl volumes'."
				}
				return ctx, utils.NewError(msg, nil)
			}

			// Only validate environment for existing commands
			return ctx, config.ValidateEnvironment()
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return utils.NewError("No command specified, use --help for usage", nil)
		},
	}
	return cmd
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func main() {
	if err := run(); err != nil {
		_ = utils.HandleError(err) //nolint:errcheck
		os.Exit(1)
	}
}
