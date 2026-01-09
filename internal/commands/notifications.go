package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v2"
)

func NotificationsCommand() *cli.Command {
	return &cli.Command{
		Name:    "notifications",
		Aliases: []string{"notif"},
		Usage:   "Manage notifications",
		Subcommands: []*cli.Command{
			notifListCommand(),
			notifCountCommand(),
			notifReadCommand(),
			notifDeleteCommand(),
		},
	}
}

func notifListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List notifications",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "unread",
				Usage: "Show only unread notifications",
			},
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Number of notifications to show",
				Value: 20,
			},
		},
		Action: handleNotifList,
	}
}

func notifCountCommand() *cli.Command {
	return &cli.Command{
		Name:   "count",
		Usage:  "Show unread notification count",
		Action: handleNotifCount,
	}
}

func notifReadCommand() *cli.Command {
	return &cli.Command{
		Name:  "read",
		Usage: "Mark notification(s) as read",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "id",
				Usage: "Notification ID to mark as read",
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Mark all notifications as read",
			},
		},
		Action: handleNotifRead,
	}
}

func notifDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a notification",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "id",
				Usage:    "Notification ID to delete",
				Required: true,
			},
		},
		Action: handleNotifDelete,
	}
}

func handleNotifList(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	unreadOnly := c.Bool("unread")
	limit := c.Int("limit")

	notifications, err := api.GetNotifications(orgID, unreadOnly, limit)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get notifications: %s", err.Error()), nil)
	}

	if len(notifications) == 0 {
		utils.PrintInfo("No notifications found")
		return nil
	}

	utils.PrintHeader("Notifications")
	for _, notif := range notifications {
		readStatus := "unread"
		if notif.Read {
			readStatus = "read"
		}
		utils.PrintStatusLine("ID", notif.ID.String())
		utils.PrintStatusLine("Type", notif.Type)
		utils.PrintStatusLine("Title", notif.Title)
		utils.PrintStatusLine("Message", notif.Message)
		utils.PrintStatusLine("Status", readStatus)
		utils.PrintStatusLine("Created", formatTimeAgo(notif.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleNotifCount(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
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

func handleNotifRead(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	notifID := c.String("id")
	markAll := c.Bool("all")

	if !markAll && notifID == "" {
		return utils.NewError("either --id or --all is required", nil)
	}

	if markAll {
		if err := api.MarkAllNotificationsAsRead(orgID); err != nil {
			return utils.NewError(fmt.Sprintf("failed to mark all as read: %s", err.Error()), nil)
		}
		utils.PrintSuccess("All notifications marked as read")
	} else {
		if err := api.MarkNotificationAsRead(orgID, notifID); err != nil {
			return utils.NewError(fmt.Sprintf("failed to mark as read: %s", err.Error()), nil)
		}
		utils.PrintSuccess("Notification marked as read")
	}
	return nil
}

func handleNotifDelete(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	notifID := c.String("id")
	if notifID == "" {
		return utils.NewError("--id is required", nil)
	}

	if err := api.DeleteNotification(orgID, notifID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete notification: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Notification deleted successfully")
	return nil
}
