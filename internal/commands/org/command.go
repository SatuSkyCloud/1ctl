// Package org defines the "1ctl org" command tree — flag names,
// input structs, and CLI wiring. Handler logic lives in handlers.go.
package org

import (
	"context"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagName        = "name"
	flagDescription = "description"
	flagOrgID       = "org-id"
	flagOrgName     = "org-name"
	flagEmail       = "email"
	flagRole        = "role"
)

// --- Input structs ------------------------------------------------------

type orgSwitchInput struct {
	OrgID   string
	OrgName string
}

type orgCreateInput struct {
	Name        string
	Description string
}

type orgTeamAddInput struct {
	Email string
	Role  string
}

type orgTeamRoleInput struct {
	OrgUserID string
	Role      string
}

// --- Command tree -------------------------------------------------------

// Command returns the root org command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "org",
		Aliases: []string{"organization"},
		Usage:   "Manage organizations",
		Commands: []*cli.Command{
			orgListCommand(),
			orgCurrentCommand(),
			orgSwitchCommand(),
			orgCreateCommand(),
			orgDeleteCommand(),
			orgTeamCommand(),
		},
	}
}

func orgListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all organizations",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleOrgList(ctx)
		},
	}
}

func orgCurrentCommand() *cli.Command {
	return &cli.Command{
		Name:  "current",
		Usage: "Show current organization",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleOrgCurrent(ctx)
		},
	}
}

func orgSwitchCommand() *cli.Command {
	var in orgSwitchInput
	return &cli.Command{
		Name:  "switch",
		Usage: "Switch to a different organization",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagOrgID,
				Usage:       "Organization ID to switch to",
				Destination: &in.OrgID,
			},
			&cli.StringFlag{
				Name:        flagOrgName,
				Usage:       "Organization name to switch to",
				Destination: &in.OrgName,
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// Accept --org-id, --org-name, or a positional org name/id
			if in.OrgID == "" && in.OrgName == "" {
				if arg := cmd.Args().First(); arg != "" {
					in.OrgID = arg
				}
			}
			if in.OrgID == "" && in.OrgName == "" {
				return ctx, cli.Exit("provide --org-id, --org-name, or a positional org name/id", 1)
			}
			return ctx, nil
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleOrgSwitch(ctx, in)
		},
	}
}

func orgCreateCommand() *cli.Command {
	var in orgCreateInput
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new organization",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagName,
				Usage:       "Organization name",
				Required:    true,
				Destination: &in.Name,
			},
			&cli.StringFlag{
				Name:        flagDescription,
				Usage:       "Organization description",
				Destination: &in.Description,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleOrgCreate(ctx, in)
		},
	}
}

func orgDeleteCommand() *cli.Command {
	var in struct{ OrgID string }
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete an organization",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagOrgID,
				Usage:       "Organization ID to delete",
				Required:    true,
				Destination: &in.OrgID,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleOrgDelete(ctx, in.OrgID)
		},
	}
}

func orgTeamCommand() *cli.Command {
	return &cli.Command{
		Name:  "team",
		Usage: "Manage organization team",
		Commands: []*cli.Command{
			orgTeamListCommand(),
			orgTeamAddCommand(),
			orgTeamRoleCommand(),
			orgTeamDeleteCommand(),
		},
	}
}

func orgTeamListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List team members",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleOrgTeamList(ctx)
		},
	}
}

func orgTeamAddCommand() *cli.Command {
	var in orgTeamAddInput
	return &cli.Command{
		Name:  "add",
		Usage: "Add a team member",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagEmail,
				Usage:       "User email to add",
				Required:    true,
				Destination: &in.Email,
			},
			&cli.StringFlag{
				Name:        flagRole,
				Usage:       "Role (admin, member)",
				Value:       "member",
				Destination: &in.Role,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleOrgTeamAdd(ctx, in)
		},
	}
}

func orgTeamRoleCommand() *cli.Command {
	var in orgTeamRoleInput
	return &cli.Command{
		Name:  "role",
		Usage: "Update team member role",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagRole,
				Usage:       "New role (admin, member)",
				Required:    true,
				Destination: &in.Role,
			},
			&cli.StringFlag{
				Name:        flagOrgID,
				Usage:       "Organization user ID",
				Required:    true,
				Destination: &in.OrgUserID,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleOrgTeamRole(ctx, in)
		},
	}
}

func orgTeamDeleteCommand() *cli.Command {
	var in struct{ OrgUserID string }
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"remove", "rm"},
		Usage:   "Remove a team member",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagOrgID,
				Usage:       "Organization user ID to remove",
				Required:    true,
				Destination: &in.OrgUserID,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleOrgTeamRemove(ctx, in.OrgUserID)
		},
	}
}
