package pricing

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	"1ctl/internal/utils"
)

// --- Handlers -----------------------------------------------------------
// All handlers receive only their input struct — no *cli.Command, no
// cmd.String(). Pure business logic, trivially testable with just the
// struct value.  No urfave/cli import anywhere in this file.

func handlePricingList(ctx context.Context) error {
	configs, err := api.ListPricingConfigs()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list pricing configs: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(configs, "No pricing configurations found") {
		return nil
	}

	utils.PrintHeader("Pricing Configurations")
	for _, c := range configs {
		printPricingConfig(&c)
		utils.PrintDivider()
	}
	return nil
}

func handlePricingGet(ctx context.Context, in pricingGetInput) error {
	config, err := api.GetPricingConfig(in.ConfigID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get pricing config: %s", err.Error()), nil)
	}

	if utils.TryPrintJSON(config) {
		return nil
	}

	utils.PrintHeader("Pricing Configuration")
	printPricingConfig(config)
	return nil
}

func handlePricingLookup(ctx context.Context, in pricingLookupInput) error {
	config, err := api.GetPricingByRegionAndType(in.Region, in.MachineType, in.SLA)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to look up pricing: %s", err.Error()), nil)
	}

	if utils.TryPrintJSON(config) {
		return nil
	}

	utils.PrintHeader("Pricing Configuration")
	printPricingConfig(config)
	return nil
}

func handlePricingCalculate(ctx context.Context, in pricingCalculateInput) error {
	req := api.CostCalculationRequest{
		StartTime: in.StartTime,
		EndTime:   in.EndTime,
	}

	result, err := api.CalculateMachineCost(in.MachineRefID, in.MachineID, req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to calculate cost: %s", err.Error()), nil)
	}

	if utils.TryPrintJSON(result) {
		return nil
	}

	utils.PrintHeader("Cost Calculation")
	utils.PrintStatusLine("Machine ID", result.MachineID)
	utils.PrintStatusLine("Start", result.StartTime)
	utils.PrintStatusLine("End", result.EndTime)
	utils.PrintStatusLine("Total Cost", fmt.Sprintf("$%.4f", result.TotalCost))
	fmt.Println()
	utils.PrintHeader("Breakdown")
	utils.PrintStatusLine("Base", fmt.Sprintf("$%.4f", result.Breakdown.BaseCost))
	utils.PrintStatusLine("CPU", fmt.Sprintf("$%.4f", result.Breakdown.CPUCost))
	utils.PrintStatusLine("Memory", fmt.Sprintf("$%.4f", result.Breakdown.MemoryCost))
	utils.PrintStatusLine("Storage", fmt.Sprintf("$%.4f", result.Breakdown.StorageCost))
	utils.PrintStatusLine("GPU", fmt.Sprintf("$%.4f", result.Breakdown.GPUCost))
	utils.PrintStatusLine("Bandwidth", fmt.Sprintf("$%.4f", result.Breakdown.BandwidthCost))
	return nil
}

// --- Shared helpers -----------------------------------------------------

func printPricingConfig(c *api.PricingConfig) {
	utils.PrintStatusLine("Config ID", c.ConfigID)
	utils.PrintStatusLine("Region", c.Region)
	utils.PrintStatusLine("Machine Type", c.MachineType)
	utils.PrintStatusLine("SLA Tier", c.SLATier)
	utils.PrintStatusLine("Base Hourly Rate", fmt.Sprintf("$%.4f/hr", c.BaseHourlyRate))
	utils.PrintStatusLine("CPU Rate", fmt.Sprintf("$%.4f/core/hr", c.CPUCoreRate))
	utils.PrintStatusLine("Memory Rate", fmt.Sprintf("$%.4f/GB/hr", c.MemoryGBRate))
	utils.PrintStatusLine("Storage Rate", fmt.Sprintf("$%.4f/GB/hr", c.StorageGBRate))
	if c.GPURate > 0 {
		utils.PrintStatusLine("GPU Rate", fmt.Sprintf("$%.4f/hr", c.GPURate))
	}
	if c.BandwidthGBPSRate > 0 {
		utils.PrintStatusLine("Bandwidth Rate", fmt.Sprintf("$%.4f/Gbps/hr", c.BandwidthGBPSRate))
	}
}
