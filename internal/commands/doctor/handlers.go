package doctor

import (
	"context"
	"fmt"
	"strings"

	"1ctl/internal/api"
	"1ctl/internal/config"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/deploy"
	"1ctl/internal/utils"
	"1ctl/internal/validator"
)

// --- Report types -------------------------------------------------------

type doctorReport struct {
	UserEmail    string                   `json:"user_email"`
	Organization string                   `json:"organization"`
	Namespace    string                   `json:"namespace"`
	Zones        int                      `json:"zones"`
	Clusters     int                      `json:"clusters"`
	Deployments  []doctorDeploymentReport `json:"deployments"`
	Issues       []string                 `json:"issues,omitempty"`
}

type doctorDeploymentReport struct {
	DeploymentID string                    `json:"deployment_id"`
	AppLabel     string                    `json:"app_label"`
	Domain       string                    `json:"domain,omitempty"`
	Status       string                    `json:"status"`
	DomainStatus *api.DomainStatusResponse `json:"domain_status,omitempty"`
	Smoke        *deploy.PublicURLSmokeResult `json:"smoke,omitempty"`
}

// --- Handlers -----------------------------------------------------------

func handleDoctor(ctx context.Context, in doctorInput) error {
	user, err := api.GetCurrentUser()
	if err != nil {
		return utils.NewError(fmt.Sprintf("auth/backend check failed: %s", err.Error()), nil)
	}

	namespace := satuskyctx.GetCurrentNamespace()
	if namespace == "" {
		return utils.NewError("not authenticated or namespace missing", nil)
	}

	if err := validator.ValidateURLPath(in.HealthPath); err != nil {
		return utils.NewError(fmt.Sprintf("invalid health path: %v", err), nil)
	}

	report := doctorReport{
		UserEmail:    user.Email,
		Organization: user.Organization,
		Namespace:    namespace,
	}

	if zones, err := api.GetAvailableZones(); err != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("zones: %s", err.Error()))
	} else {
		report.Zones = len(zones)
	}

	if clusters, err := api.GetClusters(); err != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("clusters: %s", err.Error()))
	} else {
		report.Clusters = len(clusters)
	}

	targets, smokePath, targetedMode, err := resolveDoctorTargets(in)
	if err != nil {
		return err
	}

	runSmoke := targetedMode || in.Smoke
	strictSmoke := in.HealthPath != ""

	for _, dep := range targets {
		resolvedDomain := strings.TrimSpace(dep.Domain)
		if resolvedDomain == "" {
			if ing, ingErr := api.GetIngressByDeploymentID(dep.DeploymentID.String()); ingErr == nil && ing != nil {
				resolvedDomain = strings.TrimSpace(ing.DomainName)
			}
		}

		entry := doctorDeploymentReport{
			DeploymentID: dep.DeploymentID.String(),
			AppLabel:     dep.AppLabel,
			Domain:       resolvedDomain,
			Status:       dep.Status,
		}

		if resolvedDomain != "" {
			if ing, ingErr := api.GetIngressByDomainName(resolvedDomain); ingErr != nil {
				report.Issues = append(report.Issues, fmt.Sprintf("%s domain lookup: %s", dep.AppLabel, ingErr.Error()))
			} else if ds, dsErr := api.GetDomainStatus(ing.IngressID.String(), resolvedDomain, true); dsErr != nil {
				report.Issues = append(report.Issues, fmt.Sprintf("%s domain status: %s", dep.AppLabel, dsErr.Error()))
			} else {
				entry.DomainStatus = ds
			}

			if runSmoke {
				result := smokeDeploymentURL(resolvedDomain, smokePath, strictSmoke)
				entry.Smoke = result
				if result != nil && !result.Ready {
					report.Issues = append(report.Issues, fmt.Sprintf("%s smoke: %s", dep.AppLabel, result.Reason))
				}
			}
		}

		report.Deployments = append(report.Deployments, entry)
	}

	if utils.TryPrintJSON(report) {
		if len(report.Issues) > 0 {
			return utils.NewError(fmt.Sprintf("doctor found %d issue(s)", len(report.Issues)), nil)
		}
		return nil
	}

	utils.PrintHeader("SatuSky Doctor")
	utils.PrintStatusLine("User", report.UserEmail)
	utils.PrintStatusLine("Organization", report.Organization)
	utils.PrintStatusLine("Namespace", report.Namespace)
	utils.PrintStatusLine("Zones", fmt.Sprintf("%d available", report.Zones))
	utils.PrintStatusLine("Clusters", fmt.Sprintf("%d available", report.Clusters))

	if len(report.Deployments) == 0 {
		utils.PrintInfo("No deployments found in this namespace")
	} else {
		utils.PrintHeader("Deployments")
		for _, dep := range report.Deployments {
			utils.PrintStatusLine(dep.AppLabel, dep.Status)
			if dep.Domain != "" {
				utils.PrintStatusLine("  Domain", dep.Domain)
				if dep.DomainStatus != nil {
					utils.PrintStatusLine("  Route", domainRouteText(dep.DomainStatus.Route))
					utils.PrintStatusLine("  DNS", domainDNSText(dep.DomainStatus.DNS))
					utils.PrintStatusLine("  TLS", domainTLSText(dep.DomainStatus.TLS))
					utils.PrintStatusLine("  HTTP", domainHTTPText(dep.DomainStatus.Reachability))
				}
				if dep.Smoke != nil {
					if dep.Smoke.Ready {
						if dep.Smoke.Path != "" {
							utils.PrintStatusLine("  Smoke", fmt.Sprintf("ok at %s%s", "https://"+dep.Domain, dep.Smoke.Path))
						} else {
							utils.PrintStatusLine("  Smoke", fmt.Sprintf("ok at https://%s", dep.Domain))
						}
					} else {
						utils.PrintStatusLine("  Smoke", "failed: "+dep.Smoke.Reason)
					}
				}
			} else {
				utils.PrintStatusLine("  Domain", "not attached")
			}
		}
	}

	if len(report.Issues) > 0 {
		utils.PrintWarning("Issues found:")
		for _, issue := range report.Issues {
			utils.PrintWarning(" - %s", issue)
		}
		return utils.NewError(fmt.Sprintf("doctor found %d issue(s)", len(report.Issues)), nil)
	}

	utils.PrintSuccess("No issues found")
	return nil
}

