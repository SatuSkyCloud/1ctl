package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func IngressCommand() *cli.Command {
	ingressFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     "deployment-id",
			Usage:    "Deployment ID",
			Required: false, // Remove required at command level, validate in handler
		},
		&cli.StringFlag{
			Name:     "service-id",
			Usage:    "Service ID",
			Required: false, // Remove required at command level, validate in handler
		},
		&cli.StringFlag{
			Name:     "app-label",
			Usage:    "Application label",
			Required: false, // Remove required at command level, validate in handler
		},
		&cli.StringFlag{
			Name:     "namespace",
			Usage:    "Organization name",
			Required: false, // Remove required at command level, validate in handler
		},
		&cli.StringFlag{
			Name:     "domain",
			Usage:    "Domain name",
			Required: false, // Remove required at command level, validate in handler
		},
		&cli.IntFlag{
			Name:  "port",
			Usage: "Port number",
			Value: 8080,
		},
		&cli.BoolFlag{
			Name:  "custom-dns",
			Usage: "Use custom DNS",
		},
	}

	return &cli.Command{
		Name:  "ingress",
		Usage: "Create or update an ingress for a deployment",
		Flags: ingressFlags,
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List all ingresses",
				Action: handleListIngresses,
			},
			{
				Name:  "delete",
				Usage: "Delete an ingress",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "ingress-id",
						Usage:    "Ingress ID to delete",
						Required: true,
					},
					&cli.BoolFlag{
						Name:    "yes",
						Aliases: []string{"y"},
						Usage:   "Skip confirmation prompt",
					},
				},
				Action: handleDeleteIngress,
			},
		},
		Action: func(c *cli.Context) error {
			// If no subcommand is provided and no flags, show help
			if c.NArg() == 0 && !c.IsSet("deployment-id") && !c.IsSet("domain") {
				return cli.ShowCommandHelp(c, "ingress")
			}

			// If subcommand is provided, let it handle
			if c.NArg() > 0 {
				return cli.ShowSubcommandHelp(c)
			}

			// Otherwise, handle ingress upsert
			return handleUpsertIngress(c)
		},
	}
}

func handleUpsertIngress(c *cli.Context) error {
	// Check required flags for ingress upsert
	if c.String("deployment-id") == "" {
		return utils.NewError("--deployment-id flag is required for ingress", nil)
	}
	if c.String("domain") == "" {
		return utils.NewError("--domain flag is required for ingress", nil)
	}
	if c.String("app-label") == "" {
		return utils.NewError("--app-label flag is required for ingress", nil)
	}
	if c.String("namespace") == "" {
		return utils.NewError("--namespace flag is required for ingress", nil)
	}

	deploymentIDStr := c.String("deployment-id")
	deploymentID, err := uuid.Parse(deploymentIDStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	serviceIDStr := c.String("service-id")
	if serviceIDStr == "" {
		return utils.NewError("--service-id flag is required for ingress", nil)
	}
	serviceID, err := uuid.Parse(serviceIDStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid service-id: %s", err.Error()), nil)
	}

	port, err := api.SafeInt32(c.Int("port"))
	if err != nil {
		return utils.NewError("Invalid port number", err)
	}

	dnsConfig := api.DnsConfigDefault
	if c.Bool("custom-dns") {
		dnsConfig = api.DnsConfigCustom
	}

	ingress := api.Ingress{
		DeploymentID: deploymentID,
		ServiceID:    serviceID,
		AppLabel:     c.String("app-label"),
		Namespace:    c.String("namespace"),
		DomainName:   c.String("domain"),
		Port:         port,
		DnsConfig:    dnsConfig,
	}

	ingressResp, err := api.UpsertIngress(ingress)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to upsert ingress: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Ingress for domain %s upserted successfully\n", ingressResp.DomainName)
	return nil
}

func handleListIngresses(c *cli.Context) error {
	ingresses, err := api.ListIngresses()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list ingresses: %s", err.Error()), nil)
	}

	if len(ingresses) == 0 {
		utils.PrintInfo("No ingresses found")
		return nil
	}

	headers := []string{"DOMAIN", "INGRESS ID", "DEPLOYMENT ID", "DNS CONFIG", "CREATED"}
	rows := make([][]string, 0, len(ingresses))
	for _, ing := range ingresses {
		rows = append(rows, []string{
			ing.DomainName,
			ing.IngressID.String(),
			ing.DeploymentID.String(),
			string(ing.DnsConfig),
			api.FormatTimeAgo(ing.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleDeleteIngress(c *cli.Context) error {
	ingressID := c.String("ingress-id")
	if !utils.Confirm(fmt.Sprintf("Delete ingress %s? This cannot be undone.", ingressID), c.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteIngress(ingressID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete ingress: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Ingress %s deleted successfully\n", ingressID)
	return nil
}
