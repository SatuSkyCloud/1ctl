// Package machine defines the "1ctl machine" command tree.
package machine

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagName         = "name"
	flagType         = "type"
	flagRegion       = "region"
	flagZone         = "zone"
	flagIP           = "ip"
	flagTalosVersion = "talos-version"
	flagK8sVersion   = "kubernetes-version"
	flagCPU          = "cpu"
	flagMemory       = "memory"
	flagStorage      = "storage"
	flagGPUCount     = "gpu-count"
	flagGPUType      = "gpu-type"
	flagBandwidth    = "bandwidth"
	flagBrand        = "brand"
	flagModel        = "model"
	flagManufacturer = "manufacturer"
	flagFormFactor   = "form-factor"
	flagMonetized    = "monetized"
	flagRecommended  = "recommended"
	flagPricingTier  = "pricing-tier"
	flagOrganization = "organization-id"
	flagYes          = "yes"
	flagRefresh      = "refresh"
	flagSource       = "source"
	flagTail         = "tail"
	flagSince        = "since"
	flagFilter       = "filter"
	flagComponent    = "component"
	flagPrev         = "previous"
	flagMinCPU       = "min-cpu"
	flagMinMemory    = "min-memory"
	flagGPU          = "gpu"
	flagUsageID      = "usage-id"
	flagKey          = "key"
	flagValue        = "value"
)

// --- Input structs ------------------------------------------------------

type machineGetInput struct {
	MachineID string
}

type machineCreateInput struct {
	Name           string
	Types          []string
	Region         string
	Zone           string
	IP             string
	TalosVersion   string
	K8sVersion     string
	CPU            int
	Memory         int
	Storage        int
	GPUCount       int
	GPUType        string
	Bandwidth      int
	Brand          string
	Model          string
	Manufacturer   string
	FormFactor     string
	Monetized      bool
	Recommended    bool
	PricingTier    string
	OrganizationID string
}

type machineUpdateInput struct {
	machineCreateInput
	MachineID string
}

type machineDeleteInput struct {
	MachineID string
	Yes       bool
}

type machineInspectInput struct {
	MachineID string
	Refresh   bool
}

type machineLogsInput struct {
	MachineID  string
	Sources    []string
	Tail       int
	Since      string
	Filter     string
	Components []string
	Previous   bool
}

type machineEventsInput struct {
	MachineID string
	Tail      int
}

type machineAvailableInput struct {
	Region      string
	Zone        string
	MinCPU      int
	MinMemory   int
	GPU         bool
	Recommended bool
	PricingTier string
}

type machineUsageIDInput struct {
	UsageID string
}

// --- Flag constructors --------------------------------------------------

func optionalStringFlag(name, usage string, dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
	}
}

func optionalIntFlag(name, usage string, dest *int, value int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
		Value:       value,
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

func Command() *cli.Command {
	return &cli.Command{
		Name:  "machine",
		Usage: "Manage machines and check availability",
		Commands: []*cli.Command{
			machineListCommand(),
			machineGetCommand(),
			machineCreateCommand(),
			machineUpdateCommand(),
			machineDeleteCommand(),
			machineInspectCommand(),
			machineLogsCommand(),
			machineEventsCommand(),
			machineAvailableCommand(),
			machineUsageCommand(),
			machineLabelsCommand(),
		},
	}
}

func machineListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all machines owned by the current user",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleMachineList(ctx)
		},
	}
}

func machineGetCommand() *cli.Command {
	var in machineGetInput
	return &cli.Command{
		Name:      "get",
		Usage:     "Show machine inventory details",
		ArgsUsage: "<machine-id-or-name>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.MachineID = cmd.Args().First()
			return handleMachineGet(ctx, in)
		},
	}
}

func machineCreateCommand() *cli.Command {
	var in machineCreateInput
	return &cli.Command{
		Name:  "create",
		Usage: "Create a machine inventory record",
		Flags: machineMutationFlags(true, &in),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleMachineCreate(ctx, in)
		},
	}
}

func machineUpdateCommand() *cli.Command {
	var in machineUpdateInput
	return &cli.Command{
		Name:      "update",
		Usage:     "Update a machine inventory record",
		ArgsUsage: "<machine-id>",
		Flags:     machineMutationFlags(false, &in.machineCreateInput),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.MachineID = cmd.Args().First()
			return handleMachineUpdate(ctx, in)
		},
	}
}

