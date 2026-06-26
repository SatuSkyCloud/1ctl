package deploy

import (
	"fmt"
	"time"

	"1ctl/internal/api"
	"1ctl/internal/utils"
)

// PublicURLReadiness holds the result of DNS/domain readiness checking.
type PublicURLReadiness struct {
	Ready  bool
	Reason string
}

// WaitForPublicURL checks DNS propagation and domain readiness for a deployment
// identified by its ingress ID and domain name. When ingressID is empty the check
// is skipped and the result is reported as ready (smoke will catch any issues).
func WaitForPublicURL(ingressID, domain string) PublicURLReadiness {
	if ingressID == "" || domain == "" {
		return PublicURLReadiness{Ready: true}
	}

	r := PublicURLReadiness{Ready: true}
	if _, err := api.WaitForIngressDNSStatus(ingressID, 2*time.Minute); err != nil {
		r.Ready = false
		r.Reason = fmt.Sprintf("DNS propagation timed out: %s", err.Error())
		utils.PrintWarning("DNS is still propagating for https://%s: %s", domain, err.Error())
	}

	status, err := api.GetDomainStatus(ingressID, domain, false)
	if err != nil {
		if r.Ready {
			r.Ready = false
			r.Reason = fmt.Sprintf("domain status unavailable: %s", err.Error())
		}
		return r
	}
	if !domainStatusReady(status) {
		r.Ready = false
		r.Reason = domainStatusReason(status)
	}
	return r
}

// ResolveIngressID looks up an ingress by deployment ID and returns its ID as a
// string, or empty string if no ingress is found. This is useful for commands
// (e.g. marketplace deploy) whose response does not include the ingress ID.
func ResolveIngressID(deploymentID string) string {
	ing, err := api.GetIngressByDeploymentID(deploymentID)
	if err != nil || ing == nil {
		return ""
	}
	return ing.IngressID.String()
}

// ReportDeployResult prints the deployment result with optional DNS/smoke
// testing. It handles three cases:
//   - No domain: deployment was accepted, print success.
//   - Domain but URL not ready: print success + warning + next-step hint.
//   - Domain and URL ready: run HTTP smoke probe (polls up to 30s).
func ReportDeployResult(appLabel, deploymentID, domain string, ready PublicURLReadiness, smokePath string, strictSmoke bool) error {
	utils.PrintStatusLine("Deployment ID", deploymentID)

	if !ready.Ready || domain == "" {
		utils.PrintSuccess("Deployment for %s was accepted by the platform.", appLabel)
		if domain != "" {
			utils.PrintWarning("Public URL is not ready yet: https://%s", domain)
			if ready.Reason != "" {
				utils.PrintStatusLine("Public URL reason", ready.Reason)
			}
			utils.PrintInfo("Run: 1ctl domains check %s --probe", domain)
		}
		return nil
	}

	smokeURL := "https://" + domain
	smokePaths := SmokePathCandidates(smokePath)
	utils.PrintInfo("Waiting for app to respond at %s...", smokeURL)

	var smoke PublicURLSmokeResult
	deadline := time.Now().Add(30 * time.Second)
	for {
		smoke = CheckPublicURLSmoke(smokeURL, smokePaths, strictSmoke)
		if smoke.Ready || time.Now().After(deadline) {
			break
		}
		time.Sleep(3 * time.Second)
	}

	if smoke.Ready {
		utils.PrintSuccess("🚀 Deployment for %s is successful! Your app is live at: https://%s", appLabel, domain)
		if smoke.Path != "" {
			utils.PrintInfo("Verified: %s%s", smokeURL, smoke.Path)
		}
	} else {
		utils.PrintWarning("App is starting up — not reachable yet at https://%s", domain)
		utils.PrintInfo("The platform accepted your deployment and pods are starting.")
		utils.PrintInfo("Check status: 1ctl doctor --deployment-id %s", deploymentID)
		if strictSmoke {
			return utils.NewError(fmt.Sprintf("smoke check failed for https://%s: %s", domain, smoke.Reason), nil)
		}
	}

	return nil
}

// --- Internal helpers ---------------------------------------------------

func domainStatusReady(status *api.DomainStatusResponse) bool {
	return status != nil &&
		status.Attached &&
		status.Route.Attached &&
		status.DNS.Status == api.DNSStatusResolved
}

func domainStatusReason(status *api.DomainStatusResponse) string {
	if status == nil {
		return "domain status unavailable"
	}
	if !status.Attached {
		return "domain is not attached in backend metadata"
	}
	if !status.Route.Attached {
		if status.Route.Message != "" {
			return "route is not attached: " + status.Route.Message
		}
		return "route is not attached"
	}
	if status.DNS.Status != api.DNSStatusResolved {
		if status.DNS.Message != "" {
			return fmt.Sprintf("DNS is %s: %s", status.DNS.Status, status.DNS.Message)
		}
		return fmt.Sprintf("DNS is %s", status.DNS.Status)
	}
	return "public URL is not ready"
}
