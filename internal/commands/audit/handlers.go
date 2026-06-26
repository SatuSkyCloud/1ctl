package audit

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

func handleAuditList(ctx context.Context, in auditListInput) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	logs, err := api.GetAuditLogs(orgID, in.Limit, in.Action, in.User)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get audit logs: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(logs, "No audit logs found") {
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
		utils.PrintStatusLine("Time", utils.FormatTimeAgo(log.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleAuditGet(ctx context.Context, in auditGetInput) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	log, err := api.GetAuditLog(orgID, in.ID)
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
