package deploy

import (
	"testing"
	"time"

	"1ctl/internal/api"
	deploypkg "1ctl/internal/deploy"

	"github.com/google/uuid"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func domainStatus(host string, attached bool, dns api.DNSStatus) *api.DomainStatusResponse {
	status := &api.DomainStatusResponse{DomainName: host, Attached: attached}
	status.Route.Attached = attached
	status.DNS.Status = dns
	status.DNS.Domain = host
	return status
}

func testVerifier(strict bool, status *api.DomainStatusResponse, probe func(string, string) deploypkg.PublicURLSmokeResult) (*publicURLVerifier, *fakeClock, *int) {
	clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	probes := 0
	host := "abc123.satusky.com"
	v := &publicURLVerifier{
		deploymentID: "dep-1",
		paths:        deploypkg.SmokePathCandidates(""),
		strict:       strict,
		now:          clock.now,
		fetchIngress: func(string) (*api.Ingress, error) {
			return &api.Ingress{IngressID: uuid.New(), DomainName: host}, nil
		},
		pickHost:    func(*api.Ingress) string { return host },
		fetchStatus: func(string, string) (*api.DomainStatusResponse, error) { return status, nil },
		probe: func(base, path string) deploypkg.PublicURLSmokeResult {
			probes++
			return probe(base, path)
		},
	}
	return v, clock, &probes
}

func answer(code int) func(string, string) deploypkg.PublicURLSmokeResult {
	return func(_, path string) deploypkg.PublicURLSmokeResult {
		return deploypkg.PublicURLSmokeResult{Ready: code < 400, StatusCode: code, Path: path}
	}
}

// 1ctl#146/#147: the backend still says DNS is unresolved (a stale negative
// cache) while the URL already serves. After the grace the CLI's own request
// decides.
func TestPublicVerifierProbesAfterGraceWhenBackendDNSIsStale(t *testing.T) {
	v, clock, probes := testVerifier(false, domainStatus("abc123.satusky.com", true, api.DNSStatusPropagating), answer(200))
	if v.Verify() {
		t.Fatal("verified before the DNS grace; the user's resolver would be asked about an unpublished name")
	}
	if *probes != 0 {
		t.Fatalf("probes = %d before grace, want 0", *probes)
	}
	clock.t = clock.t.Add(publicURLDNSGrace)
	if !v.Verify() {
		t.Fatal("did not verify a serving URL after the DNS grace")
	}
	if !v.Reached || v.Host != "abc123.satusky.com" || v.Path != "/health" || v.StatusCode != 200 {
		t.Fatalf("verifier = %+v", v)
	}
}

func TestPublicVerifierProbesAtOnceWhenBackendDNSResolved(t *testing.T) {
	v, _, _ := testVerifier(true, domainStatus("abc123.satusky.com", true, api.DNSStatusResolved), answer(200))
	if !v.Verify() {
		t.Fatal("did not verify with DNS resolved")
	}
}

func TestPublicVerifierWaitsForRoute(t *testing.T) {
	v, clock, probes := testVerifier(false, domainStatus("abc123.satusky.com", false, api.DNSStatusPropagating), answer(200))
	clock.t = clock.t.Add(10 * publicURLDNSGrace)
	if v.Verify() || *probes != 0 {
		t.Fatalf("probed an unattached route (probes=%d)", *probes)
	}
}

func TestUnconfiguredVerifierAnswerRules(t *testing.T) {
	for _, tc := range []struct {
		code int
		want bool
	}{
		{200, true}, {302, true}, {401, true}, {403, true},
		// The edge returns these for a host it cannot route yet.
		{404, false}, {502, false}, {503, false},
	} {
		v, _, _ := testVerifier(false, domainStatus("abc123.satusky.com", true, api.DNSStatusResolved), answer(tc.code))
		if got := v.Verify(); got != tc.want {
			t.Fatalf("HTTP %d: verified = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestHealthPathVerifierRequiresSuccess(t *testing.T) {
	v, _, _ := testVerifier(true, domainStatus("abc123.satusky.com", true, api.DNSStatusResolved), answer(401))
	v.paths = []string{"/healthz"}
	if v.Verify() {
		t.Fatal("an explicit health path accepted HTTP 401")
	}
}

func TestAttachPublicVerifier(t *testing.T) {
	opts := api.DeploymentWaitOptions{Mode: api.DeploymentWaitModeApplication}
	if v := attachPublicVerifier(&opts, "dep-1", "/healthz"); v == nil || opts.VerifyApplication == nil || opts.VerifyUnconfiguredApplication != nil {
		t.Fatalf("health path: options = %+v", opts)
	}
	opts = api.DeploymentWaitOptions{Mode: api.DeploymentWaitModeApplication}
	if v := attachPublicVerifier(&opts, "dep-1", ""); v == nil || opts.VerifyApplication != nil || opts.VerifyUnconfiguredApplication == nil {
		t.Fatalf("no health path: options = %+v", opts)
	}
	opts = api.DeploymentWaitOptions{Mode: api.DeploymentWaitModeWorkload}
	if v := attachPublicVerifier(&opts, "dep-1", "/healthz"); v != nil || opts.VerifyApplication != nil {
		t.Fatal("workload mode must not verify the application")
	}
}
