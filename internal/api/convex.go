package api

import (
	"fmt"
	"net/http"
	"net/url"

	satuskyctx "1ctl/internal/context"
	"github.com/google/uuid"
)

const StorageEngineConvex StorageEngine = "convex"

// ConvexConfig deliberately excludes provider and runtime secrets. Managed
// Spaces credentials and the immutable package pin are owned by the backend.
type ConvexConfig struct {
	InstanceName     string `json:"instance_name"`
	DashboardEnabled bool   `json:"dashboard_enabled"`
}

type ConvexCreateOptions struct {
	Name             string
	StorageSize      string
	StorageClass     string
	CPURequest       string
	CPULimit         string
	MemoryRequest    string
	MemoryLimit      string
	DashboardEnabled bool
}

type ConvexStatus struct {
	Engine                     string `json:"engine"`
	Status                     string `json:"status"`
	ClusterExists              bool   `json:"cluster_exists"`
	DatabaseReady              bool   `json:"database_ready"`
	BackendReady               bool   `json:"backend_ready"`
	DashboardEnabled           bool   `json:"dashboard_enabled"`
	DashboardReady             bool   `json:"dashboard_ready"`
	PublicReachabilityVerified bool   `json:"public_reachability_verified"`
}

type ConvexCredentials struct {
	APIHost        string `json:"api_host"`
	APIPort        string `json:"api_port"`
	APIURL         string `json:"api_url"`
	SiteURL        string `json:"site_url"`
	DashboardURL   string `json:"dashboard_url,omitempty"`
	InstanceSecret string `json:"instance_secret,omitempty"`
}

func CreateConvex(opts ConvexCreateOptions) (*StorageConfig, error) {
	org, err := uuid.Parse(satuskyctx.GetCurrentOrgID())
	if err != nil {
		return nil, fmt.Errorf("select an organization before creating Convex: %w", err)
	}
	namespace, err := satuskyctx.GetCurrentNamespaceOrError()
	if err != nil {
		return nil, err
	}
	req := StorageConfig{
		ResourceID: uuid.New(), ResourceType: "standalone", OrganizationID: &org,
		Namespace: namespace, Engine: StorageEngineConvex, Version: "1.0.0", Replicas: 1,
		ClusterName: &opts.Name, DatabaseName: &opts.Name,
		StorageSize: opts.StorageSize, StorageClass: opts.StorageClass,
		CPURequest: opts.CPURequest, CPULimit: opts.CPULimit,
		MemoryRequest: opts.MemoryRequest, MemoryLimit: opts.MemoryLimit,
		Convex: &ConvexConfig{InstanceName: opts.Name, DashboardEnabled: opts.DashboardEnabled},
	}
	var resp struct {
		Data StorageConfig `json:"data"`
	}
	if err := makeRequest(http.MethodPost, "/databases/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func ListConvex() ([]StorageConfig, error) {
	namespace, err := satuskyctx.GetCurrentNamespaceOrError()
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []StorageConfig `json:"data"`
	}
	if err := makeRequest(http.MethodGet, "/databases/namespace/"+url.PathEscape(namespace), nil, &resp); err != nil {
		return nil, err
	}
	result := make([]StorageConfig, 0, len(resp.Data))
	for _, item := range resp.Data {
		if item.Engine == StorageEngineConvex {
			result = append(result, item)
		}
	}
	return result, nil
}

func GetConvex(id string) (*StorageConfig, error) {
	var resp struct {
		Data StorageConfig `json:"data"`
	}
	if err := makeRequest(http.MethodGet, "/databases/id/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data.Engine != StorageEngineConvex {
		return nil, fmt.Errorf("storage %s is not a Convex service", id)
	}
	return &resp.Data, nil
}

func GetConvexStatus(id string) (*ConvexStatus, error) {
	var resp struct {
		Data ConvexStatus `json:"data"`
	}
	if err := makeRequest(http.MethodGet, "/databases/"+url.PathEscape(id)+"/status", nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func GetConvexCredentials(id string) (*ConvexCredentials, error) {
	var resp struct {
		Data ConvexCredentials `json:"data"`
	}
	if err := makeRequest(http.MethodGet, "/databases/"+url.PathEscape(id)+"/credentials", nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func RedeployConvex(id string) error {
	return makeRequest(http.MethodPost, "/databases/"+url.PathEscape(id)+"/redeploy", nil, nil)
}

func DeleteConvex(id string) error {
	return makeRequest(http.MethodDelete, "/databases/"+url.PathEscape(id), nil, nil)
}
