package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
)

func MarketplaceCommand() *cli.Command {
	return &cli.Command{
		Name:    "marketplace",
		Aliases: []string{"market", "apps"},
		Usage:   "Browse and deploy marketplace apps",
		Subcommands: []*cli.Command{
			marketListCommand(),
			marketGetCommand(),
			marketDeployCommand(),
		},
	}
}

func marketListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List marketplace apps",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "limit",
				Usage: "Number of apps to show",
				Value: 20,
			},
			&cli.IntFlag{
				Name:  "offset",
				Usage: "Offset for pagination",
				Value: 0,
			},
			&cli.StringFlag{
				Name:  "sort",
				Usage: "Sort by field (e.g., popularity, name)",
			},
		},
		Action: handleMarketList,
	}
}

func marketGetCommand() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Get marketplace app details",
		ArgsUsage: "<marketplace-id>",
		Action:    handleMarketGet,
	}
}

func marketDeployCommand() *cli.Command {
	return &cli.Command{
		Name:      "deploy",
		Usage:     "Deploy a marketplace app",
		ArgsUsage: "<marketplace-id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Usage:    "Deployment name (required)",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:  "hostname",
				Usage: "Machine hostname(s) to deploy to",
			},
			&cli.StringFlag{
				Name:  "cpu",
				Usage: "CPU cores allocation (e.g., '2')",
			},
			&cli.StringFlag{
				Name:  "memory",
				Usage: "Memory allocation (e.g., '4Gi')",
			},
			&cli.StringFlag{
				Name:  "domain",
				Usage: "Custom domain (default: auto-generated)",
			},
			&cli.StringFlag{
				Name:  "storage-size",
				Usage: "Storage size for PVC (e.g., '10Gi')",
			},
			&cli.StringFlag{
				Name:  "storage-class",
				Usage: "Storage class for PVC",
			},
			&cli.BoolFlag{
				Name:  "multicluster",
				Usage: "Enable multi-cluster deployment",
			},
			&cli.StringFlag{
				Name:  "multicluster-mode",
				Usage: "Multi-cluster mode: 'active-active' or 'active-passive'",
				Value: "active-passive",
			},
		},
		Action: handleMarketDeploy,
	}
}

func handleMarketList(c *cli.Context) error {
	limit := c.Int("limit")
	offset := c.Int("offset")
	sortBy := c.String("sort")

	apps, err := api.GetMarketplaceApps(limit, offset, sortBy)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list marketplace apps: %s", err.Error()), nil)
	}

	if len(apps) == 0 {
		utils.PrintInfo("No marketplace apps available")
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

func handleMarketGet(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("marketplace ID is required", nil)
	}

	marketplaceID := c.Args().First()

	app, err := api.GetMarketplaceApp(marketplaceID)
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
	utils.PrintStatusLine("Created", formatTimeAgo(app.CreatedAt))
	utils.PrintStatusLine("Updated", formatTimeAgo(app.UpdatedAt))

	if len(app.Metadata) > 0 {
		fmt.Println()
		utils.PrintHeader("Metadata")
		for key, value := range app.Metadata {
			utils.PrintStatusLine(key, fmt.Sprintf("%v", value))
		}
	}

	return nil
}

func handleMarketDeploy(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("marketplace ID is required", nil)
	}

	namespace := context.GetCurrentNamespace()
	if namespace == "" {
		return utils.NewError("namespace not found. Please run '1ctl auth login' first", nil)
	}

	marketplaceID := c.Args().First()
	deployName := c.String("name")
	hostnames := c.StringSlice("hostname")
	cpuCores := c.String("cpu")
	memory := c.String("memory")
	domain := c.String("domain")
	storageSize := c.String("storage-size")
	storageClass := c.String("storage-class")
	multicluster := c.Bool("multicluster")
	multiclusterMode := c.String("multicluster-mode")

	req := api.MarketplaceDeployRequest{
		DeploymentName: deployName,
		Hostnames:      hostnames,
		CPUCores:       cpuCores,
		Memory:         memory,
		DomainName:     domain,
		StorageSize:    storageSize,
		StorageClass:   storageClass,
	}

	if multicluster {
		req.MulticlusterConfig = &api.MulticlusterConfig{
			Enabled: true,
			Mode:    multiclusterMode,
		}
	}

	resp, err := api.DeployMarketplaceApp(namespace, marketplaceID, req)
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

	if len(hostnames) > 0 {
		utils.PrintStatusLine("Machines", strings.Join(hostnames, ", "))
	}

	return nil
}
