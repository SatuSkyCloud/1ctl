package api

import (
	"1ctl/internal/utils"
	stdctx "context"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	dnsPollInterval   = 3 * time.Second
	dnsLookupDeadline = 5 * time.Second
)

var publicDNSResolvers = []string{
	"1.1.1.1:53",
	"8.8.8.8:53",
}

// WaitForPublicDNSResolution polls a couple of public recursive resolvers
// until the given domain resolves or the timeout is reached. It is used to
// keep deploy success output honest for platform-managed hostnames that can
// take a short time to appear in public DNS after the ingress is created.
func WaitForPublicDNSResolution(domain string, timeout time.Duration) ([]string, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return nil, utils.NewError("domain is required", nil)
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(dnsPollInterval)
	defer ticker.Stop()

	var lastErr error
	var lastIPs []string

	for {
		ips, err := resolvePublicDNS(domain)
		if err == nil && len(ips) > 0 {
			return ips, nil
		}
		if err != nil {
			lastErr = err
		}
		if len(ips) > 0 {
			lastIPs = ips
		}

		if time.Now().After(deadline) {
			if lastErr == nil {
				lastErr = fmt.Errorf("DNS lookup timed out")
			}
			return lastIPs, utils.NewError(fmt.Sprintf("timeout waiting for DNS propagation for %s", domain), lastErr)
		}

		utils.PrintInfo("Waiting for DNS propagation for %s...", domain)
		<-ticker.C
	}
}

func resolvePublicDNS(domain string) ([]string, error) {
	var lastErr error
	for _, server := range publicDNSResolvers {
		ips, err := lookupHostViaResolver(domain, server)
		if err == nil && len(ips) > 0 {
			return ips, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no public resolver returned an answer")
	}
	return nil, lastErr
}

func lookupHostViaResolver(domain, server string) ([]string, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx stdctx.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			return d.DialContext(ctx, "udp", server)
		},
	}

	lookupCtx, cancel := stdctx.WithTimeout(stdctx.Background(), dnsLookupDeadline)
	defer cancel()

	return resolver.LookupHost(lookupCtx, domain)
}
