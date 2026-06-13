package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

func MachineCommand() *cli.Command {
	return &cli.Command{
		Name:  "machine",
		Usage: "Manage machines and check availability",
		Subcommands: []*cli.Command{
			machineListCommand(),
			machineGetCommand(),
			machineUpdateCommand(),
			machineVisibilityCommand(),
			machineLabelsCommand(),
			machineDeleteCommand(),
			machineInspectCommand(),
			machineLogsCommand(),
			machineEventsCommand(),
			machineAvailableCommand(),
			machineUsageCommand(),
		},
	}
}

func machineListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List machines visible to the current user",
		Action: handleListMachines,
	}
}

func machineGetCommand() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Show machine inventory details",
		ArgsUsage: "<machine-id|name|numeric-id>",
		Action:    handleGetMachine,
	}
}

func machineUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Update a machine inventory record",
		ArgsUsage: "<machine-id|name|numeric-id>",
		Flags:     machineMutationFlags(false),
		Action:    handleUpdateMachine,
	}
}

func machineVisibilityCommand() *cli.Command {
	return &cli.Command{
		Name:      "visibility",
		Usage:     "Set machine visibility",
		ArgsUsage: "<machine-id|name|numeric-id> <owner|organisation|public>",
		Action:    handleMachineVisibility,
	}
}

func machineLabelsCommand() *cli.Command {
	return &cli.Command{
		Name:  "labels",
		Usage: "Manage machine labels",
		Subcommands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List machine labels",
				ArgsUsage: "<machine-id|name|numeric-id>",
				Action:    handleMachineLabelsList,
			},
			{
				Name:      "set",
				Usage:     "Set machine labels",
				ArgsUsage: "<machine-id|name|numeric-id> <key=value>...",
				Action:    handleMachineLabelsSet,
			},
			{
				Name:      "remove",
				Usage:     "Remove machine labels",
				ArgsUsage: "<machine-id|name|numeric-id> <key>...",
				Action:    handleMachineLabelsRemove,
			},
		},
	}
}

func machineDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a machine inventory record",
		ArgsUsage: "<machine-id|name|numeric-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"},
		},
		Action: handleDeleteMachine,
	}
}

func machineInspectCommand() *cli.Command {
	return &cli.Command{
		Name:      "inspect",
		Usage:     "Inspect machine hardware, labels, and Talos status",
		ArgsUsage: "<machine-id|name|numeric-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "refresh", Usage: "Refresh hardware data before printing"},
		},
		Action: handleInspectMachine,
	}
}

func machineLogsCommand() *cli.Command {
	return &cli.Command{
		Name:      "logs",
		Usage:     "Fetch machine boot, Talos, and Kubernetes logs",
		ArgsUsage: "<machine-id|name|numeric-id>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "source", Usage: "Log source: siderolink, talos, kubernetes. Repeatable."},
			&cli.IntFlag{Name: "tail", Aliases: []string{"n"}, Usage: "Number of lines to fetch", Value: 100},
			&cli.StringFlag{Name: "since", Usage: "Fetch logs since duration or timestamp, e.g. 10m, 1h"},
			&cli.StringFlag{Name: "filter", Usage: "Text filter for returned log lines"},
			&cli.StringSliceFlag{Name: "component", Usage: "Component filter. Repeatable."},
			&cli.BoolFlag{Name: "previous", Usage: "Include previous container logs when available"},
		},
		Action: handleMachineLogs,
	}
}

func machineEventsCommand() *cli.Command {
	return &cli.Command{
		Name:      "events",
		Usage:     "Fetch recent machine runtime events",
		ArgsUsage: "<machine-id|name|numeric-id>",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "tail", Aliases: []string{"n"}, Usage: "Number of events to fetch", Value: 50},
		},
		Action: handleMachineEvents,
	}
}

