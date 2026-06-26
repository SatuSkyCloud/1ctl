package api

import (
	"fmt"
	"net/url"
)

func CheckDomainAvailability(userID, orgID string, req DomainCheckRequest) ([]DomainAvailabilityResult, error) {
	var resp struct {
		Error bool `json:"error"`
		Data  struct {
			Results []DomainAvailabilityResult `json:"results"`
		} `json:"data"`
	}
	if err := makeRequest("POST", fmt.Sprintf("/domains/check/%s/%s", url.PathEscape(userID), url.PathEscape(orgID)), req, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Results, nil
}

func SearchDomains(userID, orgID string, req DomainSearchRequest) ([]DomainSearchResult, error) {
	var resp struct {
		Error bool `json:"error"`
		Data  struct {
			Results []DomainSearchResult `json:"results"`
		} `json:"data"`
	}
	if err := makeRequest("POST", fmt.Sprintf("/domains/search/%s/%s", url.PathEscape(userID), url.PathEscape(orgID)), req, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Results, nil
}

func PurchaseDomain(userID, orgID string, req DomainPurchaseRequest) (*DomainPurchaseIntentResponse, error) {
	var resp struct {
		Error bool                         `json:"error"`
		Data  DomainPurchaseIntentResponse `json:"data"`
	}
	if err := makeRequest("POST", fmt.Sprintf("/domains/purchase-intent/%s/%s", url.PathEscape(userID), url.PathEscape(orgID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func GetDomainPurchaseStatus(userID, orgID, intentID string) (*DomainPurchaseIntentStatus, error) {
	var resp struct {
		Error bool                       `json:"error"`
		Data  DomainPurchaseIntentStatus `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/domains/purchase-intent/%s/%s/%s", url.PathEscape(userID), url.PathEscape(orgID), url.PathEscape(intentID)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func ListManagedDomains(userID, orgID string) ([]Domain, error) {
	var resp struct {
		Error bool     `json:"error"`
		Data  []Domain `json:"data"`
		Count int      `json:"count"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/domains/list/%s/%s", url.PathEscape(userID), url.PathEscape(orgID)), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func CreateManagedDomain(userID, orgID string, req DomainCreateRequest) (*Domain, *NameserverStatus, error) {
	var resp struct {
		Error            bool             `json:"error"`
		Data             Domain           `json:"data"`
		NameserverStatus NameserverStatus `json:"nameserver_status"`
	}
	if err := makeRequest("POST", fmt.Sprintf("/domains/create/%s/%s", url.PathEscape(userID), url.PathEscape(orgID)), req, &resp); err != nil {
		return nil, nil, err
	}
	return &resp.Data, &resp.NameserverStatus, nil
}

func DeleteManagedDomain(userID, orgID, domainID string) error {
	var resp apiResponse
	return makeRequest("DELETE", fmt.Sprintf("/domains/delete/%s/%s/%s", url.PathEscape(userID), url.PathEscape(orgID), url.PathEscape(domainID)), nil, &resp)
}

func VerifyManagedDomain(userID, orgID, domainID string) (*Domain, *NameserverStatus, error) {
	var resp struct {
		Error bool `json:"error"`
		Data  struct {
			Domain           Domain           `json:"domain"`
			NameserverStatus NameserverStatus `json:"nameserver_status"`
		} `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/domains/verify/%s/%s/%s", url.PathEscape(userID), url.PathEscape(orgID), url.PathEscape(domainID)), nil, &resp); err != nil {
		return nil, nil, err
	}
	return &resp.Data.Domain, &resp.Data.NameserverStatus, nil
}

func ListDNSRecords(userID, orgID, domainID string) ([]DNSRecord, error) {
	var resp struct {
		Error bool        `json:"error"`
		Data  []DNSRecord `json:"data"`
		Count int         `json:"count"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/domains/%s/%s/%s/records", url.PathEscape(userID), url.PathEscape(orgID), url.PathEscape(domainID)), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func CreateDNSRecord(userID, orgID, domainID string, req DNSRecordCreateRequest) (*DNSRecord, error) {
	var resp struct {
		Error bool      `json:"error"`
		Data  DNSRecord `json:"data"`
	}
	if err := makeRequest("POST", fmt.Sprintf("/domains/%s/%s/%s/records", url.PathEscape(userID), url.PathEscape(orgID), url.PathEscape(domainID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func UpdateDNSRecord(userID, orgID, domainID, recordID string, req DNSRecordUpdateRequest) (*DNSRecord, error) {
	var resp struct {
		Error bool      `json:"error"`
		Data  DNSRecord `json:"data"`
	}
	if err := makeRequest("PUT", fmt.Sprintf("/domains/%s/%s/%s/records/%s", url.PathEscape(userID), url.PathEscape(orgID), url.PathEscape(domainID), url.PathEscape(recordID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func DeleteDNSRecord(userID, orgID, domainID, recordID string) error {
	var resp apiResponse
	return makeRequest("DELETE", fmt.Sprintf("/domains/%s/%s/%s/records/%s", url.PathEscape(userID), url.PathEscape(orgID), url.PathEscape(domainID), url.PathEscape(recordID)), nil, &resp)
}
