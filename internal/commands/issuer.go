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

func IssuerCommand() *cli.Command {
	return &cli.Command{
		Name:  "issuer",
		Usage: "Manage cert-manager issuers (internal — TLS is automatic when adding a custom domain)",
		// Hidden from --help and shell completion: cert-manager Issuer is a
		// backend implementation detail. Custom-domain TLS is provisioned
		// automatically when `1ctl domains add` resolves a non-*.satusky.com
		// host. The command still works for scripts that depend on it.
		Hidden: true,
		Commands: []*cli.Command{
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
			{
				Name:  "delete",
				Usage: "Delete a certificate issuer",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "issuer-id",
						Aliases:  []string{"i"},
						Usage:    "Issuer ID to delete",
						Required: true,
					},
				},
				Action: handleDeleteIssuer,
			},
		},
	}
}

func handleCreateIssuer(ctx context.Context, cmd *cli.Command) error {
	deploymentIDStr := cmd.String("deployment-id")
	if deploymentIDStr == "" {
		return utils.NewError("deployment-id is required", nil)
	}

	deploymentID, err := uuid.Parse(deploymentIDStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	issuer := api.Issuer{
		DeploymentID: deploymentID,
		UserEmail:    cmd.String("email"),
		Environment:  cmd.String("environment"),
		Namespace:    satuskyctx.GetCurrentNamespace(),
	}

	resp, err := api.CreateIssuer(issuer)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create issuer: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Certificate issuer created successfully (ID: %s)\n", resp.IssuerID)
	return nil
}

func handleDeleteIssuer(ctx context.Context, cmd *cli.Command) error {
	issuerID := cmd.String("issuer-id")
	if err := api.DeleteIssuer(issuerID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete issuer: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Certificate issuer %s deleted successfully", issuerID)
	return nil
}

func handleListIssuers(ctx context.Context, cmd *cli.Command) error {
	issuers, err := api.ListIssuers()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list issuers: %s", err.Error()), nil)
	}

	if len(issuers) == 0 {
		utils.PrintInfo("No certificate issuers found")
		return nil
	}

	if utils.TryPrintJSON(issuers) {
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
