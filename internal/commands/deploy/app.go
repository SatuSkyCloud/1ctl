// Package deploy (file app.go) defines the "1ctl app" command tree.
// Subcommands delegate to the same unexported handlers as their "1ctl deploy"
// counterparts, providing a cleaner user-facing noun for app lifecycle.
package deploy

import (
	"context"
	"fmt"
	"strconv"

	"github.com/urfave/cli/v3"
)

// AppCommand returns the "1ctl app" command tree.
// This is the canonical namespace for managing deployed applications.
// The "1ctl deploy" subcommands continue to work for backward compatibility
// and for the "deploy" action itself.
func AppCommand() *cli.Command {
	return &cli.Command{
		Name:  "app",
		Usage: "Manage deployed applications",
		Description: `Manage the lifecycle of applications deployed to SatuSky Cloud.

Examples:
   1ctl app list
   1ctl app get my-app
   1ctl app status my-app
   1ctl app delete my-app
   1ctl app restart my-app
   1ctl app releases my-app
   1ctl app open my-app
   1ctl app scale my-app 3`,
		Commands: []*cli.Command{
			appListCommand(),
			appGetCommand(),
			appStatusCommand(),
			appDestroyCommand(),
			appRestartCommand(),
			appReleasesCommand(),
			appRollbackCommand(),
			appOpenCommand(),
			appScaleCommand(),
		},
	}
}

func appListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List deployed applications",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleListDeployments(ctx)
		},
	}
}

func appGetCommand() *cli.Command {
	var in GetDeploymentInput
	return &cli.Command{
		Name:      "get",
		Usage:     "Get application details",
		ArgsUsage: "<app-name>",
		Flags: []cli.Flag{
			optionalString(flagDeploymentID, "Deployment ID (alternative to positional arg)", &in.DeploymentID),
			optionalString(flagConfig, "Config name or path", &in.Config),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() >= 1 {
				arg := cmd.Args().First()
				if looksLikeUUID(arg) {
					in.DeploymentID = arg
				} else {
					in.App = arg
				}
			}
			return handleGetDeployment(ctx, in)
		},
	}
}

func appStatusCommand() *cli.Command {
	var in StatusInput
	return &cli.Command{
		Name:      "status",
		Usage:     "Check application status",
		ArgsUsage: "<app-name>",
		Flags: []cli.Flag{
			optionalString(flagDeploymentID, "Deployment ID (alternative to positional arg)", &in.DeploymentID),
			optionalBool(flagWatch, "Watch status in real-time", &in.Watch),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() >= 1 {
				arg := cmd.Args().First()
				if looksLikeUUID(arg) {
					in.DeploymentID = arg
				} else {
					in.App = arg
				}
			}
			return handleDeploymentStatus(ctx, in)
		},
	}
}

func appDestroyCommand() *cli.Command {
	var in DestroyInput
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"destroy", "rm"},
		Usage:     "Delete an application and all associated resources",
		ArgsUsage: "<app-name>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        flagYes,
				Aliases:     []string{"y"},
				Usage:       "Skip confirmation prompt",
				Destination: &in.Yes,
			},
			&cli.BoolFlag{
				Name:        "retain-volumes",
				Usage:       "Retain persistent volumes instead of deleting them",
				Destination: &in.RetainVolumes,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() >= 1 {
				arg := cmd.Args().First()
				if looksLikeUUID(arg) {
					in.DeploymentID = arg
				} else {
					in.App = arg
				}
			}
			return handleDestroyDeployment(ctx, in)
		},
	}
}

func appRestartCommand() *cli.Command {
	var in DeployRefInput
	return &cli.Command{
		Name:      "restart",
		Usage:     "Trigger a rolling restart without redeploying",
		ArgsUsage: "<app-name>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() >= 1 {
				arg := cmd.Args().First()
				if looksLikeUUID(arg) {
					in.DeploymentID = arg
				} else {
					in.App = arg
				}
			}
			return handleRestartDeployment(ctx, in)
		},
	}
}

func appReleasesCommand() *cli.Command {
	var in DeployRefInput
	return &cli.Command{
		Name:      "releases",
		Usage:     "List release history for an application",
		ArgsUsage: "<app-name>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() >= 1 {
				arg := cmd.Args().First()
				if looksLikeUUID(arg) {
					in.DeploymentID = arg
				} else {
					in.App = arg
				}
			}
			return handleListReleases(ctx, in)
		},
	}
}

func appRollbackCommand() *cli.Command {
	var in RollbackInput
	return &cli.Command{
		Name:      "rollback",
		Usage:     "Roll back to a previous release",
		ArgsUsage: "<app-name>",
		Flags: []cli.Flag{
			optionalInt(flagVersion, "Version number to roll back to (default: previous version)", &in.Version),
			optionalBool(flagYes, "Skip confirmation prompt", &in.Yes),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() >= 1 {
				arg := cmd.Args().First()
				if looksLikeUUID(arg) {
					in.DeploymentID = arg
				} else {
					in.App = arg
				}
			}
			return handleRollback(ctx, in)
		},
	}
}

func appOpenCommand() *cli.Command {
	var in DeployRefInput
	return &cli.Command{
		Name:      "open",
		Usage:     "Open an application's URL in the default browser",
		ArgsUsage: "<app-name>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() >= 1 {
				arg := cmd.Args().First()
				if looksLikeUUID(arg) {
					in.DeploymentID = arg
				} else {
					in.App = arg
				}
			}
			return handleOpenDeployment(ctx, in)
		},
	}
}

func appScaleCommand() *cli.Command {
	var in ScaleInput
	return &cli.Command{
		Name:      "scale",
		Usage:     "Set the replica count without redeploying",
		ArgsUsage: "<app-name> <replicas>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        flagReplicas,
				Usage:       "Target replica count",
				Destination: &in.Replicas,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) >= 1 {
				arg := args[0]
				if looksLikeUUID(arg) {
					in.DeploymentID = arg
				} else {
					in.App = arg
				}
			}
			// Replicas from positional arg if --replicas flag not set
			if len(args) >= 2 && !cmd.IsSet(flagReplicas) {
				n, err := strconv.Atoi(args[1])
				if err == nil {
					in.Replicas = n
				}
			}
			if in.Replicas < 1 {
				return fmt.Errorf("replica count is required — use: 1ctl app scale <app> <n> or --replicas <n>")
			}
			return handleScaleDeployment(ctx, in)
		},
	}
}
