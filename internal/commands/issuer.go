package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func IssuerCommand() *cli.Command {
	return &cli.Command{
		Name:  "issuer",
		Usage: "Manage SSL/TLS certificate issuers",
		Subcommands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Create a new certificate issuer",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "deployment-id",
						Usage:    "Deployment ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "email",
						Usage:    "Email address for Let's Encrypt notifications",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "environment",
						Usage: "Environment (production/staging)",
						Value: "staging",
					},
				},
				Action: handleCreateIssuer,
			},
			{
				Name:   "list",
				Usage:  "List all certificate issuers",
				Action: handleListIssuers,
			},
		},
	}
}

func handleCreateIssuer(c *cli.Context) error {
	issuer := api.Issuer{
		DeploymentID: uuid.MustParse(c.String("deployment-id")),
		UserEmail:    c.String("email"),
		Environment:  c.String("environment"),
		Namespace:    context.GetCurrentNamespace(),
	}

	resp, err := api.CreateIssuer(issuer)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create issuer: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Certificate issuer created successfully (ID: %s)\n", resp.IssuerID)
	return nil
}

func handleListIssuers(c *cli.Context) error {
	issuers, err := api.ListIssuers()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list issuers: %s", err.Error()), nil)
	}

	if len(issuers) == 0 {
		utils.PrintInfo("No certificate issuers found")
		return nil
	}

	utils.PrintHeader("Certificate Issuers")
	for _, issuer := range issuers {
		utils.PrintStatusLine("Issuer", issuer.IssuerName)
		utils.PrintStatusLine("ID", issuer.IssuerID.String())
		utils.PrintStatusLine("Deployment ID", issuer.DeploymentID.String())
		utils.PrintStatusLine("Environment", issuer.Environment)
		utils.PrintStatusLine("Email", issuer.UserEmail)
		utils.PrintStatusLine("Created", api.FormatTimeAgo(issuer.CreatedAt))
		utils.PrintStatusLine("Last Updated", api.FormatTimeAgo(issuer.UpdatedAt))
		utils.PrintDivider()
	}
	return nil
}
