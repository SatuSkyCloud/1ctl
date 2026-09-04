package deploy

import "testing"

func TestWaitForPublicURLSkipsLocalhostDNSPolling(t *testing.T) {
	for _, domain := range []string{"localhost", "demo.localhost", "DEMO.LOCALHOST."} {
		result := WaitForPublicURL("ingress-id", domain)
		if result.Ready || result.Reason == "" {
			t.Fatalf("WaitForPublicURL(%q) = %+v, want immediate local-only result", domain, result)
		}
	}
}
