package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func MachineCommand() *cli.Command {
	return &cli.Command{
		Name:  "machine",
		Usage: "Manage machines and check availability",
		Subcommands: []*cli.Command{
			machineListCommand(),
			machineAvailableCommand(),
			machineVMCommand(),
			machineUsageCommand(),
		},
	}
}

// resolveMachine looks up a machine by UUID or name
func resolveMachine(nameOrID string) (*api.Machine, error) {
	if nameOrID == "" {
		return nil, utils.NewError("machine name or ID is required", nil)
	}
	if _, err := uuid.Parse(nameOrID); err == nil {
		return api.GetMachineByID(api.ToUUID(nameOrID))
	}
	return api.GetMachineByName(nameOrID)
}

func machineListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all machines owned by the current user",
		Action: handleListMachines,
	}
}

func handleListMachines(c *cli.Context) error {
	userID := context.GetUserID()
	if userID == "" {
		return utils.NewError("user ID not found in context", nil)
	}

	machines, err := api.GetMachinesByOwnerID(api.ToUUID(userID))
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list machines: %s", err.Error()), nil)
	}

	if len(machines) == 0 {
		utils.PrintInfo("No machines found")
		return nil
	}

	utils.PrintHeader("Machines")
	for _, machine := range machines {
		printMachineDetails(&machine)
		utils.PrintDivider()
	}
	return nil
}

func printMachineDetails(machine *api.Machine) {
	utils.PrintStatusLine("Machine ID", machine.MachineID)
	utils.PrintStatusLine("Name", machine.MachineName)
	utils.PrintStatusLine("Types", fmt.Sprintf("%v", strings.Join(machine.MachineTypes, ", ")))
	utils.PrintStatusLine("Region", machine.MachineRegion)
	utils.PrintStatusLine("Zone", machine.MachineZone)
	utils.PrintStatusLine("IP Address", machine.IpAddr)
	utils.PrintStatusLine("Status", machine.Status)
	if machine.ConnectionMode != nil {
		utils.PrintStatusLine("Connection Mode", *machine.ConnectionMode)
	}
	if machine.VMState != nil {
		utils.PrintStatusLine("VM State", colorVMState(*machine.VMState))
	}
	utils.PrintStatusLine("CPU Cores", fmt.Sprintf("%d", machine.CPUCores))
	utils.PrintStatusLine("Memory (GB)", fmt.Sprintf("%d", machine.MemoryGB))
	utils.PrintStatusLine("Storage (GB)", fmt.Sprintf("%d", machine.StorageGB))
	utils.PrintStatusLine("GPU Count", fmt.Sprintf("%d", machine.GPUCount))
	utils.PrintStatusLine("GPU Type", machine.GPUType)
	utils.PrintStatusLine("Bandwidth (Gbps)", fmt.Sprintf("%d", machine.BandwidthGbps))
	utils.PrintStatusLine("Brand", machine.Brand)
	utils.PrintStatusLine("Model", machine.Model)
	utils.PrintStatusLine("Manufacturer", machine.Manufacturer)
	utils.PrintStatusLine("Form Factor", machine.FormFactor)
	utils.PrintStatusLine("Node Type", machine.NodeType)
	utils.PrintStatusLine("Pricing Tier", machine.PricingTier)
	utils.PrintStatusLine("Hourly Cost", fmt.Sprintf("$%.4f", machine.HourlyCost))
	utils.PrintStatusLine("Monetized", fmt.Sprintf("%v", machine.Monetized))
	utils.PrintStatusLine("Recommended", fmt.Sprintf("%v", machine.Recommended))
	if machine.ResourceScore != nil {
		utils.PrintStatusLine("Resource Score", fmt.Sprintf("%.2f", *machine.ResourceScore))
	}
	if machine.UptimePercent != nil {
		utils.PrintStatusLine("Uptime", fmt.Sprintf("%.2f%%", *machine.UptimePercent))
	}
}

