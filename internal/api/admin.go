package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AdminMachineUsage represents machine usage record for admin operations
type AdminMachineUsage struct {
	ID             uuid.UUID `json:"id"`
	MachineID      uuid.UUID `json:"machine_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time,omitempty"`
	Hours          float64   `json:"hours"`
	Cost           float64   `json:"cost"`
	Billed         bool      `json:"billed"`
	BilledAt       time.Time `json:"billed_at,omitempty"`
}

// AdminNamespace represents a namespace for admin
type AdminNamespace struct {
	Name      string            `json:"name"`
	Phase     string            `json:"phase"`
	CreatedAt time.Time         `json:"created_at"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// AdminClusterRole represents a cluster role
type AdminClusterRole struct {
	Name      string    `json:"name"`
	Rules     []string  `json:"rules,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminCreditsRequest represents request to add/refund credits
type AdminCreditsRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// CleanupRequest represents request to cleanup resources
type CleanupRequest struct {
	Label     string `json:"label"`
	Namespace string `json:"namespace"`
}

// GetUnbilledMachineUsages gets all unbilled active machine usages
func GetUnbilledMachineUsages() ([]AdminMachineUsage, error) {
	var resp apiResponse
	err := makeRequest("GET", "/admin/machine-usage/unbilled", nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var usages []AdminMachineUsage
	if err := json.Unmarshal(data, &usages); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal machine usages: %s", err.Error()), nil)
	}
	return usages, nil
}

// GetMachineUsagesByMachineID gets machine usages for a specific machine
func GetMachineUsagesByMachineID(machineID string) ([]AdminMachineUsage, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/admin/machine-usage/%s", machineID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var usages []AdminMachineUsage
	if err := json.Unmarshal(data, &usages); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal machine usages: %s", err.Error()), nil)
	}
	return usages, nil
}

// MarkMachineUsageAsBilled marks a machine usage as billed
func MarkMachineUsageAsBilled(usageID string) error {
	return makeRequest("POST", fmt.Sprintf("/admin/machine-usage/%s/bill", usageID), nil, nil)
}

// AdminAddCredits adds credits to an organization
func AdminAddCredits(orgID string, amount float64, description string) error {
	req := AdminCreditsRequest{
		Amount:      amount,
		Description: description,
	}
	return makeRequest("POST", fmt.Sprintf("/admin/credits/organizations/%s/add", orgID), req, nil)
}

// AdminRefundCredits refunds credits to an organization
func AdminRefundCredits(orgID string, amount float64, description string) error {
	req := AdminCreditsRequest{
		Amount:      amount,
		Description: description,
	}
	return makeRequest("POST", fmt.Sprintf("/admin/credits/organizations/%s/refund", orgID), req, nil)
}

// GetAdminNamespaces gets all namespaces (admin only)
func GetAdminNamespaces() ([]AdminNamespace, error) {
	var resp apiResponse
	err := makeRequest("GET", "/admin/namespaces", nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var namespaces []AdminNamespace
	if err := json.Unmarshal(data, &namespaces); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal namespaces: %s", err.Error()), nil)
	}
	return namespaces, nil
}

// GetAdminClusterRoles gets all cluster roles (admin only)
func GetAdminClusterRoles() ([]AdminClusterRole, error) {
	var resp apiResponse
	err := makeRequest("GET", "/admin/cluster-roles", nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var roles []AdminClusterRole
	if err := json.Unmarshal(data, &roles); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal cluster roles: %s", err.Error()), nil)
	}
	return roles, nil
}

// AdminCleanupResources cleans up resources by label (admin only)
func AdminCleanupResources(label, namespace string) error {
	req := CleanupRequest{
		Label:     label,
		Namespace: namespace,
	}
	return makeRequest("DELETE", "/admin/resources/cleanup", req, nil)
}
