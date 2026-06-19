// Package volumes defines the "1ctl volumes" command tree.
package volumes

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagDeploymentID = "deployment-id"
	flagApp          = "app"
	flagConfig       = "config"
	flagVolumeID     = "volume-id"
	flagYes          = "yes"
)

// --- Input structs ------------------------------------------------------

type volumesListInput struct {
	DeploymentID string
	App          string
	Config       string
}

type volumesActionInput struct {
	VolumeID string
	Yes      bool
}

// --- Flag constructors --------------------------------------------------

func requiredString(name, usage string, dest *string, validate func(string) error) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Required:    true,
		Validator:   validate,
	}
}

// --- Command tree -------------------------------------------------------

// Command returns the root volumes command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "volumes",
		Aliases: []string{"volume"},
		Usage:   "Inspect, detach, and destroy persistent volumes",
		Commands: []*cli.Command{
			volumesListCommand(),
			volumesInspectCommand(),
			volumesDetachCommand(),
			volumesDestroyCommand(),
		},
	}
}

func volumesListCommand() *cli.Command {
	var in volumesListInput
	return &cli.Command{
		Name:  "list",
		Usage: "List persistent volumes for a deployment",
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
				Usage:       "Config name or path. Default: satusky.toml",
				Destination: &in.Config,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleVolumesList(ctx, in) },
	}
}

func volumesInspectCommand() *cli.Command {
	var in volumesActionInput
	return &cli.Command{
		Name:    "inspect",
		Aliases: []string{"get"},
		Usage:   "Inspect PVC and mount state for a volume",
		Flags: []cli.Flag{
			requiredString(flagVolumeID, "Volume ID", &in.VolumeID, nil),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleVolumesInspect(ctx, in) },
	}
}

func volumesDetachCommand() *cli.Command {
	var in volumesActionInput
	return &cli.Command{
		Name:  "detach",
		Usage: "Detach a volume from its deployment without deleting the PVC",
		Flags: []cli.Flag{
			requiredString(flagVolumeID, "Volume ID", &in.VolumeID, nil),
			&cli.BoolFlag{
				Name:        flagYes,
				Aliases:     []string{"y"},
				Usage:       "Skip confirmation prompt",
				Destination: &in.Yes,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleVolumesDetach(ctx, in) },
	}
}

func volumesDestroyCommand() *cli.Command {
	var in volumesActionInput
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"destroy", "rm"},
		Usage:   "Detach and delete a persistent volume claim",
		Flags: []cli.Flag{
			requiredString(flagVolumeID, "Volume ID", &in.VolumeID, nil),
			&cli.BoolFlag{
				Name:        flagYes,
				Aliases:     []string{"y"},
				Usage:       "Skip confirmation prompt",
				Destination: &in.Yes,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleVolumesDestroy(ctx, in) },
	}
}
