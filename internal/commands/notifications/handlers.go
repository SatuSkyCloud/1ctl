package notifications

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

// --- Handlers -----------------------------------------------------------

func handleNotifList(ctx context.Context, in notifListInput) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	notifications, err := api.GetNotifications(orgID, in.Unread, in.Limit)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get notifications: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(notifications, "No notifications found") {
		return nil
	}

	headers := []string{"TYPE", "SUBJECT", "PRIORITY", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(notifications))
	for _, notif := range notifications {
		readStatus := "unread"
		if notif.ReadAt != nil {
			readStatus = "read"
		}
		rows = append(rows, []string{
			notif.Type,
			notif.Subject,
			notif.Priority,
			readStatus,
			utils.FormatTimeAgo(notif.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleNotifCount(ctx context.Context) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	count, err := api.GetUnreadCount(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get unread count: %s", err.Error()), nil)
	}

	utils.PrintHeader("Unread Notifications")
	utils.PrintStatusLine("Count", fmt.Sprintf("%d", count))
	return nil
}

func handleNotifRead(ctx context.Context, in notifReadInput) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	if in.All {
		if err := api.MarkAllNotificationsAsRead(orgID); err != nil {
			return utils.NewError(fmt.Sprintf("failed to mark all as read: %s", err.Error()), nil)
		}
		utils.PrintSuccess("All notifications marked as read")
	} else {
		if err := api.MarkNotificationAsRead(orgID, in.ID); err != nil {
			return utils.NewError(fmt.Sprintf("failed to mark as read: %s", err.Error()), nil)
		}
		utils.PrintSuccess("Notification marked as read")
	}
	return nil
}

func handleNotifDelete(ctx context.Context, in notifDeleteInput) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	if !utils.Confirm(fmt.Sprintf("Delete notification %s?", in.ID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := api.DeleteNotification(orgID, in.ID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete notification: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Notification deleted successfully")
	return nil
}
