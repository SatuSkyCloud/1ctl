package deploy

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
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
// identified by its ingress ID and domain name. Localhost domains return
// immediately because they are local-profile placeholders, not public DNS.
// Checks the requested hostname, not an ingress's potentially different alias.
func WaitForPublicURL(ingressID, domain string) PublicURLReadiness {
	return waitForPublicURL(ingressID, domain, 2*time.Minute, 3*time.Second, api.GetDomainStatusWithTimeout)
}

func waitForPublicURL(ingressID, domain string, timeout, interval time.Duration, fetch func(string, string, bool, time.Duration) (*api.DomainStatusResponse, error)) PublicURLReadiness {
	if domain == "" {
		return PublicURLReadiness{Reason: "deployment has no public hostname yet"}
	}
	if isLocalhostDomain(domain) {
		return PublicURLReadiness{
			Ready:  false,
			Reason: "localhost domains are local-profile placeholders and are not publicly reachable",
		}
	}
	if ingressID == "" {
		return PublicURLReadiness{Reason: "ingress metadata is not available yet"}
	}
	deadline := time.Now().Add(timeout)
	lastReason := "domain status has not been observed"
	for time.Until(deadline) > 0 {
		status, err := fetch(ingressID, domain, false, time.Until(deadline))
		if err != nil {
			lastReason = "domain status unavailable: " + err.Error()
			var statusErr *api.HTTPStatusError
			if errors.As(err, &statusErr) && statusErr.StatusCode >= 400 && statusErr.StatusCode < 500 && statusErr.StatusCode != http.StatusNotFound && statusErr.StatusCode != http.StatusTooManyRequests {
				return PublicURLReadiness{Reason: lastReason}
			}
		} else if status != nil && (normalizeDomain(status.DomainName) != normalizeDomain(domain) || (status.DNS.Domain != "" && normalizeDomain(status.DNS.Domain) != normalizeDomain(domain))) {
			lastReason = "backend returned status for a different or missing hostname"
		} else if domainStatusReady(status) {
			return PublicURLReadiness{Ready: true}
		} else {
			lastReason = domainStatusReason(status)
		}
		delay := min(interval, time.Until(deadline))
		if delay <= 0 {
			break
		}
		time.Sleep(delay)
	}
	return PublicURLReadiness{Reason: fmt.Sprintf("timed out waiting for %s: %s", domain, lastReason)}
}

func normalizeDomain(domain string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
}

// PublicDomainReady binds readiness evidence to the requested hostname.
func PublicDomainReady(status *api.DomainStatusResponse, domain string) bool {
	return normalizeDomain(domain) != "" && status != nil &&
		normalizeDomain(status.DomainName) == normalizeDomain(domain) &&
		(status.DNS.Domain == "" || normalizeDomain(status.DNS.Domain) == normalizeDomain(domain)) && domainStatusReady(status)
}

func isLocalhostDomain(domain string) bool {
	normalized := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	return normalized == "localhost" || strings.HasSuffix(normalized, ".localhost")
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
//   - No domain: deployment was accepted, print its acceptance.
//   - Domain but URL not ready: print acceptance + warning + next-step hint.
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
		if strictSmoke {
			return utils.NewError("public URL verification failed: "+ready.Reason, nil)
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
		utils.PrintSuccess("Public smoke check for %s succeeded at: https://%s", appLabel, domain)
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
		status.DNS.Status == api.DNSStatusResolved &&
		(status.DNS.Condition == nil || status.DNS.Condition.Status == api.DNSConditionStatusVerified)
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
	if status.DNS.Condition != nil && status.DNS.Condition.Status != api.DNSConditionStatusVerified {
		return fmt.Sprintf("DNS is %s (%s): %s", status.DNS.Condition.Status, status.DNS.Condition.Code, status.DNS.Message)
	}
	if status.DNS.Status != api.DNSStatusResolved {
		if status.DNS.Message != "" {
			return fmt.Sprintf("DNS is %s: %s", status.DNS.Status, status.DNS.Message)
		}
		return fmt.Sprintf("DNS is %s", status.DNS.Status)
	}
	return "public URL is not ready"
}
