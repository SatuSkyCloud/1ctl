package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v2"
)

func OrgCommand() *cli.Command {
	return &cli.Command{
		Name:    "org",
		Aliases: []string{"organization"},
		Usage:   "Manage organizations",
		Subcommands: []*cli.Command{
			{
				Name:   "current",
				Usage:  "Show current organization",
				Action: handleOrgCurrent,
			},
			{
				Name:   "switch",
				Usage:  "Switch to a different organization",
				Action: handleOrgSwitch,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "org-id",
						Usage:    "Organization ID to switch to",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "org-name",
						Usage:    "Organization name to switch to",
						Required: false,
					},
				},
			},
		},
	}
}

func handleOrgCurrent(c *cli.Context) error {
	orgName := context.GetCurrentOrgName()
	orgID := context.GetCurrentOrgID()
	namespace := context.GetCurrentNamespace()

	if orgName == "" {
		return utils.NewError("no organization set. Please run '1ctl auth login' first", nil)
	}

	utils.PrintSuccess("Current Organization\n")
	utils.PrintStatusLine("Organization", orgName)
	utils.PrintStatusLine("Organization ID", orgID)
	utils.PrintStatusLine("Namespace", namespace)
	return nil
}

func handleOrgSwitch(c *cli.Context) error {
	orgID := c.String("org-id")
	orgName := c.String("org-name")

	if orgID == "" && orgName == "" {
		return utils.NewError("either --org-id or --org-name is required", nil)
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
		// Switch by org name - need to get org by name
		// First get user profile to check if user has access to this org
		// For now, we'll return an error and suggest using org-id
		return utils.NewError("switching by org-name is not yet supported. Please use --org-id instead", nil)
	}

	// Update context with new organization
	if err := context.SetCurrentOrganization(org.OrganizationID.String(), org.OrganizationName, org.OrganizationName); err != nil {
		return utils.NewError(fmt.Sprintf("failed to switch organization: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Switched to organization: %s\n", org.OrganizationName)
	utils.PrintStatusLine("Organization", org.OrganizationName)
	utils.PrintStatusLine("Organization ID", org.OrganizationID.String())
	return nil
}
