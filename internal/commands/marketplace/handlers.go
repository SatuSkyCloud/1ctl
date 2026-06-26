package marketplace

import (
	"context"
	"fmt"

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
		status := "Available"
		if app.ComingSoon {
			status = "Coming Soon"
		}
		utils.PrintStatusLine("Name", app.MarketplaceName)
		utils.PrintStatusLine("Category", app.Category)
		if app.Description != "" {
			utils.PrintStatusLine("Description", app.Description)
		}
		utils.PrintStatusLine("Status", status)
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

	status := "Available"
	if app.ComingSoon {
		status = "Coming Soon"
	}

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
	if app.ComingSoon {
		return utils.NewError(fmt.Sprintf("%q is coming soon and cannot be deployed yet", app.MarketplaceName), nil)
	}

	deployName := in.DeployName
	if deployName == "" {
		deployName = app.MarketplaceName
	}

	req := api.MarketplaceDeployRequest{
		DeploymentName: deployName,
		Hostnames:      in.Hostnames,
		CPUCores:       in.CPU,
		Memory:         in.Memory,
		DomainName:     in.Domain,
		StorageSize:    in.StorageSize,
	}

	resp, err := api.DeployMarketplaceApp(namespace, app.MarketplaceID.String(), req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to deploy marketplace app: %s", err.Error()), nil)
	}

	ingressID := deploypkg.ResolveIngressID(resp.DeploymentID.String())
	publicURL := deploypkg.WaitForPublicURL(ingressID, resp.Domain)
	return deploypkg.ReportDeployResult(resp.AppLabel, resp.DeploymentID.String(), resp.Domain, publicURL, "", true)
}
