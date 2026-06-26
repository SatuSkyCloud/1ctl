// Package org defines the "1ctl org" command tree.
package org

import (
	"context"
	"strings"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagOrgID       = "org-id"
	flagOrgName     = "org-name"
	flagDescription = "description"
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
		Name:   "list",
		Usage:  "List all organizations",
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleOrgList(ctx) },
	}
}

func orgCurrentCommand() *cli.Command {
	return &cli.Command{
		Name:   "current",
		Usage:  "Show current organization",
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleOrgCurrent(ctx) },
	}
}

func orgSwitchCommand() *cli.Command {
	var in orgSwitchInput
	return &cli.Command{
		Name:      "switch",
		Usage:     "Switch to a different organization",
		ArgsUsage: "<org-name-or-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagOrgID,
				Usage:       "Organization ID to switch to (explicit, overrides positional)",
				Destination: &in.OrgID,
			},
			&cli.StringFlag{
				Name:        flagOrgName,
				Usage:       "Organization name to switch to (explicit, overrides positional)",
				Destination: &in.OrgName,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if in.OrgID == "" && in.OrgName == "" {
				if cmd.Args().Len() < 1 {
					return cli.ShowSubcommandHelp(cmd)
				}
				arg := cmd.Args().First()
				if len(arg) == 36 && strings.Count(arg, "-") == 4 {
					in.OrgID = arg
				} else {
					in.OrgName = arg
				}
			}
			return handleOrgSwitch(ctx, in)
		},
	}
}

func orgCreateCommand() *cli.Command {
	var in orgCreateInput
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new organization",
		ArgsUsage: "<name>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagDescription,
				Usage:       "Organization description",
				Destination: &in.Description,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 {
				return cli.ShowSubcommandHelp(cmd)
			}
			in.Name = cmd.Args().First()
			return handleOrgCreate(ctx, in)
		},
	}
}

func orgDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete an organization",
		ArgsUsage: "<org-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id := ""
			if cmd.Args().First() != "" {
				id = cmd.Args().First()
			}
			return handleOrgDelete(ctx, id)
		},
	}
}

func orgTeamCommand() *cli.Command {
	return &cli.Command{
		Name:  "team",
		Usage: "Manage organization team",
		Commands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List team members",
				Action: func(ctx context.Context, cmd *cli.Command) error { return handleOrgTeamList(ctx) },
			},
			orgTeamAddSubCommand(),
			orgTeamRoleSubCommand(),
			orgTeamRemoveSubCommand(),
		},
	}
}

func orgTeamAddSubCommand() *cli.Command {
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
		Action: func(ctx context.Context, cmd *cli.Command) error { return handleOrgTeamAdd(ctx, in) },
	}
}

func orgTeamRoleSubCommand() *cli.Command {
	var in orgTeamRoleInput
	return &cli.Command{
		Name:      "role",
		Usage:     "Update team member role",
		ArgsUsage: "<org-user-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagRole,
				Usage:       "New role (admin, member)",
				Required:    true,
				Destination: &in.Role,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id := ""
			if cmd.Args().First() != "" {
				id = cmd.Args().First()
			}
			in.OrgUserID = id
			return handleOrgTeamRole(ctx, in)
		},
	}
}

func orgTeamRemoveSubCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"remove", "rm"},
		Usage:     "Remove a team member",
		ArgsUsage: "<org-user-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id := ""
			if cmd.Args().First() != "" {
				id = cmd.Args().First()
			}
			return handleOrgTeamRemove(ctx, id)
		},
	}
}
