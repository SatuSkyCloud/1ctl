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

// WaitForIngressDNSStatus polls the backend until DNS is verified or the
// timeout expires.
func WaitForIngressDNSStatus(ingressID string, timeout time.Duration) (*DNSStatusResponse, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(dnsPollInterval)
	defer ticker.Stop()

	var last *DNSStatusResponse
	announced := false

	for {
		status, err := GetIngressDNSStatus(ingressID)
		if err == nil && status != nil {
			last = status
			if ingressDNSVerified(status) {
				return status, nil
			}
		}

		if time.Now().After(deadline) {
			if last == nil {
				return nil, utils.NewError(fmt.Sprintf("timeout waiting for DNS status for ingress %s", ingressID), err)
			}
			return last, utils.NewError(fmt.Sprintf("timeout waiting for DNS propagation for ingress %s", ingressID), err)
		}

		// Announced once, not per poll. The record is published by a background
		// sync, so this wait routinely spans many ticks and repeating the same
		// line for each one buries the result it is waiting for.
		if !announced {
			if last != nil && last.Domain != "" {
				utils.PrintInfo("Waiting for DNS to publish for %s (this can take a minute)...", last.Domain)
			} else {
				utils.PrintInfo("Waiting for DNS to publish (this can take a minute)...")
			}
			announced = true
		}
		<-ticker.C
	}
}

// ingressDNSVerified preserves compatibility with older backends that only
// return the legacy resolved status. When a typed condition is available, it
// is authoritative and only verified permits the wait to complete.
func ingressDNSVerified(status *DNSStatusResponse) bool {
	if status == nil {
		return false
	}
	if status.Condition != nil {
		return status.Condition.Status == DNSConditionStatusVerified
	}
	return status.Status == DNSStatusResolved
}
