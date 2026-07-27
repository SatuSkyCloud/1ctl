// Package cliapp assembles the complete 1ctl command tree.
//
// Keeping the tree outside package main lets the executable, contract
// generator, and tests inspect the exact same command definitions.
package cliapp

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

// New returns the complete 1ctl application command tree.
func New() *cli.Command {
	cmd := &cli.Command{
		EnableShellCompletion: true,
		Name:                  "1ctl",
		Usage:                 "Deploy and manage applications on SatuSky Cloud",
		Version:               version.GetVersionInfo(),
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
				Usage:   "Output format: table or json",
				Value:   "table",
			},
		},
		Commands: []*cli.Command{
			withCategory(commands.AdminCommand(), "Billing & operations"),
			withCategory(commands.InitCommand(), "Core workflow"),
			withCategory(commands.LaunchCommand(), "Core workflow"),
			withCategory(commands.DeployCommand(), "Core workflow"),
			withCategory(commands.AppCommand(), "Applications"),
			withCategory(commands.LogsCommand(), "Core workflow"),
			withCategory(commands.DoctorCommand(), "Core workflow"),
			withCategory(commands.DomainsCommand(), "Applications"),
			withCategory(commands.ConfigCommand(), "Applications"),
			withCategory(commands.SecretCommand(), "Applications"),
			withCategory(commands.VolumesCommand(), "Applications"),
			withCategory(commands.PostgresCommand(), "Data"),
			withCategory(commands.ValkeyCommand(), "Data"),
			withCategory(commands.NATSCommand(), "Data"),
			withCategory(commands.MachineCommand(), "Infrastructure"),
			withCategory(commands.ClusterCommand(), "Infrastructure"),
			withCategory(commands.MarketplaceCommand(), "Catalog"),
			withCategory(commands.PackageCommand(), "Catalog"),
			withCategory(commands.AuthCommand(), "Account"),
			withCategory(commands.ProfileCommand(), "Account"),
			withCategory(commands.OrgCommand(), "Account"),
			withCategory(commands.UserCommand(), "Account"),
			withCategory(commands.TokenCommand(), "Account"),
			withCategory(commands.CreditsCommand(), "Billing & operations"),
			withCategory(commands.PricingCommand(), "Billing & operations"),
			withCategory(commands.AuditCommand(), "Billing & operations"),
			withCategory(commands.NotificationsCommand(), "Billing & operations"),
			withCategory(commands.CompletionCommand(), "Billing & operations"),
			commands.ServiceCommand(),
			commands.IngressCommand(),
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			if profile := cmd.String("profile"); profile != "" {
				satuskyctx.SetProfileOverride(profile)
			}
			if apiURL := cmd.String("api-url"); apiURL != "" {
				if err := os.Setenv("SATUSKY_API_URL", apiURL); err != nil {
					return ctx, utils.NewError("failed to set API URL", err)
				}
			}
			if format := cmd.String("output"); format != "" {
				utils.SetOutputFormat(format)
			}

			cmdName := cmd.Args().First()
			packageCreate := cmdName == "package" && cmd.Args().Get(1) == "create"
			if cmdName == "auth" ||
				cmdName == "profile" ||
				cmdName == "org" ||
				cmdName == "init" ||
				cmdName == "completion" ||
				packageCreate ||
				cmdName == "help" ||
				cmd.Bool("help") ||
				cmd.Bool("h") ||
				cmd.Bool("version") ||
				cmd.Bool("v") ||
				len(cmd.Args().Slice()) == 0 {
				return ctx, nil
			}

			commandExists := false
			for _, subCmd := range cmd.Commands {
				if subCmd.Name == cmdName || containsString(subCmd.Aliases, cmdName) {
					commandExists = true
					break
				}
			}
			if !commandExists {
				if err := cli.ShowAppHelp(cmd); err != nil {
					return ctx, utils.NewError("failed to show help", err)
				}
				msg := fmt.Sprintf("command %q not found\nRun '1ctl --help' for usage", cmdName)
				if cmdName == "storage" || cmdName == "volume" {
					msg += "\nPersistent volumes are managed with '1ctl volumes'."
				}
				return ctx, utils.NewError(msg, nil)
			}
			return ctx, config.ValidateEnvironment()
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return utils.NewError("No command specified, use --help for usage", nil)
		},
	}
	return cmd
}

func withCategory(cmd *cli.Command, category string) *cli.Command {
	cmd.Category = category
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
