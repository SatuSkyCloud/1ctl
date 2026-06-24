// Package notifications defines the "1ctl notifications" command tree — flag
// names, input structs, and CLI wiring. Handler logic lives in handlers.go.
package notifications

import (
	"context"

	"1ctl/internal/utils"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------
const (
	flagUnread = "unread"
	flagLimit  = "limit"
	flagID     = "id"
	flagAll    = "all"
	flagYes    = "yes"
)

// --- Input structs ------------------------------------------------------

type notifListInput struct {
	Unread bool
	Limit  int
}

type notifReadInput struct {
	ID  string
	All bool
}

type notifDeleteInput struct {
	ID  string
	Yes bool
}

// --- Command tree -------------------------------------------------------

// Command returns the root notifications command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "notifications",
		
		Usage:   "Manage notifications",
		Commands: []*cli.Command{
			notifListCommand(),
			notifCountCommand(),
			notifReadCommand(),
			notifDeleteCommand(),
		},
	}
}

func notifListCommand() *cli.Command {
	var in notifListInput
	return &cli.Command{
		Name:  "list",
		Usage: "List notifications",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        flagUnread,
				Usage:       "Show only unread notifications",
				Destination: &in.Unread,
			},
			&cli.IntFlag{
				Name:        flagLimit,
				Usage:       "Number of notifications to show",
				Value:       20,
				Destination: &in.Limit,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleNotifList(ctx, in)
		},
	}
}

func notifCountCommand() *cli.Command {
	return &cli.Command{
		Name:  "count",
		Usage: "Show unread notification count",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleNotifCount(ctx)
		},
	}
}

func notifReadCommand() *cli.Command {
	var in notifReadInput
	return &cli.Command{
		Name:  "read",
		Usage: "Mark notification(s) as read",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagID,
				Usage:       "Notification ID to mark as read",
				Destination: &in.ID,
			},
			&cli.BoolFlag{
				Name:        flagAll,
				Usage:       "Mark all notifications as read",
				Destination: &in.All,
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			if !in.All && in.ID == "" {
				return ctx, utils.NewError("either --id or --all is required", nil)
			}
			return ctx, nil
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleNotifRead(ctx, in)
		},
	}
}

func notifDeleteCommand() *cli.Command {
	var in notifDeleteInput
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a notification",
		ArgsUsage: "<notification-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        flagYes,
				Aliases:     []string{"y"},
				Usage:       "Skip confirmation prompt",
				Destination: &in.Yes,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.ID = cmd.Args().First()
			return handleNotifDelete(ctx, in)
		},
	}
}