// colorVMState returns a colored string for the VM state
func colorVMState(state string) string {
	switch strings.ToLower(state) {
	case "running":
		return utils.SuccessColor(state)
	case "error":
		return utils.ErrorColor(state)
	case "stopped", "paused", "unknown":
		return utils.WarnColor(state)
	default:
		return state
	}
}

func machineAvailableCommand() *cli.Command {
	return &cli.Command{
		Name:   "available",
		Usage:  "List available machines for rent",
		Action: handleListAvailableMachines,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "region",
				Usage: "Filter by region (e.g., 'SG', 'US')",
			},
			&cli.StringFlag{
				Name:  "zone",
				Usage: "Filter by zone (e.g., 'sg-sgp-1', 'us-west-1')",
			},
			&cli.IntFlag{
				Name:  "min-cpu",
				Usage: "Minimum CPU cores required",
			},
			&cli.IntFlag{
				Name:  "min-memory",
				Usage: "Minimum memory in GB required",
			},
			&cli.BoolFlag{
				Name:  "gpu",
				Usage: "Show only machines with GPU",
			},
			&cli.BoolFlag{
				Name:  "recommended",
				Usage: "Show only recommended machines",
			},
			&cli.StringFlag{
				Name:  "pricing-tier",
				Usage: "Filter by pricing tier (e.g., 'basic', 'premium')",
			},
		},
	}
}

func handleListAvailableMachines(c *cli.Context) error {
	machines, err := api.GetAvailableMachines()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list available machines: %s", err.Error()), nil)
	}

	// Apply filters
	filteredMachines := filterMachines(machines, c)

	if len(filteredMachines) == 0 {
		utils.PrintInfo("No available machines found matching your criteria")
		return nil
	}

	utils.PrintHeader("Available Machines (%d found)", len(filteredMachines))
	for _, machine := range filteredMachines {
		printAvailableMachineDetails(&machine)
		utils.PrintDivider()
	}
	return nil
}

func filterMachines(machines []api.Machine, c *cli.Context) []api.Machine {
	var filtered []api.Machine

	for _, machine := range machines {
		// Apply region filter
		if c.IsSet("region") && machine.MachineRegion != c.String("region") {
			continue
		}

		// Apply zone filter
		if c.IsSet("zone") && machine.MachineZone != c.String("zone") {
			continue
		}

		// Apply minimum CPU filter
		if c.IsSet("min-cpu") && machine.CPUCores < c.Int("min-cpu") {
			continue
		}

		// Apply minimum memory filter
		if c.IsSet("min-memory") && machine.MemoryGB < c.Int("min-memory") {
			continue
		}

		// Apply GPU filter
		if c.Bool("gpu") && !machine.HasGPU {
			continue
		}

		// Apply recommended filter
		if c.Bool("recommended") && !machine.Recommended {
			continue
		}

		// Apply pricing tier filter
		if c.IsSet("pricing-tier") && machine.PricingTier != c.String("pricing-tier") {
			continue
		}

		filtered = append(filtered, machine)
	}

	return filtered
}

func machineVMCommand() *cli.Command {
	return &cli.Command{
		Name:  "vm",
		Usage: "Manage Mac agent VM lifecycle (start, stop, reboot, resize, etc.)",
		Subcommands: []*cli.Command{
			machineVMStatusCommand(),
			machineVMStartCommand(),
			machineVMStopCommand(),
			machineVMRebootCommand(),
			machineVMResizeCommand(),
			machineVMApplyConfigCommand(),
			machineVMConsoleCommand(),
		},
	}
}

func machineVMStatusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "Show VM state for a Mac agent machine",
		ArgsUsage: "<machine-name|id>",
		Action: func(c *cli.Context) error {
			nameOrID := c.Args().First()
			machine, err := resolveMachine(nameOrID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to get machine: %s", err.Error()), nil)
			}
			utils.PrintHeader("VM Status: %s", machine.MachineName)
			utils.PrintStatusLine("Machine ID", machine.MachineID)
			utils.PrintStatusLine("Status", machine.Status)
			if machine.ConnectionMode != nil {
				utils.PrintStatusLine("Connection Mode", *machine.ConnectionMode)
			}
			if machine.VMState != nil {
				utils.PrintStatusLine("VM State", colorVMState(*machine.VMState))
			} else {
				utils.PrintStatusLine("VM State", utils.WarnColor("unknown"))
			}
			return nil
		},
	}
}

