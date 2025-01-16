package main

import (
	"os"

	"1ctl/internal/commands"
	"1ctl/internal/config"
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
		Usage:   "1ctl is the command line tool for Satusky",
		Version: version.Version,
		// TODO: commands for deploy marketplace apps
		Commands: []*cli.Command{
			commands.AuthCommand(),
			commands.DeployCommand(),
			commands.ServiceCommand(),
			commands.SecretCommand(),
			commands.IngressCommand(),
			commands.IssuerCommand(),
			commands.EnvironmentCommand(),
			commands.MachineCommand(),
			commands.CompletionCommand(),
		},
		Before: func(c *cli.Context) error {
			// Get the command or first argument
			cmdName := c.Args().First()

			// Skip token validation for these cases
			if cmdName == "auth" ||
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
		utils.HandleError(err)
		os.Exit(1)
	}
}
