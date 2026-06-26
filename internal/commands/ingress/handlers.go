package ingress

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	"1ctl/internal/utils"

	"github.com/google/uuid"
)

// --- Handlers -----------------------------------------------------------

func handleUpsertIngress(ctx context.Context, in ingressUpsertInput) error {
	if in.DeploymentID == "" {
		return utils.NewError("--deployment-id flag is required for ingress", nil)
	}
	if in.Domain == "" {
		return utils.NewError("--domain flag is required for ingress", nil)
	}
	if in.AppLabel == "" {
		return utils.NewError("--app-label flag is required for ingress", nil)
	}
	if in.Namespace == "" {
		return utils.NewError("--namespace flag is required for ingress", nil)
	}

	deploymentID, err := uuid.Parse(in.DeploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	if in.ServiceID == "" {
		return utils.NewError("--service-id flag is required for ingress", nil)
	}
	serviceID, err := uuid.Parse(in.ServiceID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid service-id: %s", err.Error()), nil)
	}

	port, err := api.SafeInt32(in.Port)
	if err != nil {
		return utils.NewError("Invalid port number", err)
	}

	dnsConfig := api.DnsConfigDefault
	if in.CustomDNS {
		dnsConfig = api.DnsConfigCustom
	}

	ingress := api.Ingress{
		DeploymentID: deploymentID,
		ServiceID:    serviceID,
		AppLabel:     in.AppLabel,
		Namespace:    in.Namespace,
		DomainName:   in.Domain,
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

func handleListIngresses(ctx context.Context) error {
	ingresses, err := api.ListIngresses()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list ingresses: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(ingresses, "No ingresses found") {
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
			utils.FormatTimeAgo(ing.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleDeleteIngress(ctx context.Context, in ingressDeleteInput) error {
	if !utils.Confirm(fmt.Sprintf("Delete ingress %s? This cannot be undone.", in.IngressID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteIngress(in.IngressID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete ingress: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Ingress %s deleted successfully\n", in.IngressID)
	return nil
}