func machineVMStartCommand() *cli.Command {
	return &cli.Command{
		Name:      "start",
		Usage:     "Start a Mac agent VM",
		ArgsUsage: "<machine-name|id>",
		Action: func(c *cli.Context) error {
			nameOrID := c.Args().First()
			machine, err := resolveMachine(nameOrID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to get machine: %s", err.Error()), nil)
			}
			resp, err := api.SendMachineCommand(machine.MachineID, api.SendCommandRequest{Type: api.CmdStart})
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to start VM: %s", err.Error()), nil)
			}
			utils.PrintSuccess("Start command sent (command_id: %s, status: %s)", resp.CommandID, resp.Status)
			return nil
		},
	}
}

func machineVMStopCommand() *cli.Command {
	return &cli.Command{
		Name:      "stop",
		Usage:     "Stop a Mac agent VM",
		ArgsUsage: "<machine-name|id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Force stop (no graceful shutdown)",
			},
			&cli.IntFlag{
				Name:  "grace-seconds",
				Usage: "Grace period in seconds before stop (default: 0)",
				Value: 0,
			},
		},
		Action: func(c *cli.Context) error {
			nameOrID := c.Args().First()
			machine, err := resolveMachine(nameOrID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to get machine: %s", err.Error()), nil)
			}
			req := api.SendCommandRequest{Type: api.CmdStop}
			if c.Bool("force") {
				req.Type = api.CmdForceStop
			} else if grace := c.Int("grace-seconds"); grace > 0 {
				req.Payload = map[string]interface{}{"grace_seconds": grace}
			}
			resp, err := api.SendMachineCommand(machine.MachineID, req)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to stop VM: %s", err.Error()), nil)
			}
			utils.PrintSuccess("Stop command sent (command_id: %s, status: %s)", resp.CommandID, resp.Status)
			return nil
		},
	}
}

func machineVMRebootCommand() *cli.Command {
	return &cli.Command{
		Name:      "reboot",
		Usage:     "Reboot a Mac agent VM",
		ArgsUsage: "<machine-name|id>",
		Action: func(c *cli.Context) error {
			nameOrID := c.Args().First()
			machine, err := resolveMachine(nameOrID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to get machine: %s", err.Error()), nil)
			}
			resp, err := api.SendMachineCommand(machine.MachineID, api.SendCommandRequest{Type: api.CmdReboot})
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to reboot VM: %s", err.Error()), nil)
			}
			utils.PrintSuccess("Reboot command sent (command_id: %s, status: %s)", resp.CommandID, resp.Status)
			return nil
		},
	}
}

func machineVMResizeCommand() *cli.Command {
	return &cli.Command{
		Name:      "resize",
		Usage:     "Resize a Mac agent VM (CPU and memory)",
		ArgsUsage: "<machine-name|id>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "cpu",
				Usage:    "Number of CPU cores",
				Required: true,
			},
			&cli.IntFlag{
				Name:     "memory",
				Usage:    "Memory in GB",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			nameOrID := c.Args().First()
			machine, err := resolveMachine(nameOrID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to get machine: %s", err.Error()), nil)
			}
			resp, err := api.SendMachineCommand(machine.MachineID, api.SendCommandRequest{
				Type: api.CmdResize,
				Payload: map[string]interface{}{
					"cpu_cores": c.Int("cpu"),
					"memory_gb": c.Int("memory"),
				},
			})
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to resize VM: %s", err.Error()), nil)
			}
			utils.PrintSuccess("Resize command sent (command_id: %s, status: %s)", resp.CommandID, resp.Status)
			return nil
		},
	}
}

