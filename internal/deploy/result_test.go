package deploy

import (
	"1ctl/internal/api"
	"strings"
	"testing"
	"time"
)

func TestWaitForPublicURLSkipsLocalhostDNSPolling(t *testing.T) {
	for _, domain := range []string{"localhost", "demo.localhost", "DEMO.LOCALHOST."} {
		result := WaitForPublicURL("ingress-id", domain)
		if result.Ready || result.Reason == "" {
			t.Fatalf("WaitForPublicURL(%q) = %+v, want immediate local-only result", domain, result)
		}
	}
}

func readyDomain(name string) *api.DomainStatusResponse {
	return &api.DomainStatusResponse{DomainName: name, Attached: true, Route: api.DomainRouteStatus{Attached: true}, DNS: api.DNSStatusResponse{Domain: name, Status: api.DNSStatusResolved, Condition: &api.DNSCondition{Status: api.DNSConditionStatusVerified}}}
}

func TestPublicURLPollsRequestedHostnameUntilVerified(t *testing.T) {
	calls := 0
	result := waitForPublicURL("ingress", "app.example.com", time.Second, time.Millisecond, func(id, domain string, probe bool, budget time.Duration) (*api.DomainStatusResponse, error) {
		calls++
		if id != "ingress" || domain != "app.example.com" || probe || budget <= 0 {
			t.Fatal("incorrect poll request")
		}
		status := readyDomain(domain)
		if calls == 1 {
			status.DNS.Condition.Status = api.DNSConditionStatusPending
		}
		if calls == 2 {
			status.Route.Attached = false
		}
		return status, nil
	})
	if !result.Ready || calls != 3 {
		t.Fatalf("result=%+v calls=%d", result, calls)
	}
}

func TestPublicURLNeverAcceptsMissingOrWrongHostnameEvidence(t *testing.T) {
	for _, name := range []string{"", "other.example.com"} {
		result := waitForPublicURL("ingress", "app.example.com", 10*time.Millisecond, time.Millisecond, func(string, string, bool, time.Duration) (*api.DomainStatusResponse, error) {
			return readyDomain(name), nil
		})
		if result.Ready || !strings.Contains(result.Reason, "different or missing hostname") {
			t.Fatalf("result=%+v", result)
		}
	}
	for _, pair := range [][2]string{{"", "app.example.com"}, {"ingress", ""}} {
		if WaitForPublicURL(pair[0], pair[1]).Ready {
			t.Fatal("missing ingress/domain cannot prove readiness")
		}
	}
}

func TestPublicURLReportsLastDNSCondition(t *testing.T) {
	result := waitForPublicURL("ingress", "app.example.com", 10*time.Millisecond, time.Millisecond, func(string, string, bool, time.Duration) (*api.DomainStatusResponse, error) {
		status := readyDomain("app.example.com")
		status.DNS.Condition = &api.DNSCondition{Status: api.DNSConditionStatusWrongTarget, Code: "DNS_WRONG_TARGET"}
		return status, nil
	})
	if result.Ready || !strings.Contains(result.Reason, "DNS_WRONG_TARGET") {
		t.Fatalf("result=%+v", result)
	}
}

func TestPublicURLStopsOnForbiddenRatherThanCallingItPropagation(t *testing.T) {
	calls := 0
	result := waitForPublicURL("ingress", "app.example.com", time.Second, time.Millisecond, func(string, string, bool, time.Duration) (*api.DomainStatusResponse, error) {
		calls++
		return nil, &api.HTTPStatusError{StatusCode: 403, Message: "Access denied"}
	})
	if result.Ready || calls != 1 || !strings.Contains(result.Reason, "Access denied") {
		t.Fatalf("result=%+v calls=%d", result, calls)
	}
}