// --- Target resolution --------------------------------------------------

func resolveDoctorTargets(in doctorInput) ([]api.Deployment, string, bool, error) {
	healthPath := strings.TrimSpace(in.HealthPath)

	if in.DeploymentID != "" || in.Config != "" {
		deploymentID, err := deploy.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
		if err != nil {
			return nil, "", false, err
		}
		deployment, err := api.GetDeployment(deploymentID)
		if err != nil {
			return nil, "", false, utils.NewError(fmt.Sprintf("failed to load deployment %s: %s", deploymentID, err.Error()), nil)
		}
		if healthPath == "" && in.Config != "" {
			if cfg, cfgErr := config.FindConfig(in.Config); cfgErr == nil && cfg != nil {
				healthPath = strings.TrimSpace(cfg.Checks.HealthPath)
			}
		}
		return []api.Deployment{*deployment}, healthPath, true, nil
	}

	deployments, err := api.ListDeployments()
	if err != nil {
		return nil, "", false, utils.NewError(fmt.Sprintf("failed to list deployments: %s", err.Error()), nil)
	}
	return deployments, healthPath, false, nil
}

// --- Smoke helpers ------------------------------------------------------

func smokeDeploymentURL(domain, healthPath string, strict bool) *deploy.PublicURLSmokeResult {
	if domain == "" {
		return nil
	}
	candidates := deploy.SmokePathCandidates(healthPath)
	result := deploy.CheckPublicURLSmoke("https://"+domain, candidates, strict)
	return &result
}

// --- Domain status text helpers -----------------------------------------

func domainRouteText(status api.DomainRouteStatus) string {
	if !status.Attached {
		if status.Message != "" {
			return "not attached: " + status.Message
		}
		return "not attached"
	}
	if status.ResourceKind == "" && status.ResourceName == "" {
		return "attached"
	}
	return fmt.Sprintf("attached to %s/%s", status.ResourceKind, status.ResourceName)
}

func domainDNSText(status api.DNSStatusResponse) string {
	parts := []string{string(status.Status)}
	if len(status.ResolvedIPs) > 0 {
		parts = append(parts, "resolved "+strings.Join(status.ResolvedIPs, ", "))
	}
	if status.ExpectedIP != "" {
		parts = append(parts, "expected "+status.ExpectedIP)
	}
	if status.Message != "" {
		parts = append(parts, status.Message)
	}
	return strings.Join(parts, " - ")
}

func domainTLSText(status api.TLSStatusResponse) string {
	if status.Status == "" {
		return "unknown"
	}
	if status.ExpiresAt != nil {
		return fmt.Sprintf("%s, expires %s", status.Status, status.ExpiresAt.Format("2006-01-02"))
	}
	if status.Message != "" {
		return fmt.Sprintf("%s - %s", status.Status, status.Message)
	}
	return string(status.Status)
}

func domainHTTPText(status api.DomainReachabilityStatus) string {
	if !status.Checked {
		if status.Message != "" {
			return "not checked - " + status.Message
		}
		return "not checked"
	}
	if status.Reachable {
		return fmt.Sprintf("reachable %s %d", status.URL, status.StatusCode)
	}
	if status.StatusCode > 0 {
		if status.Message != "" {
			return fmt.Sprintf("unreachable: %d - %s", status.StatusCode, status.Message)
		}
		return fmt.Sprintf("unreachable: %d", status.StatusCode)
	}
	if status.Message != "" {
		return "unreachable - " + status.Message
	}
	return "unreachable"
}