func machineVMApplyConfigCommand() *cli.Command {
	return &cli.Command{
		Name:      "apply-config",
		Usage:     "Apply a Talos config to a Mac agent VM",
		ArgsUsage: "<machine-name|id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config-file",
				Usage:    "Path to Talos config file (will be base64-encoded)",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			nameOrID := c.Args().First()
			machine, err := resolveMachine(nameOrID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to get machine: %s", err.Error()), nil)
			}
			configBytes, err := os.ReadFile(c.String("config-file"))
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to read config file: %s", err.Error()), nil)
			}
			encoded := base64.StdEncoding.EncodeToString(configBytes)
			resp, err := api.SendMachineCommand(machine.MachineID, api.SendCommandRequest{
				Type:    api.CmdApplyConfig,
				Payload: map[string]interface{}{"config_base64": encoded},
			})
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to apply config: %s", err.Error()), nil)
			}
			utils.PrintSuccess("Apply-config command sent (command_id: %s, status: %s)", resp.CommandID, resp.Status)
			return nil
		},
	}
}

func machineVMConsoleCommand() *cli.Command {
	return &cli.Command{
		Name:      "console",
		Usage:     "Enable or disable console streaming on a Mac agent VM",
		ArgsUsage: "<machine-name|id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "enable",
				Usage: "Enable console streaming (default: disable)",
			},
		},
		Action: func(c *cli.Context) error {
			nameOrID := c.Args().First()
			machine, err := resolveMachine(nameOrID)
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to get machine: %s", err.Error()), nil)
			}
			resp, err := api.SendMachineCommand(machine.MachineID, api.SendCommandRequest{
				Type:    api.CmdStreamConsole,
				Payload: map[string]interface{}{"enabled": c.Bool("enable")},
			})
			if err != nil {
				return utils.NewError(fmt.Sprintf("failed to toggle console: %s", err.Error()), nil)
			}
			utils.PrintSuccess("Console command sent (command_id: %s, status: %s)", resp.CommandID, resp.Status)
			return nil
		},
	}
}

func printAvailableMachineDetails(machine *api.Machine) {
	utils.PrintStatusLine("Machine ID", machine.MachineID)
	utils.PrintStatusLine("Name", machine.MachineName)
	utils.PrintStatusLine("Region/Zone", fmt.Sprintf("%s/%s", machine.MachineRegion, machine.MachineZone))
	utils.PrintStatusLine("Status", machine.Status)
	if machine.ConnectionMode != nil {
		utils.PrintStatusLine("Connection Mode", *machine.ConnectionMode)
	}
	if machine.VMState != nil {
		utils.PrintStatusLine("VM State", colorVMState(*machine.VMState))
	}

	// Resource information
	utils.PrintStatusLine("CPU Cores", fmt.Sprintf("%d", machine.CPUCores))
	utils.PrintStatusLine("Memory (GB)", fmt.Sprintf("%d", machine.MemoryGB))
	utils.PrintStatusLine("Storage (GB)", fmt.Sprintf("%d", machine.StorageGB))

	// GPU information
	if machine.HasGPU && machine.GPUCount > 0 {
		utils.PrintStatusLine("GPU", fmt.Sprintf("%d x %s", machine.GPUCount, machine.GPUType))
	} else {
		utils.PrintStatusLine("GPU", "None")
	}

	// Network and performance
	utils.PrintStatusLine("Bandwidth (Gbps)", fmt.Sprintf("%d", machine.BandwidthGbps))
	utils.PrintStatusLine("Node Type", machine.NodeType)

	// Pricing and recommendations
	utils.PrintStatusLine("Pricing Tier", machine.PricingTier)
	utils.PrintStatusLine("Hourly Cost", fmt.Sprintf("$%.4f", machine.HourlyCost))

	if machine.Recommended {
		utils.PrintStatusLine("Status", "✅ Recommended")
	}

	if machine.ResourceScore != nil {
		utils.PrintStatusLine("Resource Score", fmt.Sprintf("%.2f", *machine.ResourceScore))
	}

	if machine.UptimePercent != nil {
		utils.PrintStatusLine("Uptime", fmt.Sprintf("%.2f%%", *machine.UptimePercent))
	}
}

