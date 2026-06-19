// Package environment defines the "1ctl env" command tree.
package environment

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagDeploymentID = "deployment-id"
	flagApp          = "app"
	flagConfig       = "config"
	flagName         = "name"
	flagEnv          = "env"
	flagKey          = "key"
)

// --- Input structs ------------------------------------------------------

type envCreateInput struct {
	DeploymentID string
	App          string
	Config       string
	Name         string
	Env          []string
}

type envListInput struct {
	DeploymentID string
	App          string
}

type envUnsetInput struct {
	DeploymentID string
	App          string
	Config       string
	Key          string
}

// --- Command tree -------------------------------------------------------

// Command returns the root env command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "env",
		Aliases: []string{"environment"},
		Usage:   "Manage environments for a deployment",
		Commands: []*cli.Command{
			envCreateCommand(),
			envListCommand(),
			envUnsetCommand(),
		},
	}
}

func envCreateCommand() *cli.Command {
	var in envCreateInput
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new environment",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagDeploymentID,
				Usage:       "Deployment ID",
				Destination: &in.DeploymentID,
			},
			&cli.StringFlag{
				Name:        flagConfig,
				Usage:       "Config name or path (e.g. staging, satusky.staging.toml). Default: satusky.toml",
				Destination: &in.Config,
			},
			&cli.StringFlag{
				Name:        flagName,
				Usage:       "App label (defaults to deployment name, auto-resolved from deployment-id)",
				Destination: &in.Name,
			},
			&cli.StringSliceFlag{
				Name:        flagEnv,
				Usage:       "Environment variables (format: KEY=VALUE)",
				Destination: &in.Env,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleCreateEnvironment(ctx, in) },
	}
}

func envListCommand() *cli.Command {
	var in envListInput
	return &cli.Command{
		Name:  "list",
		Usage: "List all environments",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagDeploymentID,
				Usage:       "Filter by deployment ID",
				Destination: &in.DeploymentID,
			},
			&cli.StringFlag{
				Name:        flagApp,
				Usage:       "App name to resolve (alternative to --deployment-id)",
				Destination: &in.App,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleListEnvironments(ctx) },
	}
}

func envUnsetCommand() *cli.Command {
	var in envUnsetInput
	return &cli.Command{
		Name:  "unset",
		Usage: "Remove a specific key from an environment",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagConfig,
				Usage:       "Config name or path",
				Destination: &in.Config,
			},
			&cli.StringFlag{
				Name:        flagDeploymentID,
				Aliases:     []string{"d"},
				Usage:       "Deployment ID",
				Destination: &in.DeploymentID,
			},
			&cli.StringFlag{
				Name:        flagApp,
				Usage:       "App name to resolve (alternative to --deployment-id)",
				Destination: &in.App,
			},
			&cli.StringFlag{
				Name:        flagKey,
				Aliases:     []string{"k"},
				Usage:       "Key to remove",
				Required:    true,
				Destination: &in.Key,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleEnvUnset(ctx, in) },
	}
}
