package admin

import (
	"context"
	"fmt"
	"strings"

	"1ctl/internal/api"
	"1ctl/internal/utils"
)

var adoptDeployment = api.AdoptDeployment
var adoptDeploymentRouting = api.AdoptDeploymentRouting

func handleDeploymentAdopt(_ context.Context, in deploymentAdoptInput) error {
	prompt := fmt.Sprintf("Adopt Deployment %s and transfer its field ownership to the durable reconciler?", in.DeploymentID)
	if !utils.Confirm(prompt, in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := adoptDeployment(in.DeploymentID, strings.TrimSpace(in.RequestID), api.DeploymentAdoptionRequest{
		Reason:                  strings.TrimSpace(in.Reason),
		ExpectedUID:             strings.TrimSpace(in.ExpectedUID),
		ExpectedResourceVersion: strings.TrimSpace(in.ExpectedResourceVersion),
		ExpectedGeneration:      in.ExpectedGeneration,
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to adopt deployment: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(result) {
		return nil
	}

	utils.PrintSuccess("Deployment adopted for durable reconciliation")
	utils.PrintStatusLine("App", result.AppLabel)
	utils.PrintStatusLine("Deployment ID", result.DeploymentID)
	utils.PrintStatusLine("Namespace", result.Namespace)
	utils.PrintStatusLine("State", result.ReconciliationState)
	utils.PrintStatusLine("Managed", fmt.Sprintf("%t", result.ReconciliationManaged))
	utils.PrintStatusLine("Field manager", result.FieldManager)
	utils.PrintStatusLine("Already managed", fmt.Sprintf("%t", result.AlreadyManaged))
	utils.PrintStatusLine("Request ID", result.RequestID)
	return nil
}

func handleDeploymentRoutingAdopt(_ context.Context, in deploymentAdoptInput) error {
	prompt := fmt.Sprintf("Adopt the HTTPRoute for Deployment %s and transfer its field ownership to the durable reconciler?", in.DeploymentID)
	if !utils.Confirm(prompt, in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}

	result, err := adoptDeploymentRouting(in.DeploymentID, strings.TrimSpace(in.RequestID), api.DeploymentAdoptionRequest{
		Reason:                  strings.TrimSpace(in.Reason),
		ExpectedUID:             strings.TrimSpace(in.ExpectedUID),
		ExpectedResourceVersion: strings.TrimSpace(in.ExpectedResourceVersion),
		ExpectedGeneration:      in.ExpectedGeneration,
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to adopt deployment routing: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(result) {
		return nil
	}

	utils.PrintSuccess("HTTPRoute adopted for durable reconciliation")
	utils.PrintStatusLine("Deployment ID", result.DeploymentID)
	utils.PrintStatusLine("Route", result.Name)
	utils.PrintStatusLine("Namespace", result.Namespace)
	utils.PrintStatusLine("Field manager", result.FieldManager)
	utils.PrintStatusLine("Force", fmt.Sprintf("%t", result.Force))
	utils.PrintStatusLine("Already managed", fmt.Sprintf("%t", result.AlreadyManaged))
	utils.PrintStatusLine("Request ID", result.RequestID)
	return nil
}
