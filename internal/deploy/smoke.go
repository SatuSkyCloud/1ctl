package deploy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// PublicURLSmokeResult holds the outcome of a single smoke-test probe.
type PublicURLSmokeResult struct {
	Ready      bool
	StatusCode int
	Reason     string
	Path       string
}

// SmokePathCandidates returns the URL paths to probe for a deployment smoke
// test. When an explicit path is given (from --health-path or satusky.toml
// [checks].health_path), it is the only candidate. Otherwise the fallback
// order is /health then /.
func SmokePathCandidates(explicitPath string) []string {
	if explicitPath != "" {
		return []string{explicitPath}
	}
	return []string{"/health", "/"}
}

// CheckPublicURLSmoke probes a base URL (https://domain) against one or more
// HTTP paths and returns the first successful result. In non-strict mode
// (strict=false), 401/403/404 are accepted as proof of platform reachability.
func CheckPublicURLSmoke(baseURL string, paths []string, strict bool) PublicURLSmokeResult {
	if len(paths) == 0 {
		paths = SmokePathCandidates("")
	}

	var failureReasons []string
	var lastFailure PublicURLSmokeResult
	for _, path := range paths {
		result := CheckPublicURLSmokeAtPath(baseURL, path, strict)
		if result.Ready {
			return result
		}
		failureReasons = append(failureReasons, fmt.Sprintf("%s: %s", path, result.Reason))
		lastFailure = result
	}

	if len(failureReasons) == 1 {
		return lastFailure
	}
	return PublicURLSmokeResult{
		Ready:      false,
		Reason:     strings.Join(failureReasons, "; "),
		Path:       lastFailure.Path,
		StatusCode: lastFailure.StatusCode,
	}
}

// CheckPublicURLSmokeAtPath probes a single URL path and returns the result.
func CheckPublicURLSmokeAtPath(baseURL, path string, strict bool) PublicURLSmokeResult {
	targetURL := strings.TrimRight(baseURL, "/") + path
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Resolver: &net.Resolver{
					PreferGo: true,
					Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
						d := net.Dialer{Timeout: 3 * time.Second}
						return d.DialContext(ctx, "udp", "8.8.8.8:53")
					},
				},
			}).DialContext,
		},
	}
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return PublicURLSmokeResult{Ready: false, Reason: fmt.Sprintf("failed to build request: %s", err.Error()), Path: path}
	}

	resp, err := client.Do(req)
	if err != nil {
		return PublicURLSmokeResult{Ready: false, Reason: fmt.Sprintf("request failed: %s", err.Error()), Path: path}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		if !strict && isReachabilityStatus(resp.StatusCode) {
			return PublicURLSmokeResult{
				Ready:      true,
				StatusCode: resp.StatusCode,
				Path:       path,
			}
		}
		return PublicURLSmokeResult{
			Ready:      false,
			StatusCode: resp.StatusCode,
			Reason:     fmt.Sprintf("unexpected HTTP status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
			Path:       path,
		}
	}

	return PublicURLSmokeResult{
		Ready:      true,
		StatusCode: resp.StatusCode,
		Path:       path,
	}
}

// IsReachabilityStatus returns true for HTTP status codes that still prove
// DNS/TLS/routing worked (platform reachable).
func IsReachabilityStatus(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden || code == http.StatusNotFound
}

func isReachabilityStatus(code int) bool {
	return IsReachabilityStatus(code)
}

// OpenBrowser opens the given URL in the user's default browser.
func OpenBrowser(url string) error {
	if url == "" {
		return fmt.Errorf("no URL to open")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}
