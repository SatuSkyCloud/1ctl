// Package secret defines the "1ctl secret" command tree — flag names,
// input structs, and CLI wiring. Handler logic lives in handlers.go.
package secret

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
	flagKV           = "kv"
	flagKey          = "key"
	flagID           = "id"
)

// --- Input structs ------------------------------------------------------

type secretCreateInput struct {
	DeploymentID string
	App          string
	Config       string
	Name         string
	KV           []string
}

type secretGetInput struct {
	ID string
}

type secretUnsetInput struct {
	DeploymentID string
	App          string
	Config       string
	Key          string
}

// --- Command tree -------------------------------------------------------

// Command returns the root secret command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "secret",
		Usage: "Manage secrets",
		Commands: []*cli.Command{
			secretCreateCommand(),
			secretListCommand(),
			secretGetCommand(),
			secretUnsetCommand(),
		},
	}
}

func secretCreateCommand() *cli.Command {
	var in secretCreateInput
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new secret",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagDeploymentID,
				Usage:       "Deployment ID",
				Destination: &in.DeploymentID,
			},
			&cli.StringFlag{
				Name:        flagApp,
				Usage:       "App name to resolve (alternative to --deployment-id)",
				Destination: &in.App,
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
				Name:        flagKV,
				Aliases:     []string{"env"},
				Usage:       "Secret key-value pairs (format: KEY=VALUE)",
				Destination: &in.KV,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleCreateSecret(ctx, in)
		},
	}
}

func secretListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all secrets",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleListSecrets(ctx)
		},
	}
}

func secretGetCommand() *cli.Command {
	var in secretGetInput
	return &cli.Command{
		Name:  "get",
		Usage: "Show secret details including key names",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagID,
				Usage:       "Secret ID",
				Required:    true,
				Destination: &in.ID,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleGetSecret(ctx, in)
		},
	}
}

func secretUnsetCommand() *cli.Command {
	var in secretUnsetInput
	return &cli.Command{
		Name:  "unset",
		Usage: "Remove a specific key from a secret",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: flagApp, Usage: "App name to resolve (alternative to --deployment-id)", Destination: &in.App},
			&cli.StringFlag{Name: flagConfig, Usage: "Config name or path", Destination: &in.Config},
			&cli.StringFlag{Name: flagDeploymentID, Aliases: []string{"d"}, Usage: "Deployment ID", Destination: &in.DeploymentID},
			&cli.StringFlag{Name: flagKey, Aliases: []string{"k"}, Usage: "Key to remove", Required: true, Destination: &in.Key},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleSecretUnset(ctx, in)
		},
	}
}
