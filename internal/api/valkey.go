package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	satuskyctx "1ctl/internal/context"

	"github.com/google/uuid"
)

const (
	ValkeyVersion      = "9.1.1"
	ValkeyChartVersion = "0.11.0"
)

type ValkeyCreateOptions struct {
	Name             string
	MachineID        string
	Topology         string
	Instances        int
	Persistence      bool
	StorageSize      string
	StorageClass     string
	CPURequest       string
	CPULimit         string
	MemoryRequest    string
	MemoryLimit      string
	AppendOnly       bool
	AppendFsync      string
	MaxmemoryPolicy  string
	MaxmemoryPercent int
	MetricsEnabled   bool
}

type ValkeyStatus struct {
	Status             string `json:"status"`
	WorkloadKind       string `json:"workload_kind"`
	ReadyInstances     int    `json:"ready_instances"`
	DesiredInstances   int    `json:"desired_instances"`
	Primary            string `json:"primary"`
	PrivateHost        string `json:"private_host"`
	ReadOnlyHost       string `json:"read_only_host,omitempty"`
	MetricsEnabled     bool   `json:"metrics_enabled"`
	PersistenceEnabled bool   `json:"persistence_enabled"`
}

type ValkeyCredentials struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	Host         string `json:"host"`
	Port         string `json:"port"`
	URI          string `json:"uri"`
	ReadOnlyHost string `json:"read_only_host,omitempty"`
	ReadOnlyURI  string `json:"read_only_uri,omitempty"`
	TLS          bool   `json:"tls"`
}

type ValkeyCreateUserRequest struct {
	Username        string   `json:"username"`
	AccessPreset    string   `json:"access_preset"`
	KeyPatterns     []string `json:"key_patterns,omitempty"`
	ChannelPatterns []string `json:"channel_patterns,omitempty"`
}

type ValkeyUpdateUserRequest struct {
	AccessPreset    *string   `json:"access_preset,omitempty"`
	KeyPatterns     *[]string `json:"key_patterns,omitempty"`
	ChannelPatterns *[]string `json:"channel_patterns,omitempty"`
}

type ValkeyCreatedUser struct {
	User           ValkeyUserConfig `json:"user"`
	Password       string           `json:"password"`
	Notice         string           `json:"notice,omitempty"`
	RestartPending bool             `json:"restart_pending,omitempty"`
}

type ValkeyRotatedCredential struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	Notice         string `json:"notice,omitempty"`
	RestartPending bool   `json:"restart_pending,omitempty"`
}

type ValkeyLogs struct {
	Lines []string `json:"lines"`
	Pod   string   `json:"pod"`
}

func CreateValkey(opts ValkeyCreateOptions) (*StorageConfig, error) {
	orgIDString := satuskyctx.GetCurrentOrgID()
	if orgIDString == "" {
		return nil, fmt.Errorf("organization ID not found. Please run '1ctl auth login' first")
	}
	orgID, err := uuid.Parse(orgIDString)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID in profile: %w", err)
	}
	namespace, err := satuskyctx.GetCurrentNamespaceOrError()
	if err != nil {
		return nil, err
	}

	port := "6379"
	name := opts.Name
	instances := opts.Instances
	appendOnly := opts.AppendOnly
	metricsEnabled := opts.MetricsEnabled
	req := StorageConfig{
		ResourceID:         uuid.New(),
		ResourceType:       "standalone",
		Namespace:          namespace,
		OrganizationID:     &orgID,
		Engine:             StorageEngineValkey,
		Version:            ValkeyVersion,
		Replicas:           opts.Instances,
		ClusterName:        &name,
		Port:               &port,
		StorageSize:        opts.StorageSize,
		StorageClass:       opts.StorageClass,
		CPURequest:         opts.CPURequest,
		CPULimit:           opts.CPULimit,
		MemoryRequest:      opts.MemoryRequest,
		MemoryLimit:        opts.MemoryLimit,
		Instances:          &instances,
		PersistenceEnabled: &opts.Persistence,
		Valkey: &ValkeyConfig{
			Topology:         opts.Topology,
			MachineID:        opts.MachineID,
			AppendOnly:       &appendOnly,
			AppendFsync:      opts.AppendFsync,
			MaxmemoryPolicy:  opts.MaxmemoryPolicy,
			MaxmemoryPercent: opts.MaxmemoryPercent,
			MetricsEnabled:   &metricsEnabled,
			ChartVersion:     ValkeyChartVersion,
			ImageVersion:     ValkeyVersion,
		},
	}

	var resp struct {
		Error bool          `json:"error"`
		Data  StorageConfig `json:"data"`
	}
	if err := makeRequest(http.MethodPost, "/databases/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func ListValkey(namespace string) ([]StorageConfig, error) {
	if namespace == "" {
		var err error
		namespace, err = satuskyctx.GetCurrentNamespaceOrError()
		if err != nil {
			return nil, err
		}
	}

	var resp struct {
		Error bool            `json:"error"`
		Data  []StorageConfig `json:"data"`
	}
	if err := makeRequest(http.MethodGet, fmt.Sprintf("/databases/namespace/%s", url.PathEscape(namespace)), nil, &resp); err != nil {
		return nil, err
	}
	instances := make([]StorageConfig, 0, len(resp.Data))
	for _, storage := range resp.Data {
		if storage.Engine == StorageEngineValkey {
			instances = append(instances, storage)
		}
	}
	return instances, nil
}

