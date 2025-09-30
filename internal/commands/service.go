package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func ServiceCommand() *cli.Command {
	serviceFlags := []cli.Flag{
		&cli.StringFlag{
			Name:     "deployment-id",
			Usage:    "Deployment ID",
			Required: false, // Remove required at command level, validate in handler
		},
		&cli.StringFlag{
			Name:     "name",
			Usage:    "Service name",
			Required: false, // Remove required at command level, validate in handler
		},
		&cli.IntFlag{
			Name:     "port",
			Usage:    "Service port",
			Required: false, // Remove required at command level, validate in handler
		},
		&cli.StringFlag{
			Name:  "namespace",
			Usage: "User's organization name",
		},
	}

	return &cli.Command{
		Name:  "service",
		Usage: "Create or update a service",
		Flags: serviceFlags,
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List all services",
				Action: handleListServices,
			},
			{
				Name:  "delete",
				Usage: "Delete a service",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "service-id",
						Usage:    "Service ID to delete",
						Required: true,
					},
				},
				Action: handleDeleteService,
			},
		},
		Action: func(c *cli.Context) error {
			// If no subcommand is provided and no flags, show help
			if c.NArg() == 0 && !c.IsSet("deployment-id") && !c.IsSet("name") && !c.IsSet("port") {
				return cli.ShowCommandHelp(c, "service")
			}

			// If subcommand is provided, let it handle
			if c.NArg() > 0 {
				return cli.ShowSubcommandHelp(c)
			}

			// Otherwise, handle service upsert
			return handleUpsertService(c)
		},
	}
}

func handleUpsertService(c *cli.Context) error {
	// Check required flags for service upsert
	if c.String("deployment-id") == "" {
		return utils.NewError("--deployment-id flag is required for service", nil)
	}
	if c.String("name") == "" {
		return utils.NewError("--name flag is required for service", nil)
	}
	if c.Int("port") == 0 {
		return utils.NewError("--port flag is required for service", nil)
	}

	deploymentIDStr := c.String("deployment-id")
	deploymentID, err := uuid.Parse(deploymentIDStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	port, err := api.SafeInt32(c.Int("port"))
	if err != nil {
		return utils.NewError("Invalid port number", err)
	}
	service := api.Service{
		DeploymentID: deploymentID,
		ServiceName:  c.String("name"),
		Port:         port,
		Namespace:    c.String("namespace"),
	}

	var serviceID string
	if err := api.UpsertService(service, &serviceID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to upsert service: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Service %s upserted successfully (ID: %s)\n", service.ServiceName, serviceID)
	return nil
}

func handleListServices(c *cli.Context) error {
	services, err := api.ListServices()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list services: %s", err.Error()), nil)
	}

	if len(services) == 0 {
		utils.PrintInfo("No services found")
		return nil
	}

	utils.PrintHeader("Services")
	for _, svc := range services {
		utils.PrintStatusLine("Service", svc.ServiceName)
		utils.PrintStatusLine("ID", svc.ServiceID.String())
		utils.PrintStatusLine("Deployment ID", svc.DeploymentID.String())
		utils.PrintStatusLine("Namespace", svc.Namespace)
		utils.PrintStatusLine("Port", fmt.Sprintf("%d", svc.Port))
		utils.PrintStatusLine("Created", api.FormatTimeAgo(svc.CreatedAt))
		utils.PrintStatusLine("Last Updated", api.FormatTimeAgo(svc.UpdatedAt))
		utils.PrintDivider()
	}
	return nil
}

// TODO: get data by id first before deleting to pass in the payload
func handleDeleteService(c *cli.Context) error {
	serviceID := c.String("service-id")
	if err := api.DeleteService(nil, serviceID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete service: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Service %s deleted successfully\n", serviceID)
	return nil
}
