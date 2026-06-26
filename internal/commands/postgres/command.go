// Package postgres defines the "1ctl postgres" command tree.
package postgres

import (
	"context"
	"regexp"

	"github.com/urfave/cli/v3"
)

var postgresNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// --- Flag name constants ------------------------------------------------

const (
	flagName         = "name"
	flagDatabase     = "database"
	flagUser         = "user"
	flagVersion      = "version"
	flagInstances    = "instances"
	flagStorageSize  = "storage-size"
	flagStorageClass = "storage-class"
	flagWALSize      = "wal-storage-size"
	flagCPURequest   = "cpu-request"
	flagCPULimit     = "cpu"
	flagMemRequest   = "memory-request"
	flagMemLimit     = "memory"
	flagYes          = "yes"
	flagBindAddr     = "bind-addr"
	flagLocalPort    = "local-port"
	flagSuperuser    = "superuser"
	flagCreatedb     = "createdb"
	flagCreaterole   = "createrole"
	flagReplication  = "replication"
	flagBypassRLS    = "bypassrls"
	flagComment      = "comment"
	flagCIDR         = "cidr"
	flagDescription  = "description"
)

// --- Input structs ------------------------------------------------------

type postgresCreateInput struct {
	Name         string
	Database     string
	User         string
	Version      string
	Instances    int
	StorageSize  string
	StorageClass string
	WALSize      string
	CPURequest   string
	CPULimit     string
	MemRequest   string
	MemLimit     string
}

type postgresStorageIDInput struct {
	StorageID string
}

type postgresDestroyInput struct {
	StorageID string
	Yes       bool
}

type postgresConnectInput struct {
	StorageID string
}

type postgresProxyInput struct {
	StorageID string
	BindAddr  string
	LocalPort string
}

type postgresUsersCreateInput struct {
	StorageID   string
	Username    string
	Superuser   bool
	Createdb    bool
	Createrole  bool
	Replication bool
	BypassRLS   bool
	Comment     string
}

type postgresUsersDeleteInput struct {
	StorageID string
	Username  string
	Yes       bool
}

type postgresFirewallAddInput struct {
	StorageID   string
	CIDR        string
	Description string
}

type postgresFirewallEnableInput struct {
	StorageID string
	RuleID    string
}

type postgresFirewallDisableInput struct {
	StorageID string
	RuleID    string
}

type postgresFirewallRemoveInput struct {
	StorageID string
	RuleID    string
	Yes       bool
}

// --- Flag constructors --------------------------------------------------

func optionalStringFlag(name, usage string, dest *string, def string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Value:       def,
	}
}

func optionalIntFlag(name, usage string, dest *int, def int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Value:       def,
	}
}

func optionalBoolFlag(name, usage string, dest *bool) *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
	}
}

// --- Command tree -------------------------------------------------------

// Command returns the root postgres command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "postgres",
		Aliases: []string{"pg"},
		Usage:   "Manage SatuSky managed Postgres clusters",
		Commands: []*cli.Command{
			postgresCreateCommand(),
			postgresListCommand(),
			postgresGetCommand(),
			postgresStatusCommand(),
			postgresCredentialsCommand(),
			postgresConnectCommand(),
			postgresProxyCommand(),
			postgresRedeployCommand(),
			postgresDestroyCommand(),
			postgresUsersCommand(),
			postgresFirewallCommand(),
			postgresStorageClassesCommand(),
		},
	}
}

func postgresCreateCommand() *cli.Command {
	var in postgresCreateInput
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a managed Postgres cluster",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			optionalStringFlag(flagDatabase, "Initial database name. Defaults to <name>", &in.Database, ""),
			optionalStringFlag(flagUser, "Initial database user", &in.User, "app"),
			optionalStringFlag(flagVersion, "Postgres major version", &in.Version, "17"),
			optionalIntFlag(flagInstances, "Number of database replicas", &in.Instances, 1),
			optionalStringFlag(flagStorageSize, "Data volume size", &in.StorageSize, "10Gi"),
			optionalStringFlag(flagStorageClass, "Kubernetes storage class (default: auto-detect)", &in.StorageClass, ""),
			optionalStringFlag(flagCPULimit, "CPU per replica (e.g., '1', '500m')", &in.CPULimit, "1"),
			optionalStringFlag(flagMemLimit, "Memory per replica (e.g., '1Gi', '512Mi')", &in.MemLimit, "1Gi"),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Name = cmd.Args().First()
			return handlePostgresCreate(ctx, in)
		},
	}
}

func postgresListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List managed Postgres clusters",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handlePostgresList(ctx)
		},
	}
}

func postgresGetCommand() *cli.Command {
	var in postgresStorageIDInput
	return &cli.Command{
		Name:      "get",
		Usage:     "Show managed Postgres cluster details",
		ArgsUsage: "<cluster>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handlePostgresGet(ctx, in)
		},
	}
}

func postgresStatusCommand() *cli.Command {
	var in postgresStorageIDInput
	return &cli.Command{
		Name:      "status",
		Usage:     "Show live managed Postgres status",
		ArgsUsage: "<cluster>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handlePostgresStatus(ctx, in)
		},
	}
}

func postgresCredentialsCommand() *cli.Command {
	var in postgresStorageIDInput
	return &cli.Command{
		Name:      "credentials",
		Usage:     "Show database connection credentials",
		ArgsUsage: "<cluster>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handlePostgresCredentials(ctx, in)
		},
	}
}

func postgresConnectCommand() *cli.Command {
	var in postgresConnectInput
	return &cli.Command{
		Name:      "connect",
		Usage:     "Connect to a managed Postgres cluster using psql",
		ArgsUsage: "<cluster>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handlePostgresConnect(ctx, in)
		},
	}
}

func postgresProxyCommand() *cli.Command {
	var in postgresProxyInput
	return &cli.Command{
		Name:      "proxy",
		Usage:     "Forward a local TCP port to a managed Postgres cluster",
		ArgsUsage: "<cluster>",
		Flags: []cli.Flag{
			optionalStringFlag(flagBindAddr, "Local address to bind", &in.BindAddr, "127.0.0.1"),
			optionalStringFlag(flagLocalPort, "Local port to listen on", &in.LocalPort, "15432"),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handlePostgresProxy(ctx, in)
		},
	}
}

func postgresRedeployCommand() *cli.Command {
	var in postgresStorageIDInput
	return &cli.Command{
		Name:      "redeploy",
		Usage:     "Re-apply CNPG resources for a managed Postgres cluster",
		ArgsUsage: "<cluster>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handlePostgresRedeploy(ctx, in)
		},
	}
}

func postgresDestroyCommand() *cli.Command {
	var in postgresDestroyInput
	return &cli.Command{
		Name:      "delete",
		Usage:     "Destroy a managed Postgres cluster",
		ArgsUsage: "<cluster>",
		Flags: []cli.Flag{
			optionalBoolFlag(flagYes, "Skip confirmation prompt", &in.Yes),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handlePostgresDestroy(ctx, in)
		},
	}
}

func postgresUsersCommand() *cli.Command {
	return &cli.Command{
		Name:    "users",
		Aliases: []string{"user"},
		Usage:   "Manage database users",
		Commands: []*cli.Command{
			postgresUsersListCommand(),
			postgresUsersCreateCommand(),
			postgresUsersDeleteCommand(),
		},
	}
}

func postgresUsersListCommand() *cli.Command {
	var in postgresStorageIDInput
	return &cli.Command{
		Name:      "list",
		Usage:     "List database users",
		ArgsUsage: "<cluster>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handlePostgresUsersList(ctx, in)
		},
	}
}