func GetValkey(storageID string) (*StorageConfig, error) {
	var resp struct {
		Error bool          `json:"error"`
		Data  StorageConfig `json:"data"`
	}
	if err := makeRequest(http.MethodGet, fmt.Sprintf("/databases/id/%s", url.PathEscape(storageID)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func GetValkeyStatus(storageID string) (*ValkeyStatus, error) {
	var resp struct {
		Error bool         `json:"error"`
		Data  ValkeyStatus `json:"data"`
	}
	if err := makeRequest(http.MethodGet, fmt.Sprintf("/databases/%s/status", url.PathEscape(storageID)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func GetValkeyCredentials(storageID string) (*ValkeyCredentials, error) {
	var resp struct {
		Error bool              `json:"error"`
		Data  ValkeyCredentials `json:"data"`
	}
	if err := makeRequest(http.MethodGet, fmt.Sprintf("/databases/%s/credentials", url.PathEscape(storageID)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func RedeployValkey(storageID string) error {
	return makeRequest(http.MethodPost, fmt.Sprintf("/databases/%s/redeploy", url.PathEscape(storageID)), nil, nil)
}

func RestartValkey(storageID string) error {
	return makeRequest(http.MethodPost, fmt.Sprintf("/databases/%s/restart", url.PathEscape(storageID)), nil, nil)
}

func DeleteValkey(storageID string) error {
	return makeRequest(http.MethodDelete, fmt.Sprintf("/databases/%s", url.PathEscape(storageID)), nil, nil)
}

func UpdateValkey(storageID string, config *StorageConfig) (*StorageConfig, error) {
	var resp struct {
		Error bool          `json:"error"`
		Data  StorageConfig `json:"data"`
	}
	if err := makeRequest(http.MethodPut, fmt.Sprintf("/databases/update/%s", url.PathEscape(storageID)), config, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func ListValkeyUsers(storageID string) ([]ValkeyUserConfig, error) {
	var resp struct {
		Error bool               `json:"error"`
		Data  []ValkeyUserConfig `json:"data"`
	}
	path := fmt.Sprintf("/databases/%s/valkey/users", url.PathEscape(storageID))
	if err := makeRequest(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func CreateValkeyUser(storageID string, request ValkeyCreateUserRequest) (*ValkeyCreatedUser, error) {
	var resp struct {
		Error bool              `json:"error"`
		Data  ValkeyCreatedUser `json:"data"`
	}
	path := fmt.Sprintf("/databases/%s/valkey/users", url.PathEscape(storageID))
	status, err := makeRequestWithStatus(http.MethodPost, path, request, &resp)
	if err != nil {
		return nil, err
	}
	resp.Data.RestartPending = status == http.StatusAccepted
	return &resp.Data, nil
}

func UpdateValkeyUser(storageID, username string, request ValkeyUpdateUserRequest) (*ValkeyUserConfig, error) {
	var resp struct {
		Error bool             `json:"error"`
		Data  ValkeyUserConfig `json:"data"`
	}
	path := fmt.Sprintf("/databases/%s/valkey/users/%s", url.PathEscape(storageID), url.PathEscape(username))
	if err := makeRequest(http.MethodPatch, path, request, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func DeleteValkeyUser(storageID, username string) error {
	path := fmt.Sprintf("/databases/%s/valkey/users/%s", url.PathEscape(storageID), url.PathEscape(username))
	return makeRequest(http.MethodDelete, path, nil, nil)
}

func RotateValkeyUserPassword(storageID, username string) (*ValkeyRotatedCredential, error) {
	var resp struct {
		Error bool                    `json:"error"`
		Data  ValkeyRotatedCredential `json:"data"`
	}
	path := fmt.Sprintf("/databases/%s/valkey/users/%s/rotate-password", url.PathEscape(storageID), url.PathEscape(username))
	status, err := makeRequestWithStatus(http.MethodPost, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	resp.Data.RestartPending = status == http.StatusAccepted
	return &resp.Data, nil
}

func RotateValkeyCredentials(storageID string) (*ValkeyRotatedCredential, error) {
	var resp struct {
		Error bool                    `json:"error"`
		Data  ValkeyRotatedCredential `json:"data"`
	}
	path := fmt.Sprintf("/databases/%s/credentials/rotate", url.PathEscape(storageID))
	status, err := makeRequestWithStatus(http.MethodPost, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	resp.Data.RestartPending = status == http.StatusAccepted
	return &resp.Data, nil
}

func GetValkeyMetrics(storageID string) (map[string]string, error) {
	var resp struct {
		Error bool              `json:"error"`
		Data  map[string]string `json:"data"`
	}
	path := fmt.Sprintf("/databases/%s/metrics", url.PathEscape(storageID))
	if err := makeRequest(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func GetValkeyLogs(storageID string, tail int) (*ValkeyLogs, error) {
	var resp struct {
		Error bool       `json:"error"`
		Data  ValkeyLogs `json:"data"`
	}
	path := fmt.Sprintf("/databases/%s/logs?tail=%s", url.PathEscape(storageID), url.QueryEscape(strconv.Itoa(tail)))
	if err := makeRequest(http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
