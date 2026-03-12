package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/util/rand"
)

// ListDomains lists all domains for an organization
func ListDomains(userID, orgID string) ([]Domain, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/domains/list/%s/%s", userID, orgID), nil, &resp)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var domains []Domain
	if err := json.Unmarshal(data, &domains); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal domains: %s", err.Error()), nil)
	}
	return domains, nil
}

// GetDomain gets a single domain by ID
func GetDomain(userID, orgID, domainID string) (*Domain, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/domains/%s/%s/%s", userID, orgID, domainID), nil, &resp)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var domain Domain
	if err := json.Unmarshal(data, &domain); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal domain: %s", err.Error()), nil)
	}
	return &domain, nil
}

// CreateDomain creates a new domain
func CreateDomain(userID, orgID string, req DomainCreateRequest) (*Domain, error) {
	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/domains/create/%s/%s", userID, orgID), req, &resp)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var domain Domain
	if err := json.Unmarshal(data, &domain); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal domain: %s", err.Error()), nil)
	}
	return &domain, nil
}

// DeleteDomain deletes a domain
func DeleteDomain(userID, orgID, domainID string) error {
	return makeRequest("DELETE", fmt.Sprintf("/domains/delete/%s/%s/%s", userID, orgID, domainID), nil, nil)
}

// VerifyDomain verifies nameservers for a domain
func VerifyDomain(userID, orgID, domainID string) (*Domain, *NameserverStatus, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/domains/verify/%s/%s/%s", userID, orgID, domainID), nil, &resp)
	if err != nil {
		return nil, nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var result struct {
		Domain           Domain           `json:"domain"`
		NameserverStatus NameserverStatus `json:"nameserver_status"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, nil, utils.NewError(fmt.Sprintf("failed to unmarshal verify response: %s", err.Error()), nil)
	}
	return &result.Domain, &result.NameserverStatus, nil
}

// CheckDomainAvailability checks if domains are available for registration
func CheckDomainAvailability(userID, orgID string, req DomainCheckRequest) ([]DomainAvailabilityResult, error) {
	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/domains/check/%s/%s", userID, orgID), req, &resp)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var result struct {
		Results []DomainAvailabilityResult `json:"results"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal check response: %s", err.Error()), nil)
	}
	return result.Results, nil
}

// SearchDomains searches for domain availability across multiple TLDs
func SearchDomains(userID, orgID string, req DomainSearchRequest) ([]DomainSearchResult, error) {
	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/domains/search/%s/%s", userID, orgID), req, &resp)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var result struct {
		Results []DomainSearchResult `json:"results"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal search response: %s", err.Error()), nil)
	}
	return result.Results, nil
}

// InitiateDomainPurchase creates a Stripe Checkout session for a domain purchase
func InitiateDomainPurchase(userID, orgID string, req DomainPurchaseRequest) (*DomainPurchaseIntentResponse, error) {
	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/domains/purchase-intent/%s/%s", userID, orgID), req, &resp)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var intent DomainPurchaseIntentResponse
	if err := json.Unmarshal(data, &intent); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal purchase intent: %s", err.Error()), nil)
	}
	return &intent, nil
}

// GetPurchaseIntentStatus returns the current status of a domain purchase intent
func GetPurchaseIntentStatus(userID, orgID, intentID string) (*DomainPurchaseIntentStatus, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/domains/purchase-intent/%s/%s/%s", userID, orgID, intentID), nil, &resp)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var status DomainPurchaseIntentStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal intent status: %s", err.Error()), nil)
	}
	return &status, nil
}

// GetSavedContact returns the last used contact info for domain purchases in the org
func GetSavedContact(userID, orgID string) (*DomainContactInfo, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/domains/contact/%s/%s", userID, orgID), nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, nil
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var contact DomainContactInfo
	if err := json.Unmarshal(data, &contact); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal contact: %s", err.Error()), nil)
	}
	return &contact, nil
}

// ListDNSRecords lists all DNS records for a domain
func ListDNSRecords(userID, orgID, domainID string) ([]DNSRecord, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/domains/%s/%s/%s/records", userID, orgID, domainID), nil, &resp)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var records []DNSRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal DNS records: %s", err.Error()), nil)
	}
	return records, nil
}

// CreateDNSRecord creates a new DNS record for a domain
func CreateDNSRecord(userID, orgID, domainID string, req DNSRecordCreateRequest) (*DNSRecord, error) {
	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/domains/%s/%s/%s/records", userID, orgID, domainID), req, &resp)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var record DNSRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal DNS record: %s", err.Error()), nil)
	}
	return &record, nil
}

// UpdateDNSRecord updates an existing DNS record
func UpdateDNSRecord(userID, orgID, domainID, recordID string, req DNSRecordUpdateRequest) (*DNSRecord, error) {
	var resp apiResponse
	err := makeRequest("PUT", fmt.Sprintf("/domains/%s/%s/%s/records/%s", userID, orgID, domainID, recordID), req, &resp)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var record DNSRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal DNS record: %s", err.Error()), nil)
	}
	return &record, nil
}

// DeleteDNSRecord deletes a DNS record
func DeleteDNSRecord(userID, orgID, domainID, recordID string) error {
	return makeRequest("DELETE", fmt.Sprintf("/domains/%s/%s/%s/records/%s", userID, orgID, domainID, recordID), nil, nil)
}

const DOMAIN_SUFFIX = "satusky.com"

func GenerateDomainName(projectName string) (string, error) {
	// Clean project name: lowercase and replace invalid chars with hyphens
	cleanName := strings.ToLower(projectName)
	cleanName = strings.ReplaceAll(cleanName, "_", "-")
	cleanName = strings.ReplaceAll(cleanName, ".", "-")

	// First try without suffix
	proposedDomain := fmt.Sprintf("%s.%s", cleanName, DOMAIN_SUFFIX)

	// Keep trying until we find an available domain
	for {
		ingress, err := GetIngressByDomainName(proposedDomain)
		if err != nil {
			return "", utils.NewError(fmt.Sprintf("failed to check domain existence: %s", err.Error()), nil)
		}

		// If domain is available (not found), we can use it
		if ingress.IngressID == uuid.Nil {
			return proposedDomain, nil
		}

		// Domain exists, try with a new random suffix
		suffix := generateShortID()
		proposedDomain = fmt.Sprintf("%s-%s.%s", cleanName, suffix, DOMAIN_SUFFIX)
	}
}

func generateShortID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
