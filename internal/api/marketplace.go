package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MarketplaceApp represents a marketplace application
type MarketplaceApp struct {
	MarketplaceID   uuid.UUID              `json:"marketplace_id"`
	MarketplaceName string                 `json:"marketplace_name"`
	Description     string                 `json:"description"`
	ImageURL        string                 `json:"image_url"`
	Category        string                 `json:"category"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	ComingSoon      bool                   `json:"coming_soon"`
	DeploymentCount int                    `json:"deployment_count,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// MarketplaceDeployRequest represents a request to deploy a marketplace app
type MarketplaceDeployRequest struct {
	DeploymentName     string              `json:"deployment_name"`
	Hostnames          []string            `json:"hostnames,omitempty"`
	CPUCores           string              `json:"cpu_cores,omitempty"`
	Memory             string              `json:"memory,omitempty"`
	DomainName         string              `json:"domain_name,omitempty"`
	StorageSize        string              `json:"storage_size,omitempty"`
	StorageClass       string              `json:"storage_class,omitempty"`
	MulticlusterConfig *MulticlusterConfig `json:"multicluster_config,omitempty"`
}

// MarketplaceDeployResponse represents a marketplace deployment response
type MarketplaceDeployResponse struct {
	DeploymentID uuid.UUID `json:"deployment_id"`
	AppLabel     string    `json:"app_label"`
	Domain       string    `json:"domain"`
	Status       string    `json:"status"`
}

// GetMarketplaceApps gets all marketplace apps
func GetMarketplaceApps(limit, offset int, sortBy string) ([]MarketplaceApp, error) {
	path := "/marketplace/all"
	params := []string{}
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if offset > 0 {
		params = append(params, fmt.Sprintf("offset=%d", offset))
	}
	if sortBy != "" {
		params = append(params, fmt.Sprintf("sort=%s", sortBy))
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

	var apps []MarketplaceApp
	if err := json.Unmarshal(data, &apps); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal marketplace apps: %s", err.Error()), nil)
	}
	return apps, nil
}

// GetMarketplaceApp gets a specific marketplace app
func GetMarketplaceApp(marketplaceID string) (*MarketplaceApp, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/marketplace/id/%s", marketplaceID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var app MarketplaceApp
	if err := json.Unmarshal(data, &app); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal marketplace app: %s", err.Error()), nil)
	}
	return &app, nil
}

// DeployMarketplaceApp deploys a marketplace app
func DeployMarketplaceApp(namespace, marketplaceID string, req MarketplaceDeployRequest) (*MarketplaceDeployResponse, error) {
	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/marketplace/deploy/create/%s/%s", namespace, marketplaceID), req, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var deployResp MarketplaceDeployResponse
	if err := json.Unmarshal(data, &deployResp); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal deploy response: %s", err.Error()), nil)
	}
	return &deployResp, nil
}
