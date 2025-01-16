package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func IngressCommand() *cli.Command {
	return &cli.Command{
		Name:  "ingress",
		Usage: "Manage ingresses for a deployment",
		Subcommands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Create a new ingress",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "deployment-id",
						Usage:    "Deployment ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "domain",
						Usage:    "Domain name",
						Required: true,
					},
					&cli.BoolFlag{
						Name:  "custom-dns",
						Usage: "Use custom DNS",
					},
				},
				Action: handleCreateIngress,
			},
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
				},
				Action: handleDeleteIngress,
			},
		},
	}
}

func handleCreateIngress(c *cli.Context) error {
	ingress := api.Ingress{
		DeploymentID: uuid.MustParse(c.String("deployment-id")),
		DomainName:   c.String("domain"),
		DnsConfig:    api.DnsConfigType(api.DnsConfigCustom),
	}

	ingressResp, err := api.CreateIngress(ingress)
	if err != nil {
		return utils.NewError("failed to create ingress: %w", err)
	}

	utils.PrintSuccess("Ingress for domain %s created successfully\n", ingressResp.DomainName)
	return nil
}

func handleListIngresses(c *cli.Context) error {
	ingresses, err := api.ListIngresses()
	if err != nil {
		return utils.NewError("failed to list ingresses: %w", err)
	}

	if len(ingresses) == 0 {
		utils.PrintInfo("No ingresses found")
		return nil
	}

	utils.PrintHeader("Ingresses")
	for _, ing := range ingresses {
		utils.PrintStatusLine("Ingress", ing.DomainName)
		utils.PrintStatusLine("ID", ing.IngressID.String())
		utils.PrintStatusLine("Deployment ID", ing.DeploymentID.String())
		utils.PrintStatusLine("Service ID", ing.ServiceID.String())
		utils.PrintStatusLine("Namespace", ing.Namespace)
		utils.PrintStatusLine("DNS Config", string(ing.DnsConfig))
		utils.PrintStatusLine("Port", fmt.Sprintf("%d", ing.Port))
		utils.PrintStatusLine("Created", api.FormatTimeAgo(ing.CreatedAt))
		utils.PrintStatusLine("Last Updated", api.FormatTimeAgo(ing.UpdatedAt))
		utils.PrintDivider()
	}
	return nil
}

// TODO: get data by id first before deleting to pass in the payload
func handleDeleteIngress(c *cli.Context) error {
	ingressID := c.String("ingress-id")
	if err := api.DeleteIngress(nil, ingressID); err != nil {
		return utils.NewError("failed to delete ingress: %w", err)
	}

	utils.PrintSuccess("Ingress %s deleted successfully\n", ingressID)
	return nil
}