func machineDeleteCommand() *cli.Command {
	var in machineDeleteInput
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a machine inventory record",
		ArgsUsage: "<machine-id-or-name>",
		Flags: []cli.Flag{
			optionalBoolFlag(flagYes, "Skip confirmation prompt", &in.Yes),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.MachineID = cmd.Args().First()
			return handleMachineDelete(ctx, in)
		},
	}
}

func machineInspectCommand() *cli.Command {
	var in machineInspectInput
	return &cli.Command{
		Name:      "inspect",
		Usage:     "Inspect machine hardware, labels, and Talos status",
		ArgsUsage: "<machine-id-or-name>",
		Flags: []cli.Flag{
			optionalBoolFlag(flagRefresh, "Refresh hardware data before printing", &in.Refresh),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.MachineID = cmd.Args().First()
			return handleMachineInspect(ctx, in)
		},
	}
}

func machineLogsCommand() *cli.Command {
	var in machineLogsInput
	return &cli.Command{
		Name:      "logs",
		Usage:     "Fetch machine boot, Talos, and Kubernetes logs",
		ArgsUsage: "<machine-id-or-name>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: flagSource, Usage: "Log source: siderolink, talos, kubernetes. Repeatable.", Destination: &in.Sources},
			optionalIntFlag(flagTail, "Number of lines to fetch", &in.Tail, 100),
			optionalStringFlag(flagSince, "Fetch logs since duration or timestamp, e.g. 10m, 1h", &in.Since),
			optionalStringFlag(flagFilter, "Text filter for returned log lines", &in.Filter),
			&cli.StringSliceFlag{Name: flagComponent, Usage: "Component filter. Repeatable.", Destination: &in.Components},
			optionalBoolFlag(flagPrev, "Include previous container logs when available", &in.Previous),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.MachineID = cmd.Args().First()
			return handleMachineLogs(ctx, in)
		},
	}
}

func machineEventsCommand() *cli.Command {
	var in machineEventsInput
	return &cli.Command{
		Name:      "events",
		Usage:     "Fetch recent machine runtime events",
		ArgsUsage: "<machine-id-or-name>",
		Flags: []cli.Flag{
			optionalIntFlag(flagTail, "Number of events to fetch", &in.Tail, 50),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.MachineID = cmd.Args().First()
			return handleMachineEvents(ctx, in)
		},
	}
}

func machineAvailableCommand() *cli.Command {
	var in machineAvailableInput
	return &cli.Command{
		Name:  "available",
		Usage: "List available machines for rent",
		Flags: []cli.Flag{
			optionalStringFlag(flagRegion, "Filter by region (e.g., 'SG', 'US')", &in.Region),
			optionalStringFlag(flagZone, "Filter by zone (e.g., 'sg-sgp-1', 'us-west-1')", &in.Zone),
			optionalIntFlag(flagMinCPU, "Minimum CPU cores required", &in.MinCPU, 0),
			optionalIntFlag(flagMinMemory, "Minimum memory in GB required", &in.MinMemory, 0),
			optionalBoolFlag(flagGPU, "Show only machines with GPU", &in.GPU),
			optionalBoolFlag(flagRecommended, "Show only recommended machines", &in.Recommended),
			optionalStringFlag(flagPricingTier, "Filter by pricing tier (e.g., 'basic', 'premium')", &in.PricingTier),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleMachineAvailable(ctx, in)
		},
	}
}

func machineUsageCommand() *cli.Command {
	return &cli.Command{
		Name:  "usage",
		Usage: "View machine usage records",
		Commands: []*cli.Command{
			machineUsageListCommand(),
			machineUsageGetCommand(),
			machineUsageCostCommand(),
		},
	}
}

func machineUsageListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List machine usage records for the current user",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleMachineUsageList(ctx)
		},
	}
}

func machineUsageGetCommand() *cli.Command {
	var in machineUsageIDInput
	return &cli.Command{
		Name:      "get",
		Usage:     "Get details for a specific usage record",
		ArgsUsage: "<usage-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.UsageID = cmd.Args().First()
			return handleMachineUsageGet(ctx, in)
		},
	}
}

