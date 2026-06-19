package org

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"

	"github.com/google/uuid"
)

func handleOrgList(ctx context.Context) error {
	orgs, err := api.GetUserOrganizations()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list organizations: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(orgs, "No organizations found") {
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
		utils.PrintStatusLine("Created", utils.FormatTimeAgo(org.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleOrgCurrent(ctx context.Context) error {
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

func handleOrgSwitch(ctx context.Context, in orgSwitchInput) error {
	orgID := in.OrgID
	orgName := in.OrgName

	// Try to parse as UUID — if it's a UUID, treat as org ID; otherwise treat as name
	if orgName == "" && orgID != "" {
		if _, err := uuid.Parse(orgID); err != nil {
			orgName = orgID
			orgID = ""
		}
	}

	if orgID == "" && orgName == "" {
		return utils.NewError("provide --org-id, --org-name, or a positional org name/id", nil)
	}

	var org *api.Organization
	var err error

	if orgID != "" {
		org, err = api.GetOrganizationByID(api.ToUUID(orgID))
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to get organization: %s", err.Error()), nil)
		}
	} else {
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

func handleOrgCreate(ctx context.Context, in orgCreateInput) error {
	req := api.CreateOrganizationRequest{
		Name:        in.Name,
		Description: in.Description,
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

func handleOrgDelete(ctx context.Context, orgID string) error {
	if err := api.DeleteOrganization(orgID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete organization: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Organization deleted successfully")
	return nil
}

func handleOrgTeamList(ctx context.Context) error {
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
		utils.PrintStatusLine("Joined", utils.FormatTimeAgo(member.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleOrgTeamAdd(ctx context.Context, in orgTeamAddInput) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	req := api.AddTeamMemberRequest{
		Email: in.Email,
		Role:  in.Role,
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

func handleOrgTeamRole(ctx context.Context, in orgTeamRoleInput) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	if err := api.UpdateTeamMemberRole(orgID, in.OrgUserID, in.Role); err != nil {
		return utils.NewError(fmt.Sprintf("failed to update team member role: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Team member role updated to '%s'", in.Role)
	return nil
}

func handleOrgTeamRemove(ctx context.Context, orgUserID string) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	if err := api.RemoveTeamMember(orgID, orgUserID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to remove team member: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Team member removed successfully")
	return nil
}
