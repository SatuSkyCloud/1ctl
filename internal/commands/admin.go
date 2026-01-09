package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v2"
)

func AdminCommand() *cli.Command {
	return &cli.Command{
		Name:  "admin",
		Usage: "Admin operations (super-admin only)",
		Subcommands: []*cli.Command{
			adminUsageCommand(),
			adminCreditsCommand(),
			adminNamespacesCommand(),
			adminClusterRolesCommand(),
			adminCleanupCommand(),
		},
	}
}

func adminUsageCommand() *cli.Command {
	return &cli.Command{
		Name:  "usage",
		Usage: "Manage machine usage records",
		Subcommands: []*cli.Command{
			{
				Name:   "unbilled",
				Usage:  "List unbilled machine usages",
				Action: handleAdminUsageUnbilled,
			},
			{
				Name:      "machine",
				Usage:     "Get usage for a specific machine",
				ArgsUsage: "<machine-id>",
				Action:    handleAdminUsageMachine,
			},
			{
				Name:      "bill",
				Usage:     "Mark usage as billed",
				ArgsUsage: "<usage-id>",
				Action:    handleAdminUsageBill,
			},
		},
	}
}

func adminCreditsCommand() *cli.Command {
	return &cli.Command{
		Name:  "credits",
		Usage: "Manage organization credits",
		Subcommands: []*cli.Command{
			{
				Name:      "add",
				Usage:     "Add credits to an organization",
				ArgsUsage: "<org-id>",
				Flags: []cli.Flag{
					&cli.Float64Flag{
						Name:     "amount",
						Usage:    "Amount to add",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "description",
						Usage: "Description for the transaction",
						Value: "Admin credit addition",
					},
				},
				Action: handleAdminCreditsAdd,
			},
			{
				Name:      "refund",
				Usage:     "Refund credits to an organization",
				ArgsUsage: "<org-id>",
				Flags: []cli.Flag{
					&cli.Float64Flag{
						Name:     "amount",
						Usage:    "Amount to refund",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "description",
						Usage: "Description for the transaction",
						Value: "Admin credit refund",
					},
				},
				Action: handleAdminCreditsRefund,
			},
		},
	}
}

func adminNamespacesCommand() *cli.Command {
	return &cli.Command{
		Name:   "namespaces",
		Usage:  "List all namespaces",
		Action: handleAdminNamespaces,
	}
}

func adminClusterRolesCommand() *cli.Command {
	return &cli.Command{
		Name:   "cluster-roles",
		Usage:  "List all cluster roles",
		Action: handleAdminClusterRoles,
	}
}

func adminCleanupCommand() *cli.Command {
	return &cli.Command{
		Name:  "cleanup",
		Usage: "Cleanup resources by label",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "label",
				Usage:    "Label selector for resources",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "namespace",
				Usage: "Namespace to cleanup (optional)",
			},
		},
		Action: handleAdminCleanup,
	}
}

func handleAdminUsageUnbilled(c *cli.Context) error {
	usages, err := api.GetUnbilledMachineUsages()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get unbilled usages: %s", err.Error()), nil)
	}

	if len(usages) == 0 {
		utils.PrintInfo("No unbilled machine usages found")
		return nil
	}

	utils.PrintHeader("Unbilled Machine Usages")
	for _, usage := range usages {
		utils.PrintStatusLine("Usage ID", usage.ID.String())
		utils.PrintStatusLine("Machine ID", usage.MachineID.String())
		utils.PrintStatusLine("Organization ID", usage.OrganizationID.String())
		utils.PrintStatusLine("Start Time", usage.StartTime.Format("2006-01-02 15:04:05"))
		if !usage.EndTime.IsZero() {
			utils.PrintStatusLine("End Time", usage.EndTime.Format("2006-01-02 15:04:05"))
		}
		utils.PrintStatusLine("Hours", fmt.Sprintf("%.2f", usage.Hours))
		utils.PrintStatusLine("Cost", fmt.Sprintf("$%.2f", usage.Cost))
		utils.PrintDivider()
	}
	return nil
}

