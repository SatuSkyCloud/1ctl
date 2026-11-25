package deploy

import (
	"net/http"
	"time"

	"1ctl/internal/api"
	deploypkg "1ctl/internal/deploy"
)

// publicURLDNSGrace is how long the verifier trusts the backend's DNS verdict
// once the route is attached. Before this grace the CLI waits for the backend
// to report DNS resolved, so the user's own resolver is not asked about a name
// that is not published yet. A resolver caches "no such name" for the zone's
// SOA minimum (30 minutes for satusky.com), which would hide the URL from the
// user long after it works. After the grace the record has almost certainly
// been published (external-dns syncs every minute), and a backend verdict
// that is still unresolved is more likely a stale cache than a missing record
// (1ctl#147). From then on the CLI's own HTTPS request is the proof.
const publicURLDNSGrace = 60 * time.Second

// publicURLVerifier proves that a deployment's public URL answers. It is used
// in two ways:
//   - with an explicit health path (--health-path / [checks].health_path),
//     where only a 2xx/3xx answer counts;
//   - for an app with no readiness probe, where any answer from the app counts
//     (2xx/3xx, or 401/403 for an app behind auth). A 404 or 5xx does not
//     count, because the edge also returns those for a host it cannot route
//     yet.
type publicURLVerifier struct {
	deploymentID string
	paths        []string
	strict       bool
	now          func() time.Time

	fetchIngress func(string) (*api.Ingress, error)
	pickHost     func(*api.Ingress) string
	fetchStatus  func(ingressID, host string) (*api.DomainStatusResponse, error)
	probe        func(baseURL, path string) deploypkg.PublicURLSmokeResult

	unresolvedSince time.Time

	// Reached, Host, Path and StatusCode describe the answer that verified the
	// URL. They are set once Verify returns true.
	Reached    bool
	Host       string
	Path       string
	StatusCode int
}

func newPublicURLVerifier(deploymentID string, paths []string, strict bool) *publicURLVerifier {
	return &publicURLVerifier{
		deploymentID: deploymentID,
		paths:        paths,
		strict:       strict,
		now:          time.Now,
		fetchIngress: api.GetIngressByDeploymentID,
		pickHost:     verificationHost,
		fetchStatus: func(ingressID, host string) (*api.DomainStatusResponse, error) {
			return api.GetDomainStatusWithTimeout(ingressID, host, false, 5*time.Second)
		},
		probe: func(baseURL, path string) deploypkg.PublicURLSmokeResult {
			return deploypkg.CheckPublicURLSmokeAtPath(baseURL, path, true)
		},
	}
}

// newHealthPathVerifier requires a 2xx/3xx answer on the configured path.
func newHealthPathVerifier(deploymentID, healthPath string) *publicURLVerifier {
	return newPublicURLVerifier(deploymentID, deploypkg.SmokePathCandidates(healthPath), true)
}

// newUnconfiguredAppVerifier accepts any answer from the app on /health or /.
func newUnconfiguredAppVerifier(deploymentID string) *publicURLVerifier {
	return newPublicURLVerifier(deploymentID, deploypkg.SmokePathCandidates(""), false)
}

// Verify is the DeploymentWaitOptions callback. It is called once per poll.
func (v *publicURLVerifier) Verify() bool {
	ingress, err := v.fetchIngress(v.deploymentID)
	if err != nil || ingress == nil || ingress.DomainName == "" {
		return false
	}
	host := v.pickHost(ingress)
	if host == "" {
		return false
	}
	status, err := v.fetchStatus(ingress.IngressID.String(), host)
	if err != nil {
		status = nil
	}
	if !v.shouldProbe(status, host) {
		return false
	}
	for _, path := range v.paths {
		result := v.probe("https://"+host, path)
		if result.Ready || (!v.strict && isAuthChallenge(result.StatusCode)) {
			v.Reached, v.Host, v.Path, v.StatusCode = true, host, path, result.StatusCode
			return true
		}
	}
	return false
}

// shouldProbe decides whether the CLI may resolve and request the host now.
func (v *publicURLVerifier) shouldProbe(status *api.DomainStatusResponse, host string) bool {
	if deploypkg.PublicDomainReady(status, host) {
		return true
	}
	if status == nil || !status.Attached || !status.Route.Attached {
		v.unresolvedSince = time.Time{}
		return false
	}
	now := v.now()
	if v.unresolvedSince.IsZero() {
		v.unresolvedSince = now
	}
	return now.Sub(v.unresolvedSince) >= publicURLDNSGrace
}

func isAuthChallenge(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}
