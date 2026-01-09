package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
)

func MachineCommand() *cli.Command {
	return &cli.Command{
		Name:  "machine",
		Usage: "Manage machines and check availability",
		Subcommands: []*cli.Command{
			machineListCommand(),
			machineAvailableCommand(),
		},
	}
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

func printAvailableMachineDetails(machine *api.Machine) {
	utils.PrintStatusLine("Machine ID", machine.MachineID)
	utils.PrintStatusLine("Name", machine.MachineName)
	utils.PrintStatusLine("Region/Zone", fmt.Sprintf("%s/%s", machine.MachineRegion, machine.MachineZone))
	utils.PrintStatusLine("Status", machine.Status)

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
