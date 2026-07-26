// Package valkey defines the "1ctl valkey" command tree.
package valkey

import (
	"context"

	"github.com/urfave/cli/v3"
)

type createInput struct {
	Name             string
	Topology         string
	Instances        int
	Persistence      bool
	StorageSize      string
	StorageClass     string
	CPURequest       string
	CPULimit         string
	MemoryRequest    string
	MemoryLimit      string
	AppendOnly       bool
	AppendFsync      string
	MaxmemoryPolicy  string
	MaxmemoryPercent int
	MetricsEnabled   bool
}

type storageInput struct {
	StorageID string
}

type destroyInput struct {
	StorageID string
	Yes       bool
}

// Command returns the root Valkey command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "valkey",
		Aliases: []string{"vk"},
		Usage:   "Manage SatuSky managed Valkey services",
		Commands: []*cli.Command{
			createCommand(),
			listCommand(),
			getCommand(),
			statusCommand(),
			credentialsCommand(),
			redeployCommand(),
			restartCommand(),
			destroyCommand(),
		},
	}
}

func createCommand() *cli.Command {
	var in createInput
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a private managed Valkey service",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "topology", Usage: "Topology: standalone or replicated", Destination: &in.Topology, Value: "standalone"},
			&cli.IntFlag{Name: "instances", Usage: "Total instances (default: 1 standalone, 3 replicated)", Destination: &in.Instances},
			&cli.BoolFlag{Name: "persistence", Usage: "Persist data on retained volumes", Destination: &in.Persistence, Value: true},
			&cli.StringFlag{Name: "storage-size", Usage: "Data volume size", Destination: &in.StorageSize, Value: "8Gi"},
			&cli.StringFlag{Name: "storage-class", Usage: "Kubernetes storage class (default: auto-detect)", Destination: &in.StorageClass},
			&cli.StringFlag{Name: "cpu-request", Usage: "CPU request per instance", Destination: &in.CPURequest, Value: "250m"},
			&cli.StringFlag{Name: "cpu", Usage: "CPU limit per instance", Destination: &in.CPULimit, Value: "500m"},
			&cli.StringFlag{Name: "memory-request", Usage: "Memory request per instance", Destination: &in.MemoryRequest, Value: "512Mi"},
			&cli.StringFlag{Name: "memory", Usage: "Memory limit per instance", Destination: &in.MemoryLimit, Value: "1Gi"},
			&cli.BoolFlag{Name: "append-only", Usage: "Enable AOF durability", Destination: &in.AppendOnly, Value: true},
			&cli.StringFlag{Name: "append-fsync", Usage: "AOF fsync policy: always, everysec, or no", Destination: &in.AppendFsync, Value: "everysec"},
			&cli.StringFlag{Name: "maxmemory-policy", Usage: "Valkey eviction policy", Destination: &in.MaxmemoryPolicy, Value: "allkeys-lru"},
			&cli.IntFlag{Name: "maxmemory-percent", Usage: "Memory limit percentage available to Valkey data", Destination: &in.MaxmemoryPercent, Value: 75},
			&cli.BoolFlag{Name: "metrics", Usage: "Enable Prometheus metrics exporter", Destination: &in.MetricsEnabled, Value: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Name = cmd.Args().First()
			return handleCreate(ctx, in)
		},
	}
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List managed Valkey services",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return handleList(ctx)
		},
	}
}

func getCommand() *cli.Command {
	return storageCommand("get", "Show managed Valkey service details", handleGet)
}

func statusCommand() *cli.Command {
	return storageCommand("status", "Show live managed Valkey status", handleStatus)
}

func credentialsCommand() *cli.Command {
	return storageCommand("credentials", "Show private Valkey connection credentials", handleCredentials)
}

func redeployCommand() *cli.Command {
	cmd := storageCommand("redeploy", "Reconcile the managed Valkey service", handleRedeploy)
	cmd.Aliases = []string{"reconcile"}
	return cmd
}

func restartCommand() *cli.Command {
	return storageCommand("restart", "Roll the managed Valkey workload", handleRestart)
}

func storageCommand(name, usage string, handler func(context.Context, storageInput) error) *cli.Command {
	var in storageInput
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		ArgsUsage: "<service>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handler(ctx, in)
		},
	}
}

func destroyCommand() *cli.Command {
	var in destroyInput
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"destroy"},
		Usage:     "Destroy a managed Valkey service",
		ArgsUsage: "<service>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt", Destination: &in.Yes},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handleDestroy(ctx, in)
		},
	}
}
