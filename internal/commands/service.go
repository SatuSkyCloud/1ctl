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
	port, err := api.SafeInt32(c.Int("port"))
	if err != nil {
		return utils.NewError("Invalid port number", err)
	}
	service := api.Service{
		DeploymentID: uuid.MustParse(c.String("deployment-id")),
		ServiceName:  c.String("name"),
		Port:         port,
		Namespace:    c.String("namespace"),
	}

	var serviceID string
	if err := api.CreateService(service, &serviceID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to create service: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Service %s created successfully (ID: %s)\n", service.ServiceName, serviceID)
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
