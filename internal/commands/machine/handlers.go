package machine

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

func handleMachineList(ctx context.Context) error {
	userID := satuskyctx.GetUserID()
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

	if utils.TryPrintJSON(machines) {
		return nil
	}

	headers := []string{"NAME", "MACHINE ID", "REGION/ZONE", "STATUS", "CPU", "MEM(GB)", "HOURLY COST"}
	rows := make([][]string, 0, len(machines))
	for _, machine := range machines {
		rows = append(rows, []string{
			machine.MachineName,
			machine.MachineID,
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

func handleMachineGet(ctx context.Context, in machineGetInput) error {
	machine, err := resolveMachineRef(in.MachineID)
	if err != nil {
		return err
	}
	if utils.TryPrintJSON(machine) {
		return nil
	}
	utils.PrintHeader("Machine Details")
	printMachineDetails(machine)
	return nil
}

func handleMachineCreate(ctx context.Context, in machineCreateInput) error {
	machine := machineFromInput(&in, nil)
	id, err := api.CreateMachine(machine)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create machine: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(map[string]interface{}{"id": id}) {
		return nil
	}
	utils.PrintSuccess("Machine created")
	utils.PrintStatusLine("ID", fmt.Sprintf("%d", id))
	return nil
}

func handleMachineUpdate(ctx context.Context, in machineUpdateInput) error {
	machine, err := resolveMachineRef(in.MachineID)
	if err != nil {
		return err
	}
	updated := machineFromInput(&in.machineCreateInput, machine)
	if err := api.UpdateMachine(machine.MachineID, updated); err != nil {
		return utils.NewError(fmt.Sprintf("failed to update machine: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Machine updated")
	utils.PrintStatusLine("Machine ID", machine.MachineID)
	return nil
}

func handleMachineDelete(ctx context.Context, in machineDeleteInput) error {
	machine, err := resolveMachineRef(in.MachineID)
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Decommission machine %s (%s)?", machine.MachineName, machine.MachineID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteMachine(machine.MachineID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete machine: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Machine decommissioned")
	return nil
}

func handleMachineInspect(ctx context.Context, in machineInspectInput) error {
	machine, err := resolveMachineRef(in.MachineID)
	if err != nil {
		return err
	}
	var hardware map[string]interface{}
	if in.Refresh {
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

func handleMachineLogs(ctx context.Context, in machineLogsInput) error {
	machine, err := resolveMachineRef(in.MachineID)
	if err != nil {
		return err
	}
	sources := in.Sources
	if len(sources) == 0 {
		sources = []string{"siderolink", "talos", "kubernetes"}
	}
	logs, err := api.FetchMachineLogs(machine.MachineID, api.MachineLogFetchRequest{
		Sources:        sources,
		TailLines:      in.Tail,
		Since:          in.Since,
		Filter:         in.Filter,
		Components:     in.Components,
		IncludePrevLog: in.Previous,
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

func handleMachineEvents(ctx context.Context, in machineEventsInput) error {
	machine, err := resolveMachineRef(in.MachineID)
	if err != nil {
		return err
	}
	events, err := api.GetMachineEvents(machine.MachineID, in.Tail)
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

func handleMachineAvailable(ctx context.Context, in machineAvailableInput) error {
	machines, err := api.GetAvailableMachines()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list available machines: %s", err.Error()), nil)
	}

	filteredMachines := filterMachines(machines, in)

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

func handleMachineUsageList(ctx context.Context) error {
	userID := satuskyctx.GetUserID()
	if userID == "" {
		return utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
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

func handleMachineUsageGet(ctx context.Context, in machineUsageIDInput) error {
	usage, err := api.GetMachineUsageByID(in.UsageID)
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

func handleMachineUsageCost(ctx context.Context, in machineUsageIDInput) error {
	cost, err := api.GetUsageCost(in.UsageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get usage cost: %s", err.Error()), nil)
	}

	utils.PrintHeader("Usage Cost")
	utils.PrintStatusLine("Usage ID", cost.UsageID)
	utils.PrintStatusLine("Total Cost", fmt.Sprintf("$%.4f", cost.Cost))
	return nil
}

// --- Shared helpers -----------------------------------------------------

func resolveMachineRef(ref string) (*api.Machine, error) {
	if ref == "" {
		return nil, utils.NewError("machine reference is required", nil)
	}
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return findOwnedMachine(func(machine api.Machine) bool { return machine.ID == id }, ref)
	}
	return findOwnedMachine(func(machine api.Machine) bool {
		return machine.MachineID == ref || machine.MachineName == ref
	}, ref)
}

func findOwnedMachine(match func(api.Machine) bool, ref string) (*api.Machine, error) {
	userID := satuskyctx.GetUserID()
	if userID == "" {
		return nil, utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
	}
	machines, err := api.GetMachinesByOwnerID(api.ToUUID(userID))
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

func machineFromInput(in *machineCreateInput, current *api.Machine) api.Machine {
	machine := api.Machine{}
	if current != nil {
		machine = *current
	}
	setStr(&machine.MachineName, in.Name)
	if current == nil || in.Types != nil {
		machine.MachineTypes = in.Types
	}
	setStr(&machine.MachineRegion, in.Region)
	setStr(&machine.MachineZone, in.Zone)
	setStr(&machine.IpAddr, in.IP)
	setStr(&machine.TalosVersion, in.TalosVersion)
	setStr(&machine.KubernetesVersion, in.K8sVersion)
	setInt(&machine.CPUCores, in.CPU)
	setInt(&machine.MemoryGB, in.Memory)
	setInt(&machine.StorageGB, in.Storage)
	setInt(&machine.GPUCount, in.GPUCount)
	setStr(&machine.GPUType, in.GPUType)
	setInt(&machine.BandwidthGbps, in.Bandwidth)
	setStr(&machine.Brand, in.Brand)
	setStr(&machine.Model, in.Model)
	setStr(&machine.Manufacturer, in.Manufacturer)
	setStr(&machine.FormFactor, in.FormFactor)
	setBool(&machine.Monetized, in.Monetized)
	setBool(&machine.Recommended, in.Recommended)
	setStr(&machine.PricingTier, in.PricingTier)
	setStr(&machine.OrganizationID, in.OrganizationID)
	return machine
}

func setStr(target *string, val string) {
	if val != "" {
		*target = val
	}
}

func setInt(target *int, val int) {
	if val != 0 {
		*target = val
	}
}

func setBool(target *bool, val bool) {
	if val {
		*target = val
	}
}

func filterMachines(machines []api.Machine, in machineAvailableInput) []api.Machine {
	var filtered []api.Machine
	for _, machine := range machines {
		if in.Region != "" && machine.MachineRegion != in.Region {
			continue
		}
		if in.Zone != "" && machine.MachineZone != in.Zone {
			continue
		}
		if in.MinCPU > 0 && machine.CPUCores < in.MinCPU {
			continue
		}
		if in.MinMemory > 0 && machine.MemoryGB < in.MinMemory {
			continue
		}
		if in.GPU && !machine.HasGPU {
			continue
		}
		if in.Recommended && !machine.Recommended {
			continue
		}
		if in.PricingTier != "" && machine.PricingTier != in.PricingTier {
			continue
		}
		filtered = append(filtered, machine)
	}
	return filtered
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

func printMapStatus(values map[string]interface{}) {
	for key, value := range values {
		utils.PrintStatusLine(key, fmt.Sprintf("%v", value))
	}
}