func machineMutationFlags(create bool) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "name", Usage: "Machine name", Required: create},
		&cli.StringSliceFlag{Name: "type", Usage: "Machine type. Repeatable. Valid values include worker and storage.", Value: cli.NewStringSlice("worker")},
		&cli.StringFlag{Name: "region", Usage: "Machine region", Required: create},
		&cli.StringFlag{Name: "zone", Usage: "Machine zone", Required: create},
		&cli.StringFlag{Name: "ip", Usage: "Machine IP address"},
		&cli.StringFlag{Name: "talos-version", Usage: "Talos version"},
		&cli.StringFlag{Name: "kubernetes-version", Usage: "Kubernetes version"},
		&cli.IntFlag{Name: "cpu", Usage: "CPU cores", Required: create},
		&cli.IntFlag{Name: "memory", Usage: "Memory in GB", Required: create},
		&cli.IntFlag{Name: "storage", Usage: "Storage in GB", Required: create},
		&cli.IntFlag{Name: "gpu-count", Usage: "GPU count"},
		&cli.StringFlag{Name: "gpu-type", Usage: "GPU type"},
		&cli.IntFlag{Name: "bandwidth", Usage: "Bandwidth in Gbps"},
		&cli.StringFlag{Name: "brand", Usage: "Machine brand"},
		&cli.StringFlag{Name: "model", Usage: "Machine model"},
		&cli.StringFlag{Name: "manufacturer", Usage: "Machine manufacturer"},
		&cli.StringFlag{Name: "form-factor", Usage: "Machine form factor"},
		&cli.BoolFlag{Name: "monetized", Usage: "Make machine available for rent"},
		&cli.BoolFlag{Name: "recommended", Usage: "Mark machine as recommended"},
		&cli.StringFlag{Name: "pricing-tier", Usage: "Pricing tier", Value: "standard"},
		&cli.StringFlag{Name: "organization-id", Usage: "Organization ID for scoping the machine"},
	}
}

