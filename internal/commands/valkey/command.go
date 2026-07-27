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

type updateInput struct {
	StorageID        string
	Instances        int
	CPURequest       string
	CPULimit         string
	MemoryRequest    string
	MemoryLimit      string
	AppendOnly       bool
	NoAppendOnly     bool
	AppendFsync      string
	MaxmemoryPolicy  string
	MaxmemoryPercent int
	MetricsEnabled   bool
	NoMetrics        bool
}

type userMutationInput struct {
	StorageID            string
	Username             string
	AccessPreset         string
	KeyPatterns          []string
	ChannelPatterns      []string
	ClearKeyPatterns     bool
	ClearChannelPatterns bool
}

type confirmedUserInput struct {
	StorageID string
	Username  string
	Yes       bool
}

type confirmedStorageInput struct {
	StorageID string
	Yes       bool
}

type logsInput struct {
	StorageID string
	Tail      int
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
			updateCommand(),
			usersCommand(),
			rotateCredentialsCommand(),
			metricsCommand(),
			logsCommand(),
			redeployCommand(),
			restartCommand(),
			destroyCommand(),
		},
	}
}

func updateCommand() *cli.Command {
	var in updateInput
	return &cli.Command{
		Name:      "update",
		Usage:     "Update mutable managed Valkey settings",
		ArgsUsage: "<service>",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "instances", Usage: "Total instances within the existing topology", Destination: &in.Instances},
			&cli.StringFlag{Name: "cpu-request", Usage: "CPU request per instance", Destination: &in.CPURequest},
			&cli.StringFlag{Name: "cpu", Usage: "CPU limit per instance", Destination: &in.CPULimit},
			&cli.StringFlag{Name: "memory-request", Usage: "Memory request per instance", Destination: &in.MemoryRequest},
			&cli.StringFlag{Name: "memory", Usage: "Memory limit per instance", Destination: &in.MemoryLimit},
			&cli.BoolFlag{Name: "append-only", Usage: "Enable AOF durability", Destination: &in.AppendOnly},
			&cli.BoolFlag{Name: "no-append-only", Usage: "Disable AOF durability", Destination: &in.NoAppendOnly},
			&cli.StringFlag{Name: "append-fsync", Usage: "AOF fsync policy: always, everysec, or no", Destination: &in.AppendFsync},
			&cli.StringFlag{Name: "maxmemory-policy", Usage: "Valkey eviction policy", Destination: &in.MaxmemoryPolicy},
			&cli.IntFlag{Name: "maxmemory-percent", Usage: "Memory limit percentage available to Valkey data", Destination: &in.MaxmemoryPercent},
			&cli.BoolFlag{Name: "metrics", Usage: "Enable Prometheus metrics exporter", Destination: &in.MetricsEnabled},
			&cli.BoolFlag{Name: "no-metrics", Usage: "Disable Prometheus metrics exporter", Destination: &in.NoMetrics},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handleUpdate(ctx, in)
		},
	}
}

func usersCommand() *cli.Command {
	return &cli.Command{
		Name:  "users",
		Usage: "Manage preset-based Valkey ACL users",
		Commands: []*cli.Command{
			usersListCommand(),
			usersCreateCommand(),
			usersUpdateCommand(),
			usersDeleteCommand(),
			usersRotatePasswordCommand(),
		},
	}
}

func usersListCommand() *cli.Command {
	return storageCommand("list", "List custom Valkey ACL users", handleUsersList)
}

func usersCreateCommand() *cli.Command {
	var in userMutationInput
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a preset-based Valkey ACL user",
		ArgsUsage: "<service> <username>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "preset", Usage: "Access preset: admin, read_write, or read_only", Destination: &in.AccessPreset, Value: "read_write"},
			&cli.StringSliceFlag{Name: "key-pattern", Usage: "Allowed key glob without ACL prefix; repeatable", Destination: &in.KeyPatterns},
			&cli.StringSliceFlag{Name: "channel-pattern", Usage: "Allowed Pub/Sub channel glob without ACL prefix; repeatable", Destination: &in.ChannelPatterns},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().Get(0)
			in.Username = cmd.Args().Get(1)
			return handleUsersCreate(ctx, in)
		},
	}
}

func usersUpdateCommand() *cli.Command {
	var in userMutationInput
	return &cli.Command{
		Name:      "update",
		Usage:     "Update a custom Valkey ACL user's preset and patterns",
		ArgsUsage: "<service> <username>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "preset", Usage: "Access preset: admin, read_write, or read_only", Destination: &in.AccessPreset},
			&cli.StringSliceFlag{Name: "key-pattern", Usage: "Replace allowed key globs; repeatable", Destination: &in.KeyPatterns},
			&cli.StringSliceFlag{Name: "channel-pattern", Usage: "Replace allowed Pub/Sub channel globs; repeatable", Destination: &in.ChannelPatterns},
			&cli.BoolFlag{Name: "clear-key-patterns", Usage: "Restore the preset's default key scope", Destination: &in.ClearKeyPatterns},
			&cli.BoolFlag{Name: "clear-channel-patterns", Usage: "Restore the preset's default channel scope", Destination: &in.ClearChannelPatterns},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().Get(0)
			in.Username = cmd.Args().Get(1)
			return handleUsersUpdate(ctx, in)
		},
	}
}

func usersDeleteCommand() *cli.Command {
	var in confirmedUserInput
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a custom Valkey ACL user",
		ArgsUsage: "<service> <username>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt", Destination: &in.Yes},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().Get(0)
			in.Username = cmd.Args().Get(1)
			return handleUsersDelete(ctx, in)
		},
	}
}

func usersRotatePasswordCommand() *cli.Command {
	var in confirmedUserInput
	return &cli.Command{
		Name:      "rotate-password",
		Usage:     "Rotate a custom Valkey ACL user's password",
		ArgsUsage: "<service> <username>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt", Destination: &in.Yes},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().Get(0)
			in.Username = cmd.Args().Get(1)
			return handleUsersRotatePassword(ctx, in)
		},
	}
}

func rotateCredentialsCommand() *cli.Command {
	var in confirmedStorageInput
	return &cli.Command{
		Name:      "rotate-credentials",
		Usage:     "Rotate the protected default user's password",
		ArgsUsage: "<service>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt", Destination: &in.Yes},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handleRotateCredentials(ctx, in)
		},
	}
}

func metricsCommand() *cli.Command {
	return storageCommand("metrics", "Show current managed Valkey metrics", handleMetrics)
}

func logsCommand() *cli.Command {
	var in logsInput
	return &cli.Command{
		Name:      "logs",
		Usage:     "Show recent managed Valkey logs",
		ArgsUsage: "<service>",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "tail", Usage: "Number of recent lines (maximum 2000)", Destination: &in.Tail, Value: 200},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handleLogs(ctx, in)
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
