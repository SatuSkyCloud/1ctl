package issuer

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"

	"github.com/google/uuid"
)

// --- Handlers -----------------------------------------------------------

func handleCreateIssuer(ctx context.Context, in issuerCreateInput) error {
	if in.DeploymentID == "" {
		return utils.NewError("deployment-id is required", nil)
	}

	deploymentID, err := uuid.Parse(in.DeploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	issuer := api.Issuer{
		DeploymentID: deploymentID,
		UserEmail:    in.Email,
		Environment:  in.Environment,
		Namespace:    satuskyctx.GetCurrentNamespace(),
	}

	resp, err := api.CreateIssuer(issuer)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create issuer: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Certificate issuer created successfully (ID: %s)\n", resp.IssuerID)
	return nil
}

func handleDeleteIssuer(ctx context.Context, in issuerDeleteInput) error {
	if err := api.DeleteIssuer(in.IssuerID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete issuer: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Certificate issuer %s deleted successfully", in.IssuerID)
	return nil
}

func handleListIssuers(ctx context.Context) error {
	issuers, err := api.ListIssuers()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list issuers: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(issuers, "No certificate issuers found") {
		return nil
	}

	utils.PrintHeader("Certificate Issuers")
	for _, issuer := range issuers {
		utils.PrintStatusLine("Issuer", issuer.IssuerName)
		utils.PrintStatusLine("ID", issuer.IssuerID.String())
		utils.PrintStatusLine("Deployment ID", issuer.DeploymentID.String())
		utils.PrintStatusLine("Environment", issuer.Environment)
		utils.PrintStatusLine("Email", issuer.UserEmail)
		utils.PrintStatusLine("Created", utils.FormatTimeAgo(issuer.CreatedAt))
		utils.PrintStatusLine("Last Updated", utils.FormatTimeAgo(issuer.UpdatedAt))
		utils.PrintDivider()
	}
	return nil
}
