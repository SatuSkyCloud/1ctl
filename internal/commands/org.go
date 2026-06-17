package commands

import (
	"context"
	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func OrgCommand() *cli.Command {
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
		Action: handleOrgList,
	}
}

func orgCurrentCommand() *cli.Command {
	return &cli.Command{
		Name:   "current",
		Usage:  "Show current organization",
		Action: handleOrgCurrent,
	}
}

func orgSwitchCommand() *cli.Command {
	return &cli.Command{
		Name:  "switch",
		Usage: "Switch to a different organization",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "org-id",
				Usage: "Organization ID to switch to",
			},
			&cli.StringFlag{
				Name:  "org-name",
				Usage: "Organization name to switch to",
			},
		},
		Action: handleOrgSwitch,
	}
}

func orgCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new organization",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Usage:    "Organization name",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "description",
				Usage: "Organization description",
			},
		},
		Action: handleOrgCreate,
	}
}

func orgDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete an organization",
		ArgsUsage: "<org-id>",
		Action:    handleOrgDelete,
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
				Action: handleOrgTeamList,
			},
			{
				Name:  "add",
				Usage: "Add a team member",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "email",
						Usage:    "User email to add",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "role",
						Usage: "Role (admin, member)",
						Value: "member",
					},
				},
				Action: handleOrgTeamAdd,
			},
			{
				Name:      "role",
				Usage:     "Update team member role",
				ArgsUsage: "<org-user-id>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "role",
						Usage:    "New role (admin, member)",
						Required: true,
					},
				},
				Action: handleOrgTeamRole,
			},
			{
				Name:      "delete",
				Aliases:   []string{"remove", "rm"},
				Usage:     "Remove a team member",
				ArgsUsage: "<org-user-id>",
				Action:    handleOrgTeamRemove,
			},
		},
	}
}

