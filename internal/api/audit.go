package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID             uuid.UUID              `json:"id"`
	OrganizationID uuid.UUID              `json:"organization_id"`
	UserID         uuid.UUID              `json:"user_id"`
	UserEmail      string                 `json:"user_email"`
	Action         string                 `json:"action"`
	ResourceType   string                 `json:"resource_type"`
	ResourceID     string                 `json:"resource_id"`
	Details        map[string]interface{} `json:"details,omitempty"`
	IPAddress      string                 `json:"ip_address"`
	UserAgent      string                 `json:"user_agent"`
	CreatedAt      time.Time              `json:"created_at"`
}

// GetAuditLogs gets audit logs for an organization
func GetAuditLogs(orgID string, limit int, action, userID string) ([]AuditLog, error) {
	path := fmt.Sprintf("/audit-logs/organizations/%s", orgID)
	params := []string{}
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if action != "" {
		params = append(params, fmt.Sprintf("action=%s", action))
	}
	if userID != "" {
		params = append(params, fmt.Sprintf("user_id=%s", userID))
	}
	if len(params) > 0 {
		path = path + "?"
		for i, p := range params {
			if i > 0 {
				path += "&"
			}
			path += p
		}
	}

	var resp apiResponse
	err := makeRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var logs []AuditLog
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal audit logs: %s", err.Error()), nil)
	}
	return logs, nil
}

// GetAuditLog gets a specific audit log
func GetAuditLog(orgID, logID string) (*AuditLog, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/audit-logs/organizations/%s/%s", orgID, logID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var log AuditLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal audit log: %s", err.Error()), nil)
	}
	return &log, nil
}

// ExportAuditLogs exports audit logs in specified format
func ExportAuditLogs(orgID, format string) ([]byte, error) {
	path := fmt.Sprintf("/audit-logs/organizations/%s/export", orgID)
	if format != "" {
		path = fmt.Sprintf("%s?format=%s", path, format)
	}

	var resp apiResponse
	err := makeRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	return data, nil
}