func machineUsageCostCommand() *cli.Command {
	var in machineUsageIDInput
	return &cli.Command{
		Name:      "cost",
		Usage:     "Calculate cost for a usage record",
		ArgsUsage: "<usage-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.UsageID = cmd.Args().First()
			return handleMachineUsageCost(ctx, in)
		},
	}
}

func machineLabelsCommand() *cli.Command {
	return &cli.Command{
		Name:    "labels",
		Aliases: []string{"label"},
		Usage:   "Manage machine labels (tags)",
		Description: `Manage satusky.com/* labels on your machines.

Labels are used with "1ctl deploy --machine-tag <tag>" to route deployments
to specific machines. When you set "satusky.com/production" on a machine,
deploying with "--machine-tag production" will schedule on that machine.

Examples:
   1ctl machine labels list <machine-id>
   1ctl machine labels set  <machine-id> environment=production tier=compute
   1ctl machine labels unset <machine-id> environment
   1ctl machine labels keys`,
		Commands: []*cli.Command{
			machineLabelsListCommand(),
			machineLabelsSetCommand(),
			machineLabelsUnsetCommand(),
			machineLabelsKeysCommand(),
		},
	}
}

func machineLabelsListCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "Show all labels on a machine",
		ArgsUsage: "<machine-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			return handleMachineLabelsList(ctx, cmd.Args().First())
		},
	}
}

func machineLabelsSetCommand() *cli.Command {
	return &cli.Command{
		Name:      "set",
		Usage:     "Set one or more labels on a machine",
		ArgsUsage: "<machine-id> <key=value> [key=value...]",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			return handleMachineLabelsSet(ctx, args[0], args[1:])
		},
	}
}

func machineLabelsUnsetCommand() *cli.Command {
	return &cli.Command{
		Name:      "unset",
		Aliases:   []string{"delete", "rm"},
		Usage:     "Remove a label from a machine",
		ArgsUsage: "<machine-id> <key>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) < 2 {
				return cli.ShowSubcommandHelp(cmd)
			}
			return handleMachineLabelsUnset(ctx, args[0], args[1])
		},
	}
}

func machineLabelsKeysCommand() *cli.Command {
	return &cli.Command{
		Name:  "keys",
		Usage: "List all available label keys across all machines",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleMachineLabelsKeys(ctx)
		},
	}
}

// --- Machine mutation flags factory -------------------------------------

func machineMutationFlags(create bool, in *machineCreateInput) []cli.Flag {
	return []cli.Flag{
		optionalStringFlag(flagName, "Machine name", &in.Name),
		&cli.StringSliceFlag{Name: flagType, Usage: "Machine type. Repeatable. Valid values include worker and storage.", Value: []string{"worker"}, Destination: &in.Types},
		optionalStringFlag(flagRegion, "Machine region", &in.Region),
		optionalStringFlag(flagZone, "Machine zone", &in.Zone),
		optionalStringFlag(flagIP, "Machine IP address", &in.IP),
		optionalStringFlag(flagTalosVersion, "Talos version", &in.TalosVersion),
		optionalStringFlag(flagK8sVersion, "Kubernetes version", &in.K8sVersion),
		optionalIntFlag(flagCPU, "CPU cores", &in.CPU, 0),
		optionalIntFlag(flagMemory, "Memory in GB", &in.Memory, 0),
		optionalIntFlag(flagStorage, "Storage in GB", &in.Storage, 0),
		optionalIntFlag(flagGPUCount, "GPU count", &in.GPUCount, 0),
		optionalStringFlag(flagGPUType, "GPU type", &in.GPUType),
		optionalIntFlag(flagBandwidth, "Bandwidth in Gbps", &in.Bandwidth, 0),
		optionalStringFlag(flagBrand, "Machine brand", &in.Brand),
		optionalStringFlag(flagModel, "Machine model", &in.Model),
		optionalStringFlag(flagManufacturer, "Machine manufacturer", &in.Manufacturer),
		optionalStringFlag(flagFormFactor, "Machine form factor", &in.FormFactor),
		optionalBoolFlag(flagMonetized, "Make machine available for rent", &in.Monetized),
		optionalBoolFlag(flagRecommended, "Mark machine as recommended", &in.Recommended),
		optionalStringFlag(flagPricingTier, "Pricing tier", &in.PricingTier),
		optionalStringFlag(flagOrganization, "Organization ID for scoping the machine", &in.OrganizationID),
	}
}
