// Package packagecmd defines the marketplace package authoring and publishing
// command tree.
package packagecmd

import (
	"context"

	"github.com/urfave/cli/v3"
)

const (
	flagConfig = "config"
	flagChart  = "chart"
	flagImage  = "image"
	flagOutput = "output"
	flagPublic = "public"
	flagReason = "reason"
)

type createInput struct {
	Config string
	Chart  string
	Image  string
	Output string
}

type publishInput struct {
	Artifact string
	Public   bool
	Reason   string
}

// Command returns the root package command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "package",
		Usage: "Create and publish marketplace packages",
		Commands: []*cli.Command{
			createCommand(),
			publishCommand(),
			listCommand(),
			statusCommand(),
		},
	}
}

func createCommand() *cli.Command {
	var input createInput
	return &cli.Command{
		Name:  "create",
		Usage: "Create an unsigned package from satusky.toml or an embedded Helm chart",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: flagConfig, Usage: "Config name or path (e.g. staging, satusky.staging.toml)", Destination: &input.Config},
			&cli.StringFlag{Name: flagChart, Usage: "Self-contained Helm chart directory (offline; requires Chart.yaml satusky.com/supported-architectures annotation)", Destination: &input.Chart},
			&cli.StringFlag{Name: flagImage, Usage: "Immutable image override (image@sha256:...)", Destination: &input.Image},
			&cli.StringFlag{Name: flagOutput, Usage: "Destination tar.gz path", Destination: &input.Output},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleCreate(ctx, input)
		},
	}
}

func publishCommand() *cli.Command {
	var input publishInput
	return &cli.Command{
		Name:      "publish",
		Usage:     "Publish a package artifact privately by default",
		ArgsUsage: "[artifact]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: flagPublic, Usage: "Request public marketplace review after upload", Destination: &input.Public},
			&cli.StringFlag{Name: flagReason, Usage: "Reason for public marketplace review", Destination: &input.Reason},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			input.Artifact = cmd.Args().First()
			return handlePublish(ctx, input)
		},
	}
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List packages published by the active organization",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleList(ctx)
		},
	}
}

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "Show one published package release",
		ArgsUsage: "<release-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			return handleStatus(ctx, cmd.Args().First())
		},
	}
}