func handleOrgList(ctx context.Context, cmd *cli.Command) error {
	orgs, err := api.GetUserOrganizations()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list organizations: %s", err.Error()), nil)
	}

	if len(orgs) == 0 {
		utils.PrintInfo("No organizations found")
		return nil
	}

	currentOrgID := satuskyctx.GetCurrentOrgID()

	utils.PrintHeader("Organizations")
	for _, org := range orgs {
		current := ""
		if org.OrganizationID.String() == currentOrgID {
			current = " (current)"
		}
		utils.PrintStatusLine("ID", org.OrganizationID.String()+current)
		utils.PrintStatusLine("Name", org.OrganizationName)
		if org.Description != "" {
			utils.PrintStatusLine("Description", org.Description)
		}
		utils.PrintStatusLine("Created", formatTimeAgo(org.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleOrgCurrent(ctx context.Context, cmd *cli.Command) error {
	orgName := satuskyctx.GetCurrentOrgName()
	orgID := satuskyctx.GetCurrentOrgID()

	if orgName == "" {
		return utils.NewError("no organization set. Please run '1ctl auth login' first", nil)
	}

	utils.PrintHeader("Current Organization")
	utils.PrintStatusLine("Organization", orgName)
	utils.PrintStatusLine("Organization ID", orgID)
	return nil
}

func handleOrgSwitch(ctx context.Context, cmd *cli.Command) error {
	orgID := cmd.String("org-id")
	orgName := cmd.String("org-name")

	// Accept positional arg as org-id or org-name
	if orgID == "" && orgName == "" {
		if arg := cmd.Args().First(); arg != "" {
			// Try as UUID first, fall back to name
			if _, err := uuid.Parse(arg); err == nil {
				orgID = arg
			} else {
				orgName = arg
			}
		}
	}

	if orgID == "" && orgName == "" {
		return utils.NewError("provide --org-id, --org-name, or a positional org name/id", nil)
	}

	// Get the organization details
	var org *api.Organization
	var err error

	if orgID != "" {
		// Switch by org ID
		org, err = api.GetOrganizationByID(api.ToUUID(orgID))
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to get organization: %s", err.Error()), nil)
		}
	} else {
		// Switch by org name - search in user's organizations
		orgs, err := api.GetUserOrganizations()
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to list organizations: %s", err.Error()), nil)
		}
		for _, o := range orgs {
			if o.OrganizationName == orgName {
				org = &o
				break
			}
		}
		if org == nil {
			return utils.NewError(fmt.Sprintf("organization '%s' not found", orgName), nil)
		}
	}

	// The organizations table enforces NOT NULL on namespace, so a successful
	// org lookup always returns a non-empty namespace. An empty value means the
	// API response is malformed; falling back to OrganizationName would produce
	// an invalid k8s namespace (org names allow spaces/uppercase/etc).
	if org.Namespace == "" {
		return utils.NewError(fmt.Sprintf("organization '%s' has no namespace assigned — contact support", org.OrganizationName), nil)
	}
	if err := satuskyctx.SetCurrentOrganization(org.OrganizationID.String(), org.OrganizationName, org.Namespace); err != nil {
		return utils.NewError(fmt.Sprintf("failed to switch organization: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Switched to organization: %s", org.OrganizationName)
	utils.PrintStatusLine("Organization", org.OrganizationName)
	utils.PrintStatusLine("Organization ID", org.OrganizationID.String())
	return nil
}

func handleOrgCreate(ctx context.Context, cmd *cli.Command) error {
	name := cmd.String("name")
	description := cmd.String("description")

	req := api.CreateOrganizationRequest{
		Name:        name,
		Description: description,
	}

	org, err := api.CreateOrganization(req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create organization: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Organization created successfully")
	utils.PrintStatusLine("ID", org.OrganizationID.String())
	utils.PrintStatusLine("Name", org.OrganizationName)
	if org.Description != "" {
		utils.PrintStatusLine("Description", org.Description)
	}
	return nil
}

func handleOrgDelete(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("organization ID is required", nil)
	}

	orgID := cmd.Args().First()

	if err := api.DeleteOrganization(orgID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete organization: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Organization deleted successfully")
	return nil
}

func handleOrgTeamList(ctx context.Context, cmd *cli.Command) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	members, err := api.GetOrganizationTeam(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list team members: %s", err.Error()), nil)
	}

	if len(members) == 0 {
		utils.PrintInfo("No team members found")
		return nil
	}

	utils.PrintHeader("Team Members")
	for _, member := range members {
		utils.PrintStatusLine("ID", member.OrganizationUserID.String())
		name := ""
		if member.Name != nil {
			name = *member.Name
		}
		utils.PrintStatusLine("Name", name)
		utils.PrintStatusLine("Email", member.Email)
		utils.PrintStatusLine("Role", member.Role)
		utils.PrintStatusLine("Joined", formatTimeAgo(member.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleOrgTeamAdd(ctx context.Context, cmd *cli.Command) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	email := cmd.String("email")
	role := cmd.String("role")

	req := api.AddTeamMemberRequest{
		Email: email,
		Role:  role,
	}

	member, err := api.AddTeamMember(orgID, req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to add team member: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Team member added successfully")
	utils.PrintStatusLine("ID", member.OrganizationUserID.String())
	utils.PrintStatusLine("Email", member.Email)
	utils.PrintStatusLine("Role", member.Role)
	return nil
}

func handleOrgTeamRole(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("org-user ID is required", nil)
	}

	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	orgUserID := cmd.Args().First()
	role := cmd.String("role")

	if err := api.UpdateTeamMemberRole(orgID, orgUserID, role); err != nil {
		return utils.NewError(fmt.Sprintf("failed to update team member role: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Team member role updated to '%s'", role)
	return nil
}

func handleOrgTeamRemove(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("org-user ID is required", nil)
	}

	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	orgUserID := cmd.Args().First()

	if err := api.RemoveTeamMember(orgID, orgUserID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to remove team member: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Team member removed successfully")
	return nil
}
