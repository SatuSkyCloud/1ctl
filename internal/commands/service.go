package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func ServiceCommand() *cli.Command {
	return &cli.Command{
		Name:  "service",
		Usage: "Manage services",
		Subcommands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Create a new service",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "deployment-id",
						Usage:    "Deployment ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "name",
						Usage:    "Service name",
						Required: true,
					},
					&cli.IntFlag{
						Name:     "port",
						Usage:    "Service port",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "namespace",
						Usage: "User's organization name",
					},
				},
				Action: handleCreateService,
			},
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
	}
}

func handleCreateService(c *cli.Context) error {
	service := api.Service{
		DeploymentID: uuid.MustParse(c.String("deployment-id")),
		ServiceName:  c.String("name"),
		Port:         int32(c.Int("port")),
		Namespace:    c.String("namespace"),
	}

	var serviceID string
	if err := api.CreateService(service, &serviceID); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	utils.PrintSuccess("Service %s created successfully (ID: %s)\n", service.ServiceName, serviceID)
	return nil
}

func handleListServices(c *cli.Context) error {
	services, err := api.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
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
		return fmt.Errorf("failed to delete service: %w", err)
	}

	utils.PrintSuccess("Service %s deleted successfully\n", serviceID)
	return nil
}