func handleListMachines(c *cli.Context) error {
	machines, err := api.GetVisibleMachines()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list machines: %s", err.Error()), nil)
	}

	if len(machines) == 0 {
		utils.PrintInfo("No machines found")
		return nil
	}

	if utils.TryPrintJSON(machines) {
		return nil
	}

	headers := []string{"NAME", "MACHINE ID", "VISIBILITY", "REGION/ZONE", "STATUS", "CPU", "MEM(GB)", "HOURLY COST"}
	rows := make([][]string, 0, len(machines))
	for _, machine := range machines {
		rows = append(rows, []string{
			machine.MachineName,
			machine.MachineID,
			machine.MachineVisibility,
			fmt.Sprintf("%s/%s", machine.MachineRegion, machine.MachineZone),
			machine.Status,
			fmt.Sprintf("%d", machine.CPUCores),
			fmt.Sprintf("%d", machine.MemoryGB),
			fmt.Sprintf("$%.4f", machine.HourlyCost),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleGetMachine(c *cli.Context) error {
	machine, err := resolveMachineRef(c.Args().First())
	if err != nil {
		return err
	}
	if utils.TryPrintJSON(machine) {
		return nil
	}
	utils.PrintHeader("Machine Details")
	printMachineDetails(machine)

	// Show deployed workloads for non-monetized machines (BYOA)
	if !machine.Monetized {
		deployments, depErr := api.ListDeployments()
		if depErr == nil {
			var matched []api.Deployment
			for _, dep := range deployments {
				for _, host := range dep.Hostnames {
					if host == machine.MachineID {
						matched = append(matched, dep)
						break
					}
				}
			}
			if len(matched) > 0 {
				utils.PrintHeader("Workloads (%d)", len(matched))
				for _, dep := range matched {
					utils.PrintStatusLine(dep.AppLabel, fmt.Sprintf("%s CPU, %s Memory", dep.CpuRequest, dep.MemoryRequest))
				}
			}
		}
	}

	// Show billing info for monetized machines
	if machine.Monetized {
		utils.PrintHeader("Billing")
		utils.PrintStatusLine("Hourly Rate", fmt.Sprintf("$%.4f/hr", machine.HourlyCost))
		utils.PrintStatusLine("Visibility", machine.MachineVisibility)
	}

	return nil
}

func handleUpdateMachine(c *cli.Context) error {
	machine, err := resolveMachineRef(c.Args().First())
	if err != nil {
		return err
	}
	updated := machineFromFlags(c, machine)
	if err := api.UpdateMachine(machine.MachineID, updated); err != nil {
		return utils.NewError(fmt.Sprintf("failed to update machine: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Machine updated")
	utils.PrintStatusLine("Machine ID", machine.MachineID)
	return nil
}

func handleMachineVisibility(c *cli.Context) error {
	if c.NArg() < 2 {
		return utils.NewError("usage: 1ctl machine visibility <machine-id|name|numeric-id> <owner|organisation|public>", nil)
	}
	machine, err := resolveMachineRef(c.Args().Get(0))
	if err != nil {
		return err
	}
	visibility := c.Args().Get(1)
	if !isValidMachineVisibility(visibility) {
		return utils.NewError("machine_visibility must be one of: owner, organisation, public", nil)
	}
	if err := api.UpdateMachineVisibility(machine.MachineID, visibility); err != nil {
		return utils.NewError(fmt.Sprintf("failed to update machine visibility: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Machine visibility updated")
	utils.PrintStatusLine("Machine ID", machine.MachineID)
	utils.PrintStatusLine("Visibility", visibility)
	return nil
}

func handleMachineLabelsList(c *cli.Context) error {
	machine, err := resolveMachineRef(c.Args().First())
	if err != nil {
		return err
	}
	labels, err := api.GetMachineLabels(machine.MachineID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list machine labels: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(labels) {
		return nil
	}
	if len(labels) == 0 {
		utils.PrintInfo("No machine labels found")
		return nil
	}
	printLabels(labels)
	return nil
}

func handleMachineLabelsSet(c *cli.Context) error {
	if c.NArg() < 2 {
		return utils.NewError("usage: 1ctl machine labels set <machine-id|name|numeric-id> <key=value>...", nil)
	}
	machine, err := resolveMachineRef(c.Args().Get(0))
	if err != nil {
		return err
	}
	labels, err := parseLabelAssignments(c.Args().Slice()[1:])
	if err != nil {
		return err
	}
	updated, err := api.UpdateMachineLabels(machine.MachineID, labels)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to update machine labels: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(updated) {
		return nil
	}
	utils.PrintSuccess("Machine labels updated")
	printLabels(updated)
	return nil
}

func handleMachineLabelsRemove(c *cli.Context) error {
	if c.NArg() < 2 {
		return utils.NewError("usage: 1ctl machine labels remove <machine-id|name|numeric-id> <key>...", nil)
	}
	machine, err := resolveMachineRef(c.Args().Get(0))
	if err != nil {
		return err
	}
	labels, err := api.GetMachineNodeLabels(machine.MachineID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to read machine labels: %s", err.Error()), nil)
	}
	for _, key := range c.Args().Slice()[1:] {
		delete(labels, normalizeMachineLabelKey(key))
	}
	updated, err := api.ReplaceMachineLabels(machine.MachineID, labels)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to remove machine labels: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(updated) {
		return nil
	}
	utils.PrintSuccess("Machine labels updated")
	printLabels(updated)
	return nil
}

func handleDeleteMachine(c *cli.Context) error {
	machine, err := resolveMachineRef(c.Args().First())
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Decommission machine %s (%s)?", machine.MachineName, machine.MachineID), c.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteMachine(machine.MachineID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete machine: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Machine decommissioned")
	return nil
}

func handleInspectMachine(c *cli.Context) error {
	machine, err := resolveMachineRef(c.Args().First())
	if err != nil {
		return err
	}
	var hardware map[string]interface{}
	if c.Bool("refresh") {
		hardware, err = api.RefreshMachineHardware(machine.MachineID)
	} else {
		hardware, err = api.GetMachineHardware(machine.MachineID)
	}
	if err != nil {
		utils.PrintWarning("Hardware inspection failed: %s", err.Error())
	}
	labels, labelsErr := api.GetMachineLabels(machine.MachineID)
	if labelsErr != nil {
		utils.PrintWarning("Label inspection failed: %s", labelsErr.Error())
	}
	talosStatus, statusErr := api.GetMachineTalosStatus(machine.MachineID)
	if statusErr != nil {
		utils.PrintWarning("Talos status inspection failed: %s", statusErr.Error())
	}
	details, detailsErr := api.GetMachineDetails(machine.MachineID)
	if detailsErr != nil {
		utils.PrintWarning("Detailed inspection failed: %s", detailsErr.Error())
	}

	result := map[string]interface{}{
		"machine":      machine,
		"hardware":     hardware,
		"labels":       labels,
		"talos_status": talosStatus,
		"details":      details,
	}
	if utils.TryPrintJSON(result) {
		return nil
	}

	utils.PrintHeader("Machine Inspection")
	printMachineDetails(machine)
	if len(labels) > 0 {
		utils.PrintHeader("Labels")
		for key, value := range labels {
			utils.PrintStatusLine(key, value)
		}
	}
	if len(talosStatus) > 0 {
		utils.PrintHeader("Talos Status")
		printMapStatus(talosStatus)
	}
	if len(hardware) > 0 {
		utils.PrintHeader("Hardware")
		printMapStatus(hardware)
	}
	if len(details) > 0 {
		utils.PrintHeader("Details")
		printMapStatus(details)
	}
	return nil
}

func handleMachineLogs(c *cli.Context) error {
	machine, err := resolveMachineRef(c.Args().First())
	if err != nil {
		return err
	}
	sources := c.StringSlice("source")
	if len(sources) == 0 {
		sources = []string{"siderolink", "talos", "kubernetes"}
	}
	logs, err := api.FetchMachineLogs(machine.MachineID, api.MachineLogFetchRequest{
		Sources:        sources,
		TailLines:      c.Int("tail"),
		Since:          c.String("since"),
		Filter:         c.String("filter"),
		Components:     c.StringSlice("component"),
		IncludePrevLog: c.Bool("previous"),
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to fetch machine logs: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(logs) {
		return nil
	}
	if len(logs.Entries) == 0 {
		utils.PrintInfo("No machine logs found")
		return nil
	}
	for _, entry := range logs.Entries {
		component := entry.Component
		if component == "" {
			component = "-"
		}
		fmt.Printf("[%s] [%s] [%s] %s\n", entry.Timestamp, entry.Source, component, entry.Message)
	}
	utils.PrintDivider()
	utils.PrintStatusLine("Stage", logs.Stage)
	utils.PrintStatusLine("Count", fmt.Sprintf("%d", logs.Count))
	return nil
}

func handleMachineEvents(c *cli.Context) error {
	machine, err := resolveMachineRef(c.Args().First())
	if err != nil {
		return err
	}
	events, err := api.GetMachineEvents(machine.MachineID, c.Int("tail"))
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to fetch machine events: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(events) {
		return nil
	}
	if len(events.Events) == 0 {
		utils.PrintInfo("No machine events found")
		return nil
	}
	headers := []string{"ID", "TYPE", "ACTOR", "NODE"}
	rows := make([][]string, 0, len(events.Events))
	for _, event := range events.Events {
		rows = append(rows, []string{event.ID, event.TypeURL, event.ActorID, event.Node})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func machineFromFlags(c *cli.Context, current *api.Machine) api.Machine {
	machine := api.Machine{}
	if current != nil {
		machine = *current
	}
	setStringFlag(c, "name", &machine.MachineName)
	if current == nil || c.IsSet("type") {
		machine.MachineTypes = c.StringSlice("type")
	}
	setStringFlag(c, "region", &machine.MachineRegion)
	setStringFlag(c, "zone", &machine.MachineZone)
	setStringFlag(c, "ip", &machine.IpAddr)
	setStringFlag(c, "talos-version", &machine.TalosVersion)
	setStringFlag(c, "kubernetes-version", &machine.KubernetesVersion)
	setIntFlag(c, "cpu", &machine.CPUCores)
	setIntFlag(c, "memory", &machine.MemoryGB)
	setIntFlag(c, "storage", &machine.StorageGB)
	setIntFlag(c, "gpu-count", &machine.GPUCount)
	setStringFlag(c, "gpu-type", &machine.GPUType)
	setIntFlag(c, "bandwidth", &machine.BandwidthGbps)
	setStringFlag(c, "brand", &machine.Brand)
	setStringFlag(c, "model", &machine.Model)
	setStringFlag(c, "manufacturer", &machine.Manufacturer)
	setStringFlag(c, "form-factor", &machine.FormFactor)
	setBoolFlag(c, "monetized", &machine.Monetized)
	setBoolFlag(c, "recommended", &machine.Recommended)
	setStringFlag(c, "pricing-tier", &machine.PricingTier)
	setStringFlag(c, "organization-id", &machine.OrganizationID)
	return machine
}

func resolveMachineRef(ref string) (*api.Machine, error) {
	if ref == "" {
		return nil, utils.NewError("machine reference is required", nil)
	}
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return findVisibleMachine(func(machine api.Machine) bool { return machine.ID == id }, ref)
	}
	return findVisibleMachine(func(machine api.Machine) bool {
		return machine.MachineID == ref || machine.MachineName == ref
	}, ref)
}

func findVisibleMachine(match func(api.Machine) bool, ref string) (*api.Machine, error) {
	machines, err := api.GetVisibleMachines()
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to resolve machine %q: %s", ref, err.Error()), nil)
	}
	for _, machine := range machines {
		if match(machine) {
			return &machine, nil
		}
	}
	return nil, utils.NewError(fmt.Sprintf("machine %q not found", ref), nil)
}

func isValidMachineVisibility(value string) bool {
	switch value {
	case "owner", "organisation", "public":
		return true
	default:
		return false
	}
}

func parseLabelAssignments(values []string) (map[string]string, error) {
	labels := make(map[string]string, len(values))
	for _, value := range values {
		key, labelValue, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, utils.NewError(fmt.Sprintf("invalid label %q: expected key=value", value), nil)
		}
		labels[normalizeMachineLabelKey(key)] = strings.ToLower(strings.TrimSpace(labelValue))
	}
	return labels, nil
}

func normalizeMachineLabelKey(key string) string {
	key = strings.TrimSpace(key)
	if !strings.Contains(key, "/") {
		return api.MachineTagLabelPrefix + key
	}
	return key
}

func printLabels(labels map[string]string) {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		utils.PrintStatusLine(key, labels[key])
	}
}

func setStringFlag(c *cli.Context, name string, target *string) {
	if c.IsSet(name) {
		*target = c.String(name)
	}
}

func setIntFlag(c *cli.Context, name string, target *int) {
	if c.IsSet(name) {
		*target = c.Int(name)
	}
}

func setBoolFlag(c *cli.Context, name string, target *bool) {
	if c.IsSet(name) {
		*target = c.Bool(name)
	}
}

func printMapStatus(values map[string]interface{}) {
	for key, value := range values {
		utils.PrintStatusLine(key, fmt.Sprintf("%v", value))
	}
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
	utils.PrintStatusLine("Visibility", machine.MachineVisibility)
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

func printAvailableMachineDetails(machine *api.Machine) {
	utils.PrintStatusLine("Machine ID", machine.MachineID)
	utils.PrintStatusLine("Name", machine.MachineName)
	utils.PrintStatusLine("Region/Zone", fmt.Sprintf("%s/%s", machine.MachineRegion, machine.MachineZone))
	utils.PrintStatusLine("Status", machine.Status)
	if machine.ConnectionMode != nil {
		utils.PrintStatusLine("Connection Mode", *machine.ConnectionMode)
	}
	utils.PrintStatusLine("CPU Cores", fmt.Sprintf("%d", machine.CPUCores))
	utils.PrintStatusLine("Memory (GB)", fmt.Sprintf("%d", machine.MemoryGB))
	utils.PrintStatusLine("Storage (GB)", fmt.Sprintf("%d", machine.StorageGB))
	if machine.HasGPU && machine.GPUCount > 0 {
		utils.PrintStatusLine("GPU", fmt.Sprintf("%d x %s", machine.GPUCount, machine.GPUType))
	} else {
		utils.PrintStatusLine("GPU", "None")
	}
	utils.PrintStatusLine("Bandwidth (Gbps)", fmt.Sprintf("%d", machine.BandwidthGbps))
	utils.PrintStatusLine("Node Type", machine.NodeType)
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

	headers := []string{"USAGE ID", "MACHINE ID", "STATUS", "HOURLY RATE", "START", "BILLED"}
	rows := make([][]string, 0, len(usages))
	for _, u := range usages {
		rows = append(rows, []string{
			u.UsageID,
			u.MachineID,
			u.Status,
			fmt.Sprintf("$%.4f/hr", u.HourlyRate),
			u.StartTime,
			fmt.Sprintf("%v", u.IsBilled),
		})
	}
	utils.PrintTable(headers, rows)
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
