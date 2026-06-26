// Package service defines the "1ctl service" command tree.
package service

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagDeploymentID = "deployment-id"
	flagName         = "name"
	flagPort         = "port"
	flagNamespace    = "namespace"
	flagServiceID    = "service-id"
	flagYes          = "yes"
)

// --- Input structs ------------------------------------------------------

type serviceUpsertInput struct {
	DeploymentID string
	Name         string
	Port         int
	Namespace    string
}

type serviceDeleteInput struct {
	ServiceID string
	Yes       bool
}

// --- Command tree -------------------------------------------------------

// Command returns the root service command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:   "service",
		Usage:  "Create or update a Kubernetes Service (low-level — deploy creates services automatically)",
		Hidden: true,
		Commands: []*cli.Command{
			serviceListCommand(),
			serviceDeleteCommand(),
		},
		Action: serviceUpsertAction,
	}
}

func serviceUpsertAction(ctx context.Context, cmd *cli.Command) error {
	// If no flags are set, show help
	if !cmd.IsSet("deployment-id") && !cmd.IsSet("name") && !cmd.IsSet("port") && cmd.NArg() == 0 {
		return cli.ShowCommandHelp(ctx, cmd, "service")
	}
	if cmd.NArg() > 0 {
		return cli.ShowSubcommandHelp(cmd)
	}
	// Gather flags
	var in serviceUpsertInput
	in.DeploymentID = cmd.String("deployment-id")
	in.Name = cmd.String("name")
	in.Port = cmd.Int("port")
	in.Namespace = cmd.String("namespace")
	return handleUpsertService(ctx, in)
}

func serviceListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all services",
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleListServices(ctx) },
	}
}

func serviceDeleteCommand() *cli.Command {
	var in serviceDeleteInput
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a service",
		ArgsUsage: "<service-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        flagYes,
				Aliases:     []string{"y"},
				Usage:       "Skip confirmation prompt",
				Destination: &in.Yes,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.ServiceID = cmd.Args().First()
			return handleDeleteService(ctx, in)
		},
	}
}
