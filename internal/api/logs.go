package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DeploymentLog represents a log entry for a deployment
type DeploymentLog struct {
	LogID        uuid.UUID `json:"log_id"`
	DeploymentID uuid.UUID `json:"deployment_id"`
	Timestamp    time.Time `json:"timestamp"`
	Message      string    `json:"message"`
	PodName      string    `json:"pod_name"`
	Container    string    `json:"container"`
	Level        string    `json:"level"`
}

// LogStats represents log statistics for a deployment
type LogStats struct {
	DeploymentID uuid.UUID `json:"deployment_id"`
	TotalLines   int       `json:"total_lines"`
	TotalSize    int64     `json:"total_size"`
	OldestLog    time.Time `json:"oldest_log"`
	NewestLog    time.Time `json:"newest_log"`
}

// PodInfo represents pod information
type PodInfo struct {
	PodName   string    `json:"name"`
	Namespace string    `json:"namespace"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UID       string    `json:"uid"`
}

// GetStoredLogs retrieves stored logs for a deployment
func GetStoredLogs(deploymentID string, tail int) ([]DeploymentLog, error) {
	path := fmt.Sprintf("/pods/logs/%s", deploymentID)
	if tail > 0 {
		path = fmt.Sprintf("%s?tail=%d", path, tail)
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

	var logs []DeploymentLog
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal logs: %s", err.Error()), nil)
	}
	return logs, nil
}

// GetLogStats retrieves log statistics for a deployment
func GetLogStats(deploymentID string) (*LogStats, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/pods/logs/stats/%s", deploymentID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var stats LogStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal log stats: %s", err.Error()), nil)
	}
	return &stats, nil
}

// DeleteLogs deletes logs for a deployment
func DeleteLogs(deploymentID string) error {
	return makeRequest("DELETE", fmt.Sprintf("/pods/logs/%s", deploymentID), nil, nil)
}

// GetPodByLabel gets pod information by deployment ID
func GetPodByLabel(namespace, deploymentID string) (*PodInfo, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/pods/%s/%s", namespace, deploymentID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var pod PodInfo
	if err := json.Unmarshal(data, &pod); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal pod info: %s", err.Error()), nil)
	}
	return &pod, nil
}
