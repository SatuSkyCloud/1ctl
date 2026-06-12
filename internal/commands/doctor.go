package commands

import (
	"fmt"
	"strings"

	"1ctl/internal/api"
	"1ctl/internal/config"
	"1ctl/internal/context"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v2"
)

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
	Smoke        *publicURLSmokeResult     `json:"smoke,omitempty"`
}

func DoctorCommand() *cli.Command {
	return &cli.Command{
		Name:  "doctor",
		Usage: "Diagnose auth, backend reachability, zones, clusters, and live deployments",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "deployment-id",
				Aliases: []string{"d"},
				Usage:   "Check one deployment instead of the whole namespace",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "Config name or path (e.g. staging, satusky.staging.toml). Used to resolve a deployment ID when not provided.",
			},
			&cli.StringFlag{
				Name:  "health-path",
				Usage: "Explicit HTTP smoke path to use for all checked deployments (default: tries /health then /)",
			},
		},
		Action: handleDoctor,
	}
}

func handleDoctor(c *cli.Context) error {
	user, err := api.GetCurrentUser()
	if err != nil {
		return utils.NewError(fmt.Sprintf("auth/backend check failed: %s", err.Error()), nil)
	}

	namespace := context.GetCurrentNamespace()
	if namespace == "" {
		return utils.NewError("not authenticated or namespace missing", nil)
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

	targets, smokePath, err := resolveDoctorTargets(c)
	if err != nil {
		return err
	}

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

			if smokePath != "" || c.String("deployment-id") != "" || c.String("config") != "" {
				entry.Smoke = smokeDeploymentURL(resolvedDomain, smokePath)
			} else {
				entry.Smoke = smokeDeploymentURL(resolvedDomain, "")
			}
			if entry.Smoke != nil && !entry.Smoke.Ready {
				report.Issues = append(report.Issues, fmt.Sprintf("%s smoke: %s", dep.AppLabel, entry.Smoke.Reason))
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

func resolveDoctorTargets(c *cli.Context) ([]api.Deployment, string, error) {
	healthPath := strings.TrimSpace(c.String("health-path"))

	if c.String("deployment-id") != "" || c.String("config") != "" {
		deploymentID, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
		if err != nil {
			return nil, "", err
		}
		deployment, err := api.GetDeployment(deploymentID)
		if err != nil {
			return nil, "", utils.NewError(fmt.Sprintf("failed to load deployment %s: %s", deploymentID, err.Error()), nil)
		}
		if healthPath == "" && c.String("config") != "" {
			if cfg, cfgErr := config.FindConfig(c.String("config")); cfgErr == nil && cfg != nil {
				healthPath = strings.TrimSpace(cfg.App.HealthPath)
			}
		}
		return []api.Deployment{*deployment}, healthPath, nil
	}

	deployments, err := api.ListDeployments()
	if err != nil {
		return nil, "", utils.NewError(fmt.Sprintf("failed to list deployments: %s", err.Error()), nil)
	}
	return deployments, healthPath, nil
}

func smokeDeploymentURL(domain, healthPath string) *publicURLSmokeResult {
	if domain == "" {
		return nil
	}
	candidates := smokePathCandidates(healthPath)
	result := checkPublicURLSmoke("https://"+domain, candidates)
	return &result
}