// ============================================================
// Machine Usage Subcommands
// ============================================================

func machineUsageCommand() *cli.Command {
	return &cli.Command{
		Name:  "usage",
		Usage: "View machine usage records",
		Subcommands: []*cli.Command{
			machineUsageListCommand(),
			machineUsageGetCommand(),
			machineUsageCostCommand(),
		},
	}
}

func machineUsageListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List machine usage records for the current user",
		Action: handleMachineUsageList,
	}
}

func machineUsageGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get details for a specific usage record",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "usage-id",
				Usage:    "Usage record ID",
				Required: true,
			},
		},
		Action: handleMachineUsageGet,
	}
}

func machineUsageCostCommand() *cli.Command {
	return &cli.Command{
		Name:  "cost",
		Usage: "Calculate cost for a usage record",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "usage-id",
				Usage:    "Usage record ID",
				Required: true,
			},
		},
		Action: handleMachineUsageCost,
	}
}

func handleMachineUsageList(c *cli.Context) error {
	userID, err := requireUserContext()
	if err != nil {
		return err
	}

	usages, err := api.GetUserMachineUsages(userID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get machine usages: %s", err.Error()), nil)
	}

	if len(usages) == 0 {
		utils.PrintInfo("No machine usage records found")
		return nil
	}

	utils.PrintHeader("Machine Usage Records")
	for _, u := range usages {
		utils.PrintStatusLine("Usage ID", u.UsageID)
		utils.PrintStatusLine("Machine", u.MachineID)
		utils.PrintStatusLine("Deployment", u.DeploymentID)
		utils.PrintStatusLine("Status", u.Status)
		utils.PrintStatusLine("Hourly Rate", fmt.Sprintf("$%.4f/hr", u.HourlyRate))
		utils.PrintStatusLine("Start", u.StartTime)
		if u.EndTime != nil {
			utils.PrintStatusLine("End", *u.EndTime)
		}
		utils.PrintStatusLine("Billed", fmt.Sprintf("%v", u.IsBilled))
		utils.PrintDivider()
	}
	return nil
}

func handleMachineUsageGet(c *cli.Context) error {
	usageID := c.String("usage-id")

	usage, err := api.GetMachineUsageByID(usageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get usage record: %s", err.Error()), nil)
	}

	utils.PrintHeader("Machine Usage Record")
	utils.PrintStatusLine("Usage ID", usage.UsageID)
	utils.PrintStatusLine("Machine ID", usage.MachineID)
	utils.PrintStatusLine("Deployment ID", usage.DeploymentID)
	utils.PrintStatusLine("User ID", usage.UserID)
	utils.PrintStatusLine("Organization", usage.OrganizationID)
	utils.PrintStatusLine("Status", usage.Status)
	utils.PrintStatusLine("Hourly Rate", fmt.Sprintf("$%.4f/hr", usage.HourlyRate))
	utils.PrintStatusLine("Start Time", usage.StartTime)
	if usage.EndTime != nil {
		utils.PrintStatusLine("End Time", *usage.EndTime)
	}
	utils.PrintStatusLine("Billed", fmt.Sprintf("%v", usage.IsBilled))
	if usage.LastBilledAt != nil {
		utils.PrintStatusLine("Last Billed", *usage.LastBilledAt)
	}
	return nil
}

func handleMachineUsageCost(c *cli.Context) error {
	usageID := c.String("usage-id")

	cost, err := api.GetUsageCost(usageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get usage cost: %s", err.Error()), nil)
	}

	utils.PrintHeader("Usage Cost")
	utils.PrintStatusLine("Usage ID", cost.UsageID)
	utils.PrintStatusLine("Total Cost", fmt.Sprintf("$%.4f", cost.Cost))
	return nil
}
