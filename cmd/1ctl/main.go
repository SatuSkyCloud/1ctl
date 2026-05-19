package main

import (
	"fmt"
	"os"
	"strings"

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
	return app.Run(normalizeGlobalOutputArgs(os.Args))
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
			commands.LaunchCommand(),
			commands.DeployCommand(),
			commands.ServiceCommand(),
			commands.SecretCommand(),
			commands.IngressCommand(),
			commands.IssuerCommand(),
			commands.DomainsCommand(),
			commands.VolumesCommand(),
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
				if cmd.Name == cmdName || containsString(cmd.Aliases, cmdName) {
					commandExists = true
					break
				}
			}

			// If command doesn't exist, show help and return error
			if !commandExists {
				if err := cli.ShowAppHelp(c); err != nil {
					return utils.NewError("failed to show help", err)
				}
				msg := fmt.Sprintf("command %q not found\nRun '1ctl --help' for usage", cmdName)
				if cmdName == "storage" {
					msg += "\nPersistent volumes are managed with '1ctl volumes'."
				}
				if cmdName == "volume" {
					msg += "\nPersistent volumes are managed with '1ctl volumes'."
				}
				return utils.NewError(msg, nil)
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

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func normalizeGlobalOutputArgs(args []string) []string {
	if len(args) <= 2 {
		return args
	}

	normalized := make([]string, 0, len(args))
	outputArgs := make([]string, 0, 2)
	normalized = append(normalized, args[0])

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o" || arg == "--output":
			if i+1 < len(args) {
				outputArgs = append(outputArgs, arg, args[i+1])
				i++
				continue
			}
		case strings.HasPrefix(arg, "--output="), strings.HasPrefix(arg, "-o="):
			outputArgs = append(outputArgs, arg)
			continue
		}
		normalized = append(normalized, arg)
	}

	if len(outputArgs) == 0 {
		return args
	}

	result := make([]string, 0, len(args))
	result = append(result, normalized[0])
	result = append(result, outputArgs...)
	result = append(result, normalized[1:]...)
	return result
}

func main() {
	if err := run(); err != nil {
		_ = utils.HandleError(err) //nolint:errcheck
		os.Exit(1)
	}
}
