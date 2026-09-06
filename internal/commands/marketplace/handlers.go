package marketplace

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	deploypkg "1ctl/internal/deploy"
	"1ctl/internal/utils"
)

func handleMarketplaceList(ctx context.Context, in marketplaceListInput) error {
	apps, err := api.GetMarketplaceApps(in.Limit, in.Offset, in.Sort)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list marketplace apps: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(apps, "No marketplace apps available") {
		return nil
	}

	utils.PrintHeader("Marketplace Apps")
	for _, app := range apps {
		status := marketplaceAvailability(app)
		utils.PrintStatusLine("Name", app.MarketplaceName)
		utils.PrintStatusLine("Category", app.Category)
		if app.Description != "" {
			utils.PrintStatusLine("Description", app.Description)
		}
		utils.PrintStatusLine("Status", status)
		if !app.Deployable && app.DeployabilityCode != "" {
			utils.PrintStatusLine("Availability code", app.DeployabilityCode)
		}
		if app.DeploymentCount > 0 {
			utils.PrintStatusLine("Deployments", fmt.Sprintf("%d", app.DeploymentCount))
		}
		utils.PrintDivider()
	}
	return nil
}

func handleMarketplaceGet(ctx context.Context, nameOrID string) error {
	app, err := api.ResolveMarketplaceApp(nameOrID)
	if err != nil {
		return err
	}
	if utils.TryPrintJSON(app) {
		return nil
	}

	status := marketplaceAvailability(*app)

	utils.PrintHeader("Marketplace App: %s", app.MarketplaceName)
	utils.PrintStatusLine("Name", app.MarketplaceName)
	utils.PrintStatusLine("Category", app.Category)
	if app.Description != "" {
		utils.PrintStatusLine("Description", app.Description)
	}
	if app.ImageURL != "" {
		utils.PrintStatusLine("Image URL", app.ImageURL)
	}
	utils.PrintStatusLine("Status", status)
	if !app.Deployable && app.DeployabilityCode != "" {
		utils.PrintStatusLine("Availability code", app.DeployabilityCode)
	}
	if app.DeploymentCount > 0 {
		utils.PrintStatusLine("Deployments", fmt.Sprintf("%d", app.DeploymentCount))
	}
	utils.PrintStatusLine("Created", utils.FormatTimeAgo(app.CreatedAt))
	utils.PrintStatusLine("Updated", utils.FormatTimeAgo(app.UpdatedAt))

	if len(app.Metadata) > 0 {
		fmt.Println()
		utils.PrintHeader("Details")
		for key, value := range app.Metadata {
			utils.PrintStatusLine(key, fmt.Sprintf("%v", value))
		}
	}

	return nil
}

func handleMarketplaceDeploy(ctx context.Context, in marketplaceDeployInput) error {
	namespace := satuskyctx.GetCurrentNamespace()
	if namespace == "" {
		return utils.NewError("namespace not found. Please run '1ctl auth login' first", nil)
	}

	app, err := api.ResolveMarketplaceApp(in.AppName)
	if err != nil {
		return err
	}
	if !app.Deployable {
		code := app.DeployabilityCode
		if code == "" {
			code = "FEATURE_NOT_AVAILABLE"
		}
		return utils.NewLocalDiagnosticError(
			fmt.Sprintf("deployment is unavailable for %q", app.MarketplaceName),
			code,
			marketplaceDeploymentRemediation(code),
		)
	}

	deployName := in.DeployName
	if deployName == "" {
		deployName = app.MarketplaceName
	}
	machineIDs, err := resolveMarketplaceMachineReferences(in.Hostnames, api.GetMachineByName)
	if err != nil {
		return err
	}

	req := api.MarketplaceDeployRequest{
		DeploymentName: deployName,
		Hostnames:      machineIDs,
		CPURequest:     in.CPU,
		MemoryRequest:  in.Memory,
		StorageSize:    in.StorageSize,
	}

	resp, err := api.DeployMarketplaceApp(namespace, app.MarketplaceID.String(), req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to deploy marketplace app: %s", err.Error()), nil)
	}

	// Marketplace create returns 202 Accepted: the request was queued, not made ready.
	if utils.TryPrintJSON(resp) {
		return nil
	}
	return deploypkg.ReportDeployResult(resp.AppLabel, resp.DeploymentID.String(), resp.Domain,
		deploypkg.PublicURLReadiness{Ready: false, Reason: "deployment accepted; readiness was not verified"}, "", false)
}

func resolveMarketplaceMachineReferences(references []string, lookup func(string) (*api.Machine, error)) ([]string, error) {
	var result []string
	seen := make(map[string]bool)
	for _, reference := range references {
		reference = strings.TrimSpace(reference)
		if reference == "" {
			return nil, fmt.Errorf("machine hostname must not be empty")
		}
		id := reference
		raw, hexErr := hex.DecodeString(reference)
		_, uuidErr := uuid.Parse(reference)
		if (hexErr != nil || len(raw) != 16) && uuidErr != nil {
			machine, err := lookup(reference)
			if err != nil {
				return nil, fmt.Errorf("resolve marketplace machine %q: %w", reference, err)
			}
			if machine == nil || machine.MachineID == "" {
				return nil, fmt.Errorf("machine %q has no stable ID", reference)
			}
			id = machine.MachineID
		}
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result, nil
}

// marketplaceAvailability answers "can I deploy this right now?".
//
// Deployable is the operational answer the platform computes from the pinned
// package release, and it is the same fact handleMarketplaceDeploy gates on.
// ComingSoon is curated roadmap metadata that gates nothing. So ComingSoon may
// only describe an app that is not deployable — telling a user "Coming Soon"
// about an app that deploys on request is the display contradicting the
// command.
func marketplaceAvailability(app api.MarketplaceApp) string {
	if app.Deployable {
		return "Available"
	}
	if app.ComingSoon {
		return "Coming Soon"
	}
	return "Unavailable"
}

func marketplaceDeploymentRemediation(code string) []string {
	if code == "PACKAGE_TRUST_INVALID" {
		return []string{"Ask an authorized operator to re-sign the package before deploying."}
	}
	return []string{"This marketplace app is not currently available for deployment."}
}
