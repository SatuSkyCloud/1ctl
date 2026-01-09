package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func AuditCommand() *cli.Command {
	return &cli.Command{
		Name:  "audit",
		Usage: "View audit logs",
		Subcommands: []*cli.Command{
			auditListCommand(),
			auditGetCommand(),
			auditExportCommand(),
		},
	}
}

func auditListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List audit logs",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Number of logs to show",
				Value: 20,
			},
			&cli.StringFlag{
				Name:  "action",
				Usage: "Filter by action type",
			},
			&cli.StringFlag{
				Name:  "user",
				Usage: "Filter by user ID",
			},
		},
		Action: handleAuditList,
	}
}

func auditGetCommand() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Get audit log details",
		ArgsUsage: "<log-id>",
		Action:    handleAuditGet,
	}
}

func auditExportCommand() *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "Export audit logs",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "format",
				Usage: "Export format (json, csv)",
				Value: "json",
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output file path",
			},
		},
		Action: handleAuditExport,
	}
}

func handleAuditList(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	limit := c.Int("limit")
	action := c.String("action")
	userID := c.String("user")

	logs, err := api.GetAuditLogs(orgID, limit, action, userID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get audit logs: %s", err.Error()), nil)
	}

	if len(logs) == 0 {
		utils.PrintInfo("No audit logs found")
		return nil
	}

	utils.PrintHeader("Audit Logs")
	for _, log := range logs {
		utils.PrintStatusLine("ID", log.ID.String())
		utils.PrintStatusLine("Action", log.Action)
		utils.PrintStatusLine("User", log.UserEmail)
		utils.PrintStatusLine("Resource Type", log.ResourceType)
		if log.ResourceID != "" {
			utils.PrintStatusLine("Resource ID", log.ResourceID)
		}
		if log.IPAddress != "" {
			utils.PrintStatusLine("IP Address", log.IPAddress)
		}
		utils.PrintStatusLine("Time", formatTimeAgo(log.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleAuditGet(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("log ID is required", nil)
	}

	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	logID := c.Args().First()

	log, err := api.GetAuditLog(orgID, logID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get audit log: %s", err.Error()), nil)
	}

	utils.PrintHeader("Audit Log Details")
	utils.PrintStatusLine("ID", log.ID.String())
	utils.PrintStatusLine("Action", log.Action)
	utils.PrintStatusLine("User", log.UserEmail)
	utils.PrintStatusLine("User ID", log.UserID.String())
	utils.PrintStatusLine("Resource Type", log.ResourceType)
	if log.ResourceID != "" {
		utils.PrintStatusLine("Resource ID", log.ResourceID)
	}
	if log.IPAddress != "" {
		utils.PrintStatusLine("IP Address", log.IPAddress)
	}
	if log.UserAgent != "" {
		utils.PrintStatusLine("User Agent", log.UserAgent)
	}
	utils.PrintStatusLine("Time", log.CreatedAt.Format("2006-01-02 15:04:05"))

	if len(log.Details) > 0 {
		fmt.Println()
		utils.PrintHeader("Details")
		for key, value := range log.Details {
			utils.PrintStatusLine(key, fmt.Sprintf("%v", value))
		}
	}

	return nil
}

func handleAuditExport(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	format := c.String("format")
	output := c.String("output")

	data, err := api.ExportAuditLogs(orgID, format)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to export audit logs: %s", err.Error()), nil)
	}

	if output != "" {
		if err := os.WriteFile(output, data, 0644); err != nil {
			return utils.NewError(fmt.Sprintf("failed to write file: %s", err.Error()), nil)
		}
		utils.PrintSuccess("Audit logs exported to %s", output)
	} else {
		fmt.Println(string(data))
	}

	return nil
}
