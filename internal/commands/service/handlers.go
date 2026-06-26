package service

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	"1ctl/internal/utils"

	"github.com/google/uuid"
)

// --- Handlers -----------------------------------------------------------

func handleUpsertService(ctx context.Context, in serviceUpsertInput) error {
	if in.DeploymentID == "" {
		return utils.NewError("--deployment-id flag is required for service", nil)
	}
	if in.Name == "" {
		return utils.NewError("--name flag is required for service", nil)
	}
	if in.Port == 0 {
		return utils.NewError("--port flag is required for service", nil)
	}

	deploymentID, err := uuid.Parse(in.DeploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	port, err := api.SafeInt32(in.Port)
	if err != nil {
		return utils.NewError("Invalid port number", err)
	}
	svc := api.Service{
		DeploymentID: deploymentID,
		ServiceName:  in.Name,
		Port:         port,
		Namespace:    in.Namespace,
	}

	var serviceID string
	if err := api.UpsertService(svc, &serviceID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to upsert service: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Service %s upserted successfully (ID: %s)\n", svc.ServiceName, serviceID)
	return nil
}

func handleListServices(ctx context.Context) error {
	services, err := api.ListServices()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list services: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(services, "No services found") {
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
			utils.FormatTimeAgo(svc.UpdatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleDeleteService(ctx context.Context, in serviceDeleteInput) error {
	if !utils.Confirm(fmt.Sprintf("Delete service %s? This cannot be undone.", in.ServiceID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteService(in.ServiceID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete service: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Service %s deleted successfully\n", in.ServiceID)
	return nil
}
