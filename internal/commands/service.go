package commands

import (
	"context"
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
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
		Usage: "Create or update a Kubernetes Service (low-level — deploy creates services automatically)",
		// Hidden from --help and shell completion: Kubernetes Service is a
		// deploy implementation detail. `1ctl deploy` already creates/updates
		// services as part of the orchestration. The command still works for
		// scripts that depend on it.
		Hidden: true,
		Flags:  serviceFlags,
		Commands: []*cli.Command{
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
					&cli.BoolFlag{
						Name:    "yes",
						Aliases: []string{"y"},
						Usage:   "Skip confirmation prompt",
					},
				},
				Action: handleDeleteService,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// If no subcommand is provided and no flags, show help
			if cmd.NArg() == 0 && !cmd.IsSet("deployment-id") && !cmd.IsSet("name") && !cmd.IsSet("port") {
				return cli.ShowCommandHelp(ctx, cmd, "service")
			}

			// If subcommand is provided, let it handle
			if cmd.NArg() > 0 {
				return cli.ShowSubcommandHelp(cmd)
			}

			// Otherwise, handle service upsert
			return handleUpsertService(ctx, cmd)
		},
	}
}

func handleUpsertService(ctx context.Context, cmd *cli.Command) error {
	// Check required flags for service upsert
	if cmd.String("deployment-id") == "" {
		return utils.NewError("--deployment-id flag is required for service", nil)
	}
	if cmd.String("name") == "" {
		return utils.NewError("--name flag is required for service", nil)
	}
	if cmd.Int("port") == 0 {
		return utils.NewError("--port flag is required for service", nil)
	}

	deploymentIDStr := cmd.String("deployment-id")
	deploymentID, err := uuid.Parse(deploymentIDStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	port, err := api.SafeInt32(cmd.Int("port"))
	if err != nil {
		return utils.NewError("Invalid port number", err)
	}
	service := api.Service{
		DeploymentID: deploymentID,
		ServiceName:  cmd.String("name"),
		Port:         port,
		Namespace:    cmd.String("namespace"),
	}

	var serviceID string
	if err := api.UpsertService(service, &serviceID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to upsert service: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Service %s upserted successfully (ID: %s)\n", service.ServiceName, serviceID)
	return nil
}

func handleListServices(ctx context.Context, cmd *cli.Command) error {
	services, err := api.ListServices()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list services: %s", err.Error()), nil)
	}

	if len(services) == 0 {
		utils.PrintInfo("No services found")
		return nil
	}

	if utils.TryPrintJSON(services) {
		return nil
	}

	headers := []string{"NAME", "SERVICE ID", "DEPLOYMENT ID", "PORT", "UPDATED"}
	rows := make([][]string, 0, len(services))
	for _, svc := range services {
		rows = append(rows, []string{
			svc.ServiceName,
			svc.ServiceID.String(),
			svc.DeploymentID.String(),
			fmt.Sprintf("%d", svc.Port),
			api.FormatTimeAgo(svc.UpdatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleDeleteService(ctx context.Context, cmd *cli.Command) error {
	serviceID := cmd.String("service-id")
	if !utils.Confirm(fmt.Sprintf("Delete service %s? This cannot be undone.", serviceID), cmd.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteService(serviceID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete service: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Service %s deleted successfully\n", serviceID)
	return nil
}
