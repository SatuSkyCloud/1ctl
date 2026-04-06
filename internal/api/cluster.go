package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
)

// ZoneOption represents an available deployment zone.
type ZoneOption struct {
	Value     string `json:"value"`
	Label     string `json:"label"`
	ClusterID string `json:"cluster_id"`
}

// ClusterInfo represents a cluster in the registry.
type ClusterInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Region      string `json:"region"`
	Zone        string `json:"zone"`
	Endpoint    string `json:"endpoint"`
	Priority    int    `json:"priority"`
	IsDefault   bool   `json:"is_default"`
	Enabled     bool   `json:"enabled"`
	Healthy     bool   `json:"healthy"`
}

// GetAvailableZones fetches available deployment zones from the backend.
// Backend wraps responses as {"error": false, "data": [...]}.
func GetAvailableZones() ([]ZoneOption, error) {
	var resp apiResponse
	if err := makeRequest("GET", "/clusters/zones", nil, &resp); err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal zones data: %s", err.Error()), nil)
	}

	var zones []ZoneOption
	if err := json.Unmarshal(data, &zones); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal zones: %s", err.Error()), nil)
	}
	return zones, nil
}

// GetClusters fetches all enabled clusters from the backend.
// Backend wraps responses as {"error": false, "data": [...]}.
func GetClusters() ([]ClusterInfo, error) {
	var resp apiResponse
	if err := makeRequest("GET", "/clusters", nil, &resp); err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal clusters data: %s", err.Error()), nil)
	}

	var clusters []ClusterInfo
	if err := json.Unmarshal(data, &clusters); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal clusters: %s", err.Error()), nil)
	}
	return clusters, nil
}
