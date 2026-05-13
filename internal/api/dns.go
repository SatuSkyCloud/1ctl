package api

import (
	"1ctl/internal/utils"
	"fmt"
	"time"
)

const dnsPollInterval = 3 * time.Second

// GetIngressDNSStatus asks the backend control plane for the current DNS
// propagation status of a specific ingress. The backend owns the authoritative
// view, so the CLI does not guess from the workstation's resolver.
func GetIngressDNSStatus(ingressID string) (*DNSStatusResponse, error) {
	var resp struct {
		Error bool              `json:"error"`
		Data  DNSStatusResponse `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/ingresses/%s/dns-status", ingressID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// WaitForIngressDNSStatus polls the backend until the ingress DNS status is
// resolved or the timeout expires.
func WaitForIngressDNSStatus(ingressID string, timeout time.Duration) (*DNSStatusResponse, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(dnsPollInterval)
	defer ticker.Stop()

	var last *DNSStatusResponse

	for {
		status, err := GetIngressDNSStatus(ingressID)
		if err == nil && status != nil {
			last = status
			if status.Status == DNSStatusResolved {
				return status, nil
			}
		}

		if time.Now().After(deadline) {
			if last == nil {
				return nil, utils.NewError(fmt.Sprintf("timeout waiting for DNS status for ingress %s", ingressID), err)
			}
			return last, utils.NewError(fmt.Sprintf("timeout waiting for DNS propagation for ingress %s", ingressID), err)
		}

		if last != nil && last.Domain != "" {
			utils.PrintInfo("Waiting for DNS propagation for %s...", last.Domain)
		} else {
			utils.PrintInfo("Waiting for DNS propagation for ingress %s...", ingressID)
		}
		<-ticker.C
	}
}
