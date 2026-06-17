package api

import (
	"1ctl/internal/context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type StorageEngine string

const StorageEngineCNPG StorageEngine = "cnpg"

type StorageConfig struct {
	StorageID          uuid.UUID         `json:"storage_id,omitempty"`
	ResourceID         uuid.UUID         `json:"resource_id"`
	ResourceType       string            `json:"resource_type"`
	Namespace          string            `json:"namespace"`
	OrganizationID     *uuid.UUID        `json:"organization_id,omitempty"`
	Engine             StorageEngine     `json:"engine"`
	Version            string            `json:"version"`
	Replicas           int               `json:"replicas"`
	DatabaseName       *string           `json:"database_name,omitempty"`
	ClusterName        *string           `json:"cluster_name,omitempty"`
	Username           *string           `json:"username,omitempty"`
	Port               *string           `json:"port,omitempty"`
	StorageSize        string            `json:"storage_size"`
	StorageClass       string            `json:"storage_class"`
	CPURequest         string            `json:"cpu_request"`
	CPULimit           string            `json:"cpu_limit"`
	MemoryRequest      string            `json:"memory_request"`
	MemoryLimit        string            `json:"memory_limit"`
	WALStorageSize     *string           `json:"wal_storage_size,omitempty"`
	WALStorageClass    *string           `json:"wal_storage_class,omitempty"`
	Instances          *int              `json:"instances,omitempty"`
	ReplicationMode    *string           `json:"replication_mode,omitempty"`
	PersistenceEnabled *bool             `json:"persistence_enabled,omitempty"`
	AdminUIEnabled     *bool             `json:"admin_ui_enabled,omitempty"`
	Labels             map[string]string `json:"labels,omitempty"`
	Annotations        map[string]string `json:"annotations,omitempty"`
	CreatedAt          time.Time         `json:"created_at,omitempty"`
	UpdatedAt          time.Time         `json:"updated_at,omitempty"`
}

type PostgresCreateOptions struct {
	Name           string
	Database       string
	Username       string
	Version        string
	Instances      int
	StorageSize    string
	StorageClass   string
	WALStorageSize string
	CPURequest     string
	CPULimit       string
	MemoryRequest  string
	MemoryLimit    string
}

type PostgresCredentials struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	Host         string `json:"host"`
	Port         string `json:"port"`
	DBName       string `json:"dbname"`
	URI          string `json:"uri"`
	InternalHost string `json:"internal_host"`
	InternalURI  string `json:"internal_uri"`
	ExternalHost string `json:"external_host,omitempty"`
	ExternalPort string `json:"external_port,omitempty"`
	ExternalURI  string `json:"external_uri,omitempty"`
}

type PostgresStatus struct {
	Status                string `json:"status"`
	ClusterPhase          string `json:"cluster_phase"`
	ClusterExists         bool   `json:"cluster_exists"`
	ReadyInstances        int    `json:"ready_instances"`
	TotalInstances        int    `json:"total_instances"`
	Primary               string `json:"primary"`
	PoolerReady           bool   `json:"pooler_ready"`
	PoolerReadyReplicas   int    `json:"pooler_ready_replicas"`
	PoolerDesiredReplicas int    `json:"pooler_desired_replicas"`
	ExternalAccessible    bool   `json:"external_accessible"`
}

type CNPGDatabaseUser struct {
	UserID         uuid.UUID `json:"user_id"`
	StorageID      uuid.UUID `json:"storage_id"`
	Username       string    `json:"username"`
	SecretName     string    `json:"secret_name"`
	RoleAttributes []string  `json:"role_attributes"`
	Comment        *string   `json:"comment,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateDatabaseUserRequest struct {
	Username       string   `json:"username"`
	RoleAttributes []string `json:"role_attributes,omitempty"`
	Comment        string   `json:"comment,omitempty"`
}

type CreateDatabaseUserResponse struct {
	User                 CNPGDatabaseUser `json:"data"`
	Password             string           `json:"password"`
	Ready                bool             `json:"ready"`
	ReconciliationStatus string           `json:"reconciliation_status,omitempty"`
	ReadinessMessage     string           `json:"readiness_message,omitempty"`
}

type CNPGFirewallRule struct {
	RuleID      uuid.UUID `json:"rule_id"`
	StorageID   uuid.UUID `json:"storage_id"`
	Description string    `json:"description"`
	Cidr        string    `json:"cidr"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateFirewallRuleRequest struct {
	Description string `json:"description"`
	Cidr        string `json:"cidr"`
}

type UpdateFirewallRuleRequest struct {
	Description *string `json:"description,omitempty"`
	Cidr        *string `json:"cidr,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

type StorageClassInfo struct {
	Name        string `json:"name"`
	Provisioner string `json:"provisioner"`
	IsDefault   bool   `json:"is_default"`
}

func CreatePostgresCluster(opts PostgresCreateOptions) (*StorageConfig, error) {
	orgIDString := context.GetCurrentOrgID()
	if orgIDString == "" {
		return nil, fmt.Errorf("organization ID not found. Please run '1ctl auth login' first")
	}
	orgID, err := uuid.Parse(orgIDString)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID in profile: %w", err)
	}

	namespace, err := context.GetCurrentNamespaceOrError()
	if err != nil {
		return nil, err
	}

	resourceID := uuid.New()
	port := "5432"
	adminUIEnabled := false
	dbName := opts.Database
	clusterName := opts.Name
	username := opts.Username
	instances := opts.Instances

	req := StorageConfig{
		ResourceID:     resourceID,
		ResourceType:   "database",
		Namespace:      namespace,
		OrganizationID: &orgID,
		Engine:         StorageEngineCNPG,
		Version:        opts.Version,
		Replicas:       opts.Instances,
		DatabaseName:   &dbName,
		ClusterName:    &clusterName,
		Username:       &username,
		Port:           &port,
		StorageSize:    opts.StorageSize,
		StorageClass:   opts.StorageClass,
		CPURequest:     opts.CPURequest,
		CPULimit:       opts.CPULimit,
		MemoryRequest:  opts.MemoryRequest,
		MemoryLimit:    opts.MemoryLimit,
		Instances:      &instances,
		AdminUIEnabled: &adminUIEnabled,
	}
	if opts.WALStorageSize != "" {
		req.WALStorageSize = &opts.WALStorageSize
	}

	var resp struct {
		Error bool          `json:"error"`
		Data  StorageConfig `json:"data"`
	}
	if err := makeRequest("POST", "/storage/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func ListPostgresClusters(namespace string) ([]StorageConfig, error) {
	if namespace == "" {
		var err error
		namespace, err = context.GetCurrentNamespaceOrError()
		if err != nil {
			return nil, err
		}
	}

	var resp struct {
		Error bool            `json:"error"`
		Data  []StorageConfig `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/storage/namespace/%s", url.PathEscape(namespace)), nil, &resp); err != nil {
		return nil, err
	}

	clusters := make([]StorageConfig, 0, len(resp.Data))
	for _, storage := range resp.Data {
		if storage.Engine == StorageEngineCNPG {
			clusters = append(clusters, storage)
		}
	}
	return clusters, nil
}

