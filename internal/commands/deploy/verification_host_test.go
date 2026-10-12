package deploy

import (
	"testing"

	"1ctl/internal/api"

	"github.com/google/uuid"
)

// The verifier used the ingress's own DomainName, which is the managed
// <random>.satusky.com hostname and is fronted by Cloudflare. Probing that from a
// runner tested Cloudflare's treatment of the runner's IP rather than the
// deployment, and it never passed — so every workflow fell back to
// --wait-mode workload. These pin the precedence that fixes it.
func TestPickVerificationHostPrefersCustomAlias(t *testing.T) {
	ingress := &api.Ingress{IngressID: uuid.New(), DomainName: "abc123.satusky.com"}
	got := pickVerificationHost(ingress, []api.IngressAlias{
		{DomainName: "app.example.com"},
	})
	if got != "app.example.com" {
		t.Fatalf("expected the custom alias, got %q", got)
	}
}

func TestPickVerificationHostSkipsRedirectAliases(t *testing.T) {
	ingress := &api.Ingress{IngressID: uuid.New(), DomainName: "abc123.satusky.com"}
	got := pickVerificationHost(ingress, []api.IngressAlias{
		{DomainName: "www.example.com", IsRedirect: true},
		{DomainName: "app.example.com"},
	})
	if got != "app.example.com" {
		t.Fatalf("a www->apex alias must not be probed; got %q", got)
	}
}

func TestPickVerificationHostIgnoresManagedAliases(t *testing.T) {
	ingress := &api.Ingress{IngressID: uuid.New(), DomainName: "abc123.satusky.com"}
	for _, alias := range []string{"satusky.com", "other.satusky.com", "SATUSKY.COM"} {
		if got := pickVerificationHost(ingress, []api.IngressAlias{{DomainName: alias}}); got != "abc123.satusky.com" {
			t.Errorf("managed alias %q should be skipped; got %q", alias, got)
		}
	}
}

func TestPickVerificationHostFallsBackToTheIngress(t *testing.T) {
	ingress := &api.Ingress{IngressID: uuid.New(), DomainName: "abc123.satusky.com"}
	if got := pickVerificationHost(ingress, nil); got != "abc123.satusky.com" {
		t.Fatalf("no aliases must fall back to the ingress hostname; got %q", got)
	}
	if got := pickVerificationHost(ingress, []api.IngressAlias{{DomainName: "  "}}); got != "abc123.satusky.com" {
		t.Fatalf("a blank alias must fall back; got %q", got)
	}
	if got := pickVerificationHost(nil, nil); got != "" {
		t.Fatalf("no ingress must yield no host; got %q", got)
	}
}
