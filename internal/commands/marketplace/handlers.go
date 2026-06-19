package marketplace

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
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
		utils.PrintStatusLine("ID", app.MarketplaceID.String())
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

func handleMarketplaceGet(ctx context.Context, in marketplaceGetInput) error {
	app, err := api.GetMarketplaceApp(in.MarketplaceID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get marketplace app: %s", err.Error()), nil)
	}

	status := "Available"
	if app.ComingSoon {
		status = "Coming Soon"
	}

	utils.PrintHeader("Marketplace App: %s", app.MarketplaceName)
	utils.PrintStatusLine("ID", app.MarketplaceID.String())
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
		utils.PrintHeader("Metadata")
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

	req := api.MarketplaceDeployRequest{
		DeploymentName: in.Name,
		Hostnames:      in.Hostnames,
		CPUCores:       in.CPU,
		Memory:         in.Memory,
		DomainName:     in.Domain,
		StorageSize:    in.StorageSize,
		StorageClass:   in.StorageClass,
	}

	if in.Multicluster {
		req.MulticlusterConfig = &api.MulticlusterConfig{
			Enabled: true,
			Mode:    in.MulticlusterMode,
		}
	}

	resp, err := api.DeployMarketplaceApp(namespace, in.MarketplaceID, req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to deploy marketplace app: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Marketplace app deployed successfully!")
	utils.PrintStatusLine("Deployment ID", resp.DeploymentID.String())
	utils.PrintStatusLine("App Label", resp.AppLabel)
	if resp.Domain != "" {
		utils.PrintStatusLine("Domain", resp.Domain)
	}
	utils.PrintStatusLine("Status", resp.Status)

	if len(in.Hostnames) > 0 {
		utils.PrintStatusLine("Machines", utils.JoinOptions(in.Hostnames))
	}

	return nil
}
