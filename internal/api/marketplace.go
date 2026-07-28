package api

import (
	cliContext "1ctl/internal/context"
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MarketplaceApp represents a marketplace application
type MarketplaceApp struct {
	MarketplaceID     uuid.UUID                  `json:"marketplace_id"`
	MarketplaceName   string                     `json:"marketplace_name"`
	Description       string                     `json:"description"`
	ImageURL          string                     `json:"image_url"`
	Category          string                     `json:"category"`
	Metadata          map[string]interface{}     `json:"metadata,omitempty"`
	SupportedArchs    []string                   `json:"supported_archs,omitempty"`
	PackageReleaseID  *uuid.UUID                 `json:"package_release_id,omitempty"`
	PackageRelease    *MarketplacePackageRelease `json:"package_release,omitempty"`
	Deployable        bool                       `json:"deployable"`
	DeployabilityCode string                     `json:"deployability_code,omitempty"`
	ComingSoon        bool                       `json:"coming_soon"`
	DeploymentCount   int                        `json:"deployment_count,omitempty"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
}

// MarketplacePackageRelease is the immutable package release pinned to a catalog app.
type MarketplacePackageRelease struct {
	ReleaseID             uuid.UUID                            `json:"release_id"`
	PackageName           string                               `json:"package_name"`
	PackageVersion        string                               `json:"package_version"`
	ContentDigest         string                               `json:"content_digest"`
	ManifestMetadata      map[string]interface{}               `json:"manifest_metadata,omitempty"`
	CertificationMetadata map[string]interface{}               `json:"certification_metadata,omitempty"`
	Trusted               bool                                 `json:"trusted"`
	CertifiedAt           time.Time                            `json:"certified_at"`
	RegisteredAt          time.Time                            `json:"registered_at"`
	Governance            *MarketplacePackageReleaseGovernance `json:"governance,omitempty"`
}

type MarketplacePackageReleaseGovernance struct {
	State      string    `json:"state"`
	Actor      string    `json:"actor"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

// MarketplaceDeployRequest represents a request to deploy a marketplace app
type MarketplaceDeployRequest struct {
	DeploymentName string                 `json:"deployment_name"`
	Hostnames      []string               `json:"hostnames,omitempty"`
	Replicas       int32                  `json:"replicas,omitempty"`
	CPURequest     string                 `json:"cpu_request,omitempty"`
	MemoryRequest  string                 `json:"memory_request,omitempty"`
	StorageSize    string                 `json:"storage_size,omitempty"`
	Values         map[string]interface{} `json:"values,omitempty"`
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
	path := "/marketplaces/all"
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}
	if sortBy != "" {
		params.Set("sortBy", sortBy)
	}
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp apiResponse
	err := makeMainAPIRequestWithHeaders(http.MethodGet, path, nil, &resp, marketplaceOrganizationHeaders())
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
	err := makeMainAPIRequestWithHeaders(http.MethodGet, fmt.Sprintf("/marketplaces/id/%s", url.PathEscape(marketplaceID)), nil, &resp, marketplaceOrganizationHeaders())
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

// ResolveMarketplaceApp resolves a marketplace app name or UUID to a full app record.
// If nameOrID looks like a UUID, it tries direct lookup first; otherwise it fetches
// the catalog and matches by marketplace_name (case-insensitive).
func ResolveMarketplaceApp(nameOrID string) (*MarketplaceApp, error) {
	// Try direct UUID lookup first.
	if _, err := uuid.Parse(nameOrID); err == nil {
		app, err := GetMarketplaceApp(nameOrID)
		if err == nil {
			return app, nil
		}
	}

	// Fall back to name-based lookup in the catalog.
	// Use a high limit to guarantee all apps are returned (backend defaults to 9).
	apps, err := GetMarketplaceApps(100, 0, "name")
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to list marketplace apps: %s", err.Error()), nil)
	}
	nameOrIDLower := strings.ToLower(nameOrID)
	for _, a := range apps {
		if strings.ToLower(a.MarketplaceName) == nameOrIDLower {
			return &a, nil
		}
	}
	return nil, utils.NewError(fmt.Sprintf("marketplace app %q not found — use '1ctl marketplace list' to see available apps", nameOrID), nil)
}

// DeployMarketplaceApp deploys a marketplace app
func DeployMarketplaceApp(namespace, marketplaceID string, req MarketplaceDeployRequest) (*MarketplaceDeployResponse, error) {
	var resp apiResponse
	err := makeMainAPIRequestWithHeaders(http.MethodPost, fmt.Sprintf("/marketplaces/deploy/create/%s/%s", url.PathEscape(namespace), url.PathEscape(marketplaceID)), req, &resp, marketplaceOrganizationHeaders())
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

// DownloadMarketplaceDeploymentOutput downloads one package-declared output.
// Outputs may be sensitive, so callers must not include the returned bytes in
// ordinary structured output.
func DownloadMarketplaceDeploymentOutput(deploymentID, outputName string) ([]byte, error) {
	path := fmt.Sprintf(
		"/marketplace-outputs/deployments/%s/outputs/%s/download",
		url.PathEscape(deploymentID),
		url.PathEscape(outputName),
	)
	return downloadMainAPIResponse(path, marketplaceOrganizationHeaders())
}

// marketplaceOrganizationHeaders carries the user's selected organization for
// owner-private marketplace visibility. The backend validates membership; an
// empty context retains the public catalog behavior.
func marketplaceOrganizationHeaders() http.Header {
	headers := make(http.Header)
	if organizationID := strings.TrimSpace(cliContext.GetCurrentOrgID()); organizationID != "" {
		headers.Set("x-satusky-organization-id", organizationID)
	}
	return headers
}
