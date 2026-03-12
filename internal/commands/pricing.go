package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v2"
)

// PricingCommand returns the root pricing command
func PricingCommand() *cli.Command {
	return &cli.Command{
		Name:    "pricing",
		Aliases: []string{"price"},
		Usage:   "View machine pricing configurations",
		Subcommands: []*cli.Command{
			pricingListCommand(),
			pricingGetCommand(),
			pricingLookupCommand(),
			pricingCalculateCommand(),
		},
	}
}

func pricingListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all pricing configurations",
		Action: handlePricingList,
	}
}

func pricingGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a pricing configuration by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config-id",
				Usage:    "Pricing config ID",
				Required: true,
			},
		},
		Action: handlePricingGet,
	}
}

func pricingLookupCommand() *cli.Command {
	return &cli.Command{
		Name:  "lookup",
		Usage: "Look up pricing for a region, machine type, and SLA tier",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "region",
				Usage:    "Region (e.g. my-kul-1b)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "type",
				Usage:    "Machine type (e.g. standard, premium)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "sla",
				Usage:    "SLA tier (e.g. standard, premium)",
				Required: true,
			},
		},
		Action: handlePricingLookup,
	}
}

func pricingCalculateCommand() *cli.Command {
	return &cli.Command{
		Name:  "calculate",
		Usage: "Calculate cost for a machine over a time range",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "machine-ref-id",
				Usage:    "Machine reference ID (numeric)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "machine-id",
				Usage:    "Machine ID (UUID)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "start",
				Usage:    "Start time (RFC3339 format, e.g. 2024-01-01T00:00:00Z)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "end",
				Usage:    "End time (RFC3339 format, e.g. 2024-01-02T00:00:00Z)",
				Required: true,
			},
		},
		Action: handlePricingCalculate,
	}
}

func handlePricingList(_ *cli.Context) error {
	configs, err := api.ListPricingConfigs()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list pricing configs: %s", err.Error()), nil)
	}

	if len(configs) == 0 {
		utils.PrintInfo("No pricing configurations found")
		return nil
	}

	utils.PrintHeader("Pricing Configurations")
	for _, c := range configs {
		printPricingConfig(&c)
		utils.PrintDivider()
	}
	return nil
}

func handlePricingGet(c *cli.Context) error {
	configID := c.String("config-id")

	config, err := api.GetPricingConfig(configID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get pricing config: %s", err.Error()), nil)
	}

	utils.PrintHeader("Pricing Configuration")
	printPricingConfig(config)
	return nil
}

func handlePricingLookup(c *cli.Context) error {
	region := c.String("region")
	machineType := c.String("type")
	sla := c.String("sla")

	config, err := api.GetPricingByRegionAndType(region, machineType, sla)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to look up pricing: %s", err.Error()), nil)
	}

	utils.PrintHeader("Pricing Configuration")
	printPricingConfig(config)
	return nil
}

func handlePricingCalculate(c *cli.Context) error {
	machineRefID := c.String("machine-ref-id")
	machineID := c.String("machine-id")
	req := api.CostCalculationRequest{
		StartTime: c.String("start"),
		EndTime:   c.String("end"),
	}

	result, err := api.CalculateMachineCost(machineRefID, machineID, req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to calculate cost: %s", err.Error()), nil)
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
