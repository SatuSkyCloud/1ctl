// Package nats defines the "1ctl nats" managed marketplace facade.
package nats

import (
	"context"
	"regexp"

	"github.com/urfave/cli/v3"
)

var (
	natsNamePattern         = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	natsCPUPattern          = regexp.MustCompile(`^[1-9][0-9]*m$`)
	natsMemoryPattern       = regexp.MustCompile(`^[1-9][0-9]*(Mi|Gi)$`)
	natsStoragePattern      = regexp.MustCompile(`^[1-9][0-9]*(Mi|Gi|Ti)$`)
	natsStorageClassPattern = regexp.MustCompile(`^$|^[a-z0-9]([-.a-z0-9]*[a-z0-9])?$`)
)

const (
	flagJetStream    = "jetstream"
	flagCPU          = "cpu"
	flagMemory       = "memory"
	flagStorageSize  = "storage-size"
	flagStorageClass = "storage-class"
	flagOutputDir    = "output-dir"
	flagStdout       = "stdout"
	flagYes          = "yes"
	flagPurge        = "purge-retained"
	flagNoWait       = "no-wait"
)

type createInput struct {
	Name           string
	JetStream      bool
	CPU            string
	Memory         string
	StorageSize    string
	StorageSizeSet bool
	StorageClass   string
}

type deploymentInput struct {
	Deployment string
}

type credentialsInput struct {
	Deployment string
	OutputDir  string
	Stdout     bool
}

type deleteInput struct {
	Deployment string
	Yes        bool
	Purge      bool
	NoWait     bool
}

// Command returns the root NATS command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "nats",
		Usage: "Manage NATS marketplace deployments",
		Commands: []*cli.Command{
			createCommand(),
			listCommand(),
			getCommand(),
			statusCommand(),
			credentialsCommand(),
			deleteCommand(),
		},
	}
}

func createCommand() *cli.Command {
	var in createInput
	return &cli.Command{
		Name:      "create",
		Usage:     "Create core NATS or a three-node JetStream HA deployment",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: flagJetStream, Usage: "Enable the three-node JetStream HA profile", Destination: &in.JetStream},
			&cli.StringFlag{Name: flagCPU, Usage: "CPU request per replica", Value: "250m", Destination: &in.CPU},
			&cli.StringFlag{Name: flagMemory, Usage: "Memory request per replica", Value: "256Mi", Destination: &in.Memory},
			&cli.StringFlag{Name: flagStorageSize, Usage: "JetStream PVC size", Value: "10Gi", Destination: &in.StorageSize},
			&cli.StringFlag{Name: flagStorageClass, Usage: "JetStream PVC storage class", Destination: &in.StorageClass},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Name = cmd.Args().First()
			in.StorageSizeSet = cmd.IsSet(flagStorageSize)
			return handleCreate(ctx, in)
		},
	}
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List NATS deployments",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleList(ctx)
		},
	}
}

func getCommand() *cli.Command {
	var in deploymentInput
	return deploymentCommand("get", "Show NATS deployment details", func(ctx context.Context) error {
		return handleGet(ctx, in)
	}, &in)
}

func statusCommand() *cli.Command {
	var in deploymentInput
	return deploymentCommand("status", "Show live NATS deployment status", func(ctx context.Context) error {
		return handleStatus(ctx, in)
	}, &in)
}

func deploymentCommand(name, usage string, action func(context.Context) error, in *deploymentInput) *cli.Command {
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		ArgsUsage: "<deployment-name-or-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Deployment = cmd.Args().First()
			return action(ctx)
		},
	}
}

func credentialsCommand() *cli.Command {
	var in credentialsInput
	return &cli.Command{
		Name:      "credentials",
		Usage:     "Download NATS client credentials",
		ArgsUsage: "<deployment-name-or-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: flagOutputDir, Usage: "Secure directory for client-url.txt and client-token.txt", Destination: &in.OutputDir},
			&cli.BoolFlag{Name: flagStdout, Usage: "Reveal credentials on standard output", Destination: &in.Stdout},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Deployment = cmd.Args().First()
			return handleCredentials(ctx, in)
		},
	}
}

func deleteCommand() *cli.Command {
	var in deleteInput
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"destroy", "rm"},
		Usage:     "Delete a NATS deployment",
		ArgsUsage: "<deployment-name-or-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: flagYes, Aliases: []string{"y"}, Usage: "Skip confirmation", Destination: &in.Yes},
			&cli.BoolFlag{Name: flagPurge, Usage: "Also purge retained persistent volumes", Destination: &in.Purge},
			&cli.BoolFlag{Name: flagNoWait, Usage: "Return after deletion is accepted", Destination: &in.NoWait},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Deployment = cmd.Args().First()
			return handleDelete(ctx, in)
		},
	}
}
