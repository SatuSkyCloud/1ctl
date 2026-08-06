package doctor

import (
	"strings"
	"testing"

	"1ctl/internal/api"
)

// A deployment that is failed, unrouted, not resolving and unreachable used to
// produce zero issues, so Doctor printed every one of those lines and then
// summarized them as "No issues found" with a zero exit code.
func TestDeploymentHealthIssuesFlagsBrokenDeployment(t *testing.T) {
	entry := doctorDeploymentReport{
		AppLabel: "broken-app",
		Domain:   "broken.satusky.com",
		Status:   "failed",
		DomainStatus: &api.DomainStatusResponse{
			Route: api.DomainRouteStatus{Attached: false, Message: "HTTPRoute not found"},
			DNS:   api.DNSStatusResponse{Status: api.DNSStatusPropagating},
			Reachability: api.DomainReachabilityStatus{
				Checked: true, Reachable: false, Message: "no such host",
			},
		},
	}

	issues := deploymentHealthIssues(entry)
	if len(issues) != 4 {
		t.Fatalf("expected 4 issues (status, route, dns, http), got %d: %v", len(issues), issues)
	}
	for _, want := range []string{"deployment status", "route", "dns", "http"} {
		if !containsSubstring(issues, want) {
			t.Errorf("expected an issue mentioning %q, got %v", want, issues)
		}
	}
}

func TestDeploymentHealthIssuesSilentOnHealthyDeployment(t *testing.T) {
	entry := doctorDeploymentReport{
		AppLabel: "healthy-app",
		Domain:   "healthy.satusky.com",
		Status:   "ready",
		DomainStatus: &api.DomainStatusResponse{
			Route: api.DomainRouteStatus{Attached: true, ResourceKind: "HTTPRoute", ResourceName: "healthy-app-route"},
			DNS:   api.DNSStatusResponse{Status: api.DNSStatusResolved},
			Reachability: api.DomainReachabilityStatus{
				Checked: true, Reachable: true, StatusCode: 200,
			},
		},
	}

	if issues := deploymentHealthIssues(entry); len(issues) != 0 {
		t.Fatalf("healthy deployment must raise no issues, got %v", issues)
	}
}

// Doctor is routinely run while a rollout is still settling; those states are
// not failures and must not make it exit non-zero.
func TestDeploymentHealthIssuesIgnoresTransientStates(t *testing.T) {
	for _, status := range []string{"pending", "reconciling", "deleting"} {
		entry := doctorDeploymentReport{AppLabel: "app", Status: status}
		if issues := deploymentHealthIssues(entry); len(issues) != 0 {
			t.Errorf("status %q must not be reported as an issue, got %v", status, issues)
		}
	}
}

// A typed condition is authoritative when present: resolved-but-unverified is
// still a failure, and verified passes even though the check would otherwise
// look at Status.
func TestDeploymentHealthIssuesPrefersTypedDNSCondition(t *testing.T) {
	base := func(cond *api.DNSCondition) doctorDeploymentReport {
		return doctorDeploymentReport{
			AppLabel: "app",
			Domain:   "app.example.com",
			Status:   "ready",
			DomainStatus: &api.DomainStatusResponse{
				Route: api.DomainRouteStatus{Attached: true},
				DNS:   api.DNSStatusResponse{Status: api.DNSStatusResolved, Condition: cond},
			},
		}
	}

	unverified := base(&api.DNSCondition{Status: api.DNSConditionStatusWrongTarget})
	if issues := deploymentHealthIssues(unverified); len(issues) != 1 {
		t.Errorf("wrong_target condition must be an issue even when status is resolved, got %v", issues)
	}

	verified := base(&api.DNSCondition{Status: api.DNSConditionStatusVerified})
	if issues := deploymentHealthIssues(verified); len(issues) != 0 {
		t.Errorf("verified condition must raise no issue, got %v", issues)
	}
}

// An unrequested probe proves nothing and must not be reported as unreachable.
func TestDeploymentHealthIssuesIgnoresUncheckedReachability(t *testing.T) {
	entry := doctorDeploymentReport{
		AppLabel: "app",
		Domain:   "app.satusky.com",
		Status:   "ready",
		DomainStatus: &api.DomainStatusResponse{
			Route:        api.DomainRouteStatus{Attached: true},
			DNS:          api.DNSStatusResponse{Status: api.DNSStatusResolved},
			Reachability: api.DomainReachabilityStatus{Checked: false, Reachable: false},
		},
	}

	if issues := deploymentHealthIssues(entry); len(issues) != 0 {
		t.Fatalf("unchecked reachability must not be an issue, got %v", issues)
	}
}

func containsSubstring(values []string, want string) bool {
	for _, v := range values {
		if strings.Contains(v, want) {
			return true
		}
	}
	return false
}