func handleAdminUsageMachine(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("machine ID is required", nil)
	}

	machineID := c.Args().First()

	usages, err := api.GetMachineUsagesByMachineID(machineID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get machine usages: %s", err.Error()), nil)
	}

	if len(usages) == 0 {
		utils.PrintInfo("No usages found for this machine")
		return nil
	}

	utils.PrintHeader("Machine Usage Records")
	for _, usage := range usages {
		billedStatus := "No"
		if usage.Billed {
			billedStatus = fmt.Sprintf("Yes (%s)", usage.BilledAt.Format("2006-01-02"))
		}
		utils.PrintStatusLine("Usage ID", usage.ID.String())
		utils.PrintStatusLine("Start Time", usage.StartTime.Format("2006-01-02 15:04:05"))
		if !usage.EndTime.IsZero() {
			utils.PrintStatusLine("End Time", usage.EndTime.Format("2006-01-02 15:04:05"))
		}
		utils.PrintStatusLine("Hours", fmt.Sprintf("%.2f", usage.Hours))
		utils.PrintStatusLine("Cost", fmt.Sprintf("$%.2f", usage.Cost))
		utils.PrintStatusLine("Billed", billedStatus)
		utils.PrintDivider()
	}
	return nil
}

func handleAdminUsageBill(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("usage ID is required", nil)
	}

	usageID := c.Args().First()

	if err := api.MarkMachineUsageAsBilled(usageID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to mark usage as billed: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Machine usage marked as billed")
	return nil
}

func handleAdminCreditsAdd(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("organization ID is required", nil)
	}

	orgID := c.Args().First()
	amount := c.Float64("amount")
	description := c.String("description")

	if err := api.AdminAddCredits(orgID, amount, description); err != nil {
		return utils.NewError(fmt.Sprintf("failed to add credits: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Added $%.2f credits to organization %s", amount, orgID)
	return nil
}

func handleAdminCreditsRefund(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("organization ID is required", nil)
	}

	orgID := c.Args().First()
	amount := c.Float64("amount")
	description := c.String("description")

	if err := api.AdminRefundCredits(orgID, amount, description); err != nil {
		return utils.NewError(fmt.Sprintf("failed to refund credits: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Refunded $%.2f credits to organization %s", amount, orgID)
	return nil
}

func handleAdminNamespaces(c *cli.Context) error {
	namespaces, err := api.GetAdminNamespaces()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get namespaces: %s", err.Error()), nil)
	}

	if len(namespaces) == 0 {
		utils.PrintInfo("No namespaces found")
		return nil
	}

	utils.PrintHeader("Namespaces")
	for _, ns := range namespaces {
		utils.PrintStatusLine("Name", ns.Name)
		utils.PrintStatusLine("Phase", ns.Phase)
		utils.PrintStatusLine("Created", formatTimeAgo(ns.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleAdminClusterRoles(c *cli.Context) error {
	roles, err := api.GetAdminClusterRoles()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get cluster roles: %s", err.Error()), nil)
	}

	if len(roles) == 0 {
		utils.PrintInfo("No cluster roles found")
		return nil
	}

	utils.PrintHeader("Cluster Roles")
	for _, role := range roles {
		utils.PrintStatusLine("Name", role.Name)
		utils.PrintStatusLine("Created", formatTimeAgo(role.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleAdminCleanup(c *cli.Context) error {
	label := c.String("label")
	namespace := c.String("namespace")

	if err := api.AdminCleanupResources(label, namespace); err != nil {
		return utils.NewError(fmt.Sprintf("failed to cleanup resources: %s", err.Error()), nil)
	}

	if namespace != "" {
		utils.PrintSuccess("Resources with label '%s' cleaned up in namespace '%s'", label, namespace)
	} else {
		utils.PrintSuccess("Resources with label '%s' cleaned up", label)
	}
	return nil
}
