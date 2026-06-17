package commands

import (
	"context"
	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v3"
)

func AuditCommand() *cli.Command {
	return &cli.Command{
		Name:  "audit",
		Usage: "View audit logs",
		Commands: []*cli.Command{
			auditListCommand(),
			auditGetCommand(),
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

func handleAuditList(ctx context.Context, cmd *cli.Command) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	limit := cmd.Int("limit")
	action := cmd.String("action")
	userID := cmd.String("user")

	logs, err := api.GetAuditLogs(orgID, limit, action, userID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get audit logs: %s", err.Error()), nil)
	}

	if len(logs) == 0 {
		utils.PrintInfo("No audit logs found")
		return nil
	}

	if utils.TryPrintJSON(logs) {
		return nil
	}

	utils.PrintHeader("Audit Logs")
	for _, log := range logs {
		utils.PrintStatusLine("ID", log.ID.String())
		utils.PrintStatusLine("Action", log.Action)
		utils.PrintStatusLine("User", log.ActorEmail)
		utils.PrintStatusLine("Resource Type", log.ResourceType)
		if log.ResourceID != nil {
			utils.PrintStatusLine("Resource ID", log.ResourceID.String())
		}
		if log.IPAddress != "" {
			utils.PrintStatusLine("IP Address", log.IPAddress)
		}
		utils.PrintStatusLine("Time", formatTimeAgo(log.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleAuditGet(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("log ID is required", nil)
	}

	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	logID := cmd.Args().First()

	log, err := api.GetAuditLog(orgID, logID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get audit log: %s", err.Error()), nil)
	}

	if utils.TryPrintJSON(log) {
		return nil
	}

	utils.PrintHeader("Audit Log Details")
	utils.PrintStatusLine("ID", log.ID.String())
	utils.PrintStatusLine("Action", log.Action)
	utils.PrintStatusLine("User", log.ActorEmail)
	if log.ActorID != nil {
		utils.PrintStatusLine("User ID", log.ActorID.String())
	}
	utils.PrintStatusLine("Resource Type", log.ResourceType)
	if log.ResourceID != nil {
		utils.PrintStatusLine("Resource ID", log.ResourceID.String())
	}
	if log.IPAddress != "" {
		utils.PrintStatusLine("IP Address", log.IPAddress)
	}
	if log.UserAgent != "" {
		utils.PrintStatusLine("User Agent", log.UserAgent)
	}
	utils.PrintStatusLine("Time", log.CreatedAt.Format("2006-01-02 15:04:05"))

	if len(log.Metadata) > 0 {
		fmt.Println()
		utils.PrintHeader("Metadata")
		for key, value := range log.Metadata {
			utils.PrintStatusLine(key, fmt.Sprintf("%v", value))
		}
	}

	return nil
}
