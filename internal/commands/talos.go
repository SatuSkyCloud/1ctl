package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

func TalosCommand() *cli.Command {
	return &cli.Command{
		Name:  "talos",
		Usage: "Talos Linux configuration",
		Subcommands: []*cli.Command{
			talosGenerateCommand(),
			talosApplyCommand(),
			talosHistoryCommand(),
			talosNetworkCommand(),
		},
	}
}

func talosGenerateCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generate Talos configuration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "machine-id",
				Usage:    "Machine ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "cluster-name",
				Usage:    "Cluster name",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "role",
				Usage: "Machine role (controlplane, worker)",
				Value: "worker",
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output file path",
			},
		},
		Action: handleTalosGenerate,
	}
}

func talosApplyCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Usage: "Apply Talos configuration to a machine",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "machine-id",
				Usage:    "Machine ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "config-file",
				Usage:    "Path to config file",
				Required: true,
			},
		},
		Action: handleTalosApply,
	}
}

func talosHistoryCommand() *cli.Command {
	return &cli.Command{
		Name:      "history",
		Usage:     "View configuration history for a machine",
		ArgsUsage: "<machine-id>",
		Action:    handleTalosHistory,
	}
}

func talosNetworkCommand() *cli.Command {
	return &cli.Command{
		Name:      "network",
		Usage:     "View network info for a machine",
		ArgsUsage: "<machine-id>",
		Action:    handleTalosNetwork,
	}
}

func handleTalosGenerate(c *cli.Context) error {
	machineID := c.String("machine-id")
	clusterName := c.String("cluster-name")
	role := c.String("role")
	output := c.String("output")

	req := api.GenerateTalosConfigRequest{
		MachineID:   machineID,
		ClusterName: clusterName,
		Role:        role,
	}

	config, err := api.GenerateTalosConfig(req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to generate Talos config: %s", err.Error()), nil)
	}

	if output != "" {
		if err := os.WriteFile(output, []byte(config.ConfigData), 0600); err != nil {
			return utils.NewError(fmt.Sprintf("failed to write config file: %s", err.Error()), nil)
		}
		utils.PrintSuccess("Talos config written to %s", output)
	} else {
		utils.PrintHeader("Generated Talos Configuration")
		utils.PrintStatusLine("Config ID", config.ID.String())
		utils.PrintStatusLine("Cluster", config.ClusterName)
		utils.PrintStatusLine("Role", config.Role)
		utils.PrintStatusLine("Version", config.Version)
		utils.PrintStatusLine("Created", formatTimeAgo(config.CreatedAt))
		fmt.Println()
		utils.PrintInfo("Use --output flag to save config to a file")
	}

	return nil
}

func handleTalosApply(c *cli.Context) error {
	machineID := c.String("machine-id")
	configFile := c.String("config-file")

	configData, err := os.ReadFile(configFile) // #nosec G304 -- User-provided config file path is intentional
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to read config file: %s", err.Error()), nil)
	}

	req := api.ApplyTalosConfigRequest{
		MachineID:  machineID,
		ConfigData: string(configData),
	}

	if err := api.ApplyTalosConfig(req); err != nil {
		return utils.NewError(fmt.Sprintf("failed to apply Talos config: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Talos configuration applied successfully")
	return nil
}

func handleTalosHistory(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("machine ID is required", nil)
	}

	machineID := c.Args().First()

	history, err := api.GetTalosConfigHistory(machineID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get config history: %s", err.Error()), nil)
	}

	if len(history) == 0 {
		utils.PrintInfo("No configuration history found")
		return nil
	}

	utils.PrintHeader("Configuration History")
	for _, entry := range history {
		utils.PrintStatusLine("ID", entry.ID.String())
		utils.PrintStatusLine("Version", entry.Version)
		utils.PrintStatusLine("Applied At", entry.AppliedAt.Format("2006-01-02 15:04:05"))
		utils.PrintStatusLine("Applied By", entry.AppliedBy)
		utils.PrintStatusLine("Status", entry.Status)
		utils.PrintDivider()
	}
	return nil
}

func handleTalosNetwork(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("machine ID is required", nil)
	}

	machineID := c.Args().First()

	info, err := api.GetTalosNetworkInfo(machineID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get network info: %s", err.Error()), nil)
	}

	utils.PrintHeader("Network Information")
	utils.PrintStatusLine("Machine ID", info.MachineID.String())
	utils.PrintStatusLine("Hostname", info.Hostname)
	if len(info.Addresses) > 0 {
		utils.PrintStatusLine("Addresses", strings.Join(info.Addresses, ", "))
	}
	utils.PrintStatusLine("Default Gateway", info.DefaultGW)
	if len(info.DNS) > 0 {
		utils.PrintStatusLine("DNS Servers", strings.Join(info.DNS, ", "))
	}

	if len(info.Interfaces) > 0 {
		fmt.Println()
		utils.PrintHeader("Network Interfaces")
		for _, iface := range info.Interfaces {
			utils.PrintStatusLine("Name", iface.Name)
			utils.PrintStatusLine("MAC", iface.MAC)
			utils.PrintStatusLine("State", iface.State)
			if len(iface.Addresses) > 0 {
				utils.PrintStatusLine("Addresses", strings.Join(iface.Addresses, ", "))
			}
			utils.PrintDivider()
		}
	}

	return nil
}