func GetPostgresCluster(storageID string) (*StorageConfig, error) {
	var resp struct {
		Error bool          `json:"error"`
		Data  StorageConfig `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/storage/id/%s", url.PathEscape(storageID)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func DeletePostgresCluster(storageID string) error {
	return makeRequest("DELETE", fmt.Sprintf("/storage/%s", url.PathEscape(storageID)), nil, nil)
}

func RedeployPostgresCluster(storageID string) error {
	return makeRequest("POST", fmt.Sprintf("/storage/%s/redeploy", url.PathEscape(storageID)), nil, nil)
}

func GetPostgresStatus(storageID string) (*PostgresStatus, error) {
	var resp struct {
		Error bool           `json:"error"`
		Data  PostgresStatus `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/storage/%s/status", url.PathEscape(storageID)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func GetPostgresCredentials(storageID string) (*PostgresCredentials, error) {
	var resp struct {
		Error bool                `json:"error"`
		Data  PostgresCredentials `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/storage/%s/credentials", url.PathEscape(storageID)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func ListPostgresUsers(storageID string) ([]CNPGDatabaseUser, error) {
	var resp struct {
		Error bool               `json:"error"`
		Data  []CNPGDatabaseUser `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/storage/%s/database-users", url.PathEscape(storageID)), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func CreatePostgresUser(storageID string, req CreateDatabaseUserRequest) (*CreateDatabaseUserResponse, error) {
	var resp struct {
		Error                bool             `json:"error"`
		Data                 CNPGDatabaseUser `json:"data"`
		Password             string           `json:"password"`
		Ready                bool             `json:"ready"`
		ReconciliationStatus string           `json:"reconciliation_status,omitempty"`
		ReadinessMessage     string           `json:"readiness_message,omitempty"`
	}
	if err := makeRequest("POST", fmt.Sprintf("/storage/%s/database-users", url.PathEscape(storageID)), req, &resp); err != nil {
		return nil, err
	}
	return &CreateDatabaseUserResponse{
		User:                 resp.Data,
		Password:             resp.Password,
		Ready:                resp.Ready,
		ReconciliationStatus: resp.ReconciliationStatus,
		ReadinessMessage:     resp.ReadinessMessage,
	}, nil
}

func DeletePostgresUser(storageID, username string) error {
	return makeRequest("DELETE", fmt.Sprintf("/storage/%s/database-users/%s", url.PathEscape(storageID), url.PathEscape(username)), nil, nil)
}

func ListPostgresFirewallRules(storageID string) ([]CNPGFirewallRule, error) {
	var resp struct {
		Error bool               `json:"error"`
		Data  []CNPGFirewallRule `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/storage/%s/firewall-rules", url.PathEscape(storageID)), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func CreatePostgresFirewallRule(storageID string, req CreateFirewallRuleRequest) (*CNPGFirewallRule, error) {
	var resp struct {
		Error bool             `json:"error"`
		Data  CNPGFirewallRule `json:"data"`
	}
	if err := makeRequest("POST", fmt.Sprintf("/storage/%s/firewall-rules", url.PathEscape(storageID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func UpdatePostgresFirewallRule(storageID, ruleID string, req UpdateFirewallRuleRequest) (*CNPGFirewallRule, error) {
	var resp struct {
		Error bool             `json:"error"`
		Data  CNPGFirewallRule `json:"data"`
	}
	if err := makeRequest("PATCH", fmt.Sprintf("/storage/%s/firewall-rules/%s", url.PathEscape(storageID), url.PathEscape(ruleID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func DeletePostgresFirewallRule(storageID, ruleID string) error {
	return makeRequest("DELETE", fmt.Sprintf("/storage/%s/firewall-rules/%s", url.PathEscape(storageID), url.PathEscape(ruleID)), nil, nil)
}

func ListStorageClasses() ([]StorageClassInfo, error) {
	var resp struct {
		Error bool               `json:"error"`
		Data  []StorageClassInfo `json:"data"`
	}
	if err := makeRequest("GET", "/storage/storage-classes", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