func postgresUsersCreateCommand() *cli.Command {
	var in postgresUsersCreateInput
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a database user",
		ArgsUsage: "<cluster> <username>",
		Flags: []cli.Flag{
			optionalBoolFlag(flagSuperuser, "Grant SUPERUSER", &in.Superuser),
			optionalBoolFlag(flagCreatedb, "Grant CREATEDB", &in.Createdb),
			optionalBoolFlag(flagCreaterole, "Grant CREATEROLE", &in.Createrole),
			optionalBoolFlag(flagReplication, "Grant REPLICATION", &in.Replication),
			optionalBoolFlag(flagBypassRLS, "Grant BYPASSRLS", &in.BypassRLS),
			optionalStringFlag(flagComment, "Role comment", &in.Comment, ""),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().Get(0)
			in.Username = cmd.Args().Get(1)
			return handlePostgresUsersCreate(ctx, in)
		},
	}
}

func postgresUsersDeleteCommand() *cli.Command {
	var in postgresUsersDeleteInput
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a database user",
		ArgsUsage: "<cluster> <username>",
		Flags: []cli.Flag{
			optionalBoolFlag(flagYes, "Skip confirmation prompt", &in.Yes),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().Get(0)
			in.Username = cmd.Args().Get(1)
			return handlePostgresUsersDelete(ctx, in)
		},
	}
}

func postgresFirewallCommand() *cli.Command {
	return &cli.Command{
		Name:    "firewall",
		Aliases: []string{"fw"},
		Usage:   "Manage Postgres firewall rules",
		Commands: []*cli.Command{
			postgresFirewallListCommand(),
			postgresFirewallAddCommand(),
			postgresFirewallEnableCommand(),
			postgresFirewallDisableCommand(),
			postgresFirewallRemoveCommand(),
		},
	}
}

func postgresFirewallListCommand() *cli.Command {
	var in postgresStorageIDInput
	return &cli.Command{
		Name:      "list",
		Usage:     "List firewall rules",
		ArgsUsage: "<cluster>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().First()
			return handlePostgresFirewallList(ctx, in)
		},
	}
}

func postgresFirewallAddCommand() *cli.Command {
	var in postgresFirewallAddInput
	return &cli.Command{
		Name:      "add",
		Usage:     "Add a firewall rule",
		ArgsUsage: "<cluster> <cidr>",
		Flags: []cli.Flag{
			optionalStringFlag(flagDescription, "Rule description", &in.Description, "CLI rule"),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().Get(0)
			in.CIDR = cmd.Args().Get(1)
			return handlePostgresFirewallAdd(ctx, in)
		},
	}
}

func postgresFirewallEnableCommand() *cli.Command {
	var in postgresFirewallEnableInput
	return &cli.Command{
		Name:      "enable",
		Usage:     "Enable a firewall rule",
		ArgsUsage: "<cluster> <rule-id>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().Get(0)
			in.RuleID = cmd.Args().Get(1)
			return handlePostgresFirewallSetEnabled(ctx, in.StorageID, in.RuleID, true)
		},
	}
}

func postgresFirewallDisableCommand() *cli.Command {
	var in postgresFirewallDisableInput
	return &cli.Command{
		Name:      "disable",
		Usage:     "Disable a firewall rule",
		ArgsUsage: "<cluster> <rule-id>",
		Flags:     []cli.Flag{},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().Get(0)
			in.RuleID = cmd.Args().Get(1)
			return handlePostgresFirewallSetEnabled(ctx, in.StorageID, in.RuleID, false)
		},
	}
}

func postgresFirewallRemoveCommand() *cli.Command {
	var in postgresFirewallRemoveInput
	return &cli.Command{
		Name:      "delete",
		Usage:     "Remove a firewall rule",
		ArgsUsage: "<cluster> <rule-id>",
		Flags: []cli.Flag{
			optionalBoolFlag(flagYes, "Skip confirmation prompt", &in.Yes),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.StorageID = cmd.Args().Get(0)
			in.RuleID = cmd.Args().Get(1)
			return handlePostgresFirewallRemove(ctx, in)
		},
	}
}

func postgresStorageClassesCommand() *cli.Command {
	return &cli.Command{
		Name:  "storage-classes",
		Usage: "List available Kubernetes storage classes",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handlePostgresStorageClasses(ctx)
		},
	}
}
