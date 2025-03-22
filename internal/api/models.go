package api

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type APIToken struct {
	TokenID   uuid.UUID `json:"token_id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Token     string    `json:"token,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TokenValidate struct {
	Valid            bool      `json:"valid"`
	ExpiresAt        time.Time `json:"expires_at"`
	IsActive         bool      `json:"is_active"`
	TokenID          uuid.UUID `json:"token_id,omitempty"`
	UserID           uuid.UUID `json:"user_id,omitempty"`
	UserEmail        string    `json:"user_email,omitempty"`
	UserConfigKey    string    `json:"user_config_key,omitempty"`
	OrganizationID   uuid.UUID `json:"organization_id,omitempty"`
	OrganizationName string    `json:"organization_name,omitempty"`
}

type Deployment struct {
	DeploymentID       uuid.UUID `json:"deployment_id,omitempty"`
	UserID             uuid.UUID `json:"user_id"`
	Hostnames          []string  `json:"hostnames"`
	Type               string    `json:"type"`
	Zone               string    `json:"zone"`
	Region             string    `json:"region"`
	SSD                string    `json:"ssd"`
	GPU                string    `json:"gpu"`
	Namespace          string    `json:"namespace"`
	Replicas           int32     `json:"replicas"`
	Image              string    `json:"image"`
	AppLabel           string    `json:"app_label"`
	Port               int32     `json:"port"`
	CpuRequest         string    `json:"cpu_request"`
	MemoryRequest      string    `json:"memory_request"`
	MemoryLimit        string    `json:"memory_limit"`
	RepoURL            string    `json:"repo_url,omitempty"`
	BranchName         string    `json:"branch_name,omitempty"`
	DockerfilePath     string    `json:"dockerfile_path,omitempty"`
	EnvEnabled         bool      `json:"env_enabled"`
	SecretEnabled      bool      `json:"secret_enabled"`
	VolumeEnabled      bool      `json:"volume_enabled"`
	Status             string    `json:"status"`
	Environment        string    `json:"environment"`
	MarketplaceAppName string    `json:"marketplace_app_name"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreateDeploymentResponse struct {
	DeploymentID uuid.UUID `json:"deployment_id"`
	AppLabel     string    `json:"app_label"`
	Domain       string    `json:"domain"`
}

type Secret struct {
	SecretID     uuid.UUID      `json:"secret_id"`
	DeploymentID uuid.UUID      `json:"deployment_id"`
	Namespace    string         `json:"namespace"`
	KeyValues    []KeyValuePair `json:"key_values"`
	AppLabel     string         `json:"app_label"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Ingress struct {
	IngressID    uuid.UUID     `json:"ingress_id"`
	DeploymentID uuid.UUID     `json:"deployment_id"`
	Namespace    string        `json:"namespace"`
	ServiceID    uuid.UUID     `json:"service_id"`
	AppLabel     string        `json:"app_label"`
	Port         int32         `json:"port"`
	DomainName   string        `json:"domain_name"`
	DnsConfig    DnsConfigType `json:"dns_config"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type Dependency struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Service *Service `json:"service,omitempty"`
	Volume  *Volume  `json:"volume,omitempty"`
}

type Volume struct {
	VolumeID     uuid.UUID `json:"volume_id"`
	DeploymentID uuid.UUID `json:"deployment_id"`
	VolumeName   string    `json:"volume_name"`
	StorageClass string    `json:"storage_class"`
	StorageSize  string    `json:"storage_size"`
	ClaimName    string    `json:"claim_name"`
	MountPath    string    `json:"mount_path"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Service struct {
	ServiceID    uuid.UUID `json:"service_id"`
	DeploymentID uuid.UUID `json:"deployment_id"`
	Namespace    string    `json:"namespace"`
	ServiceName  string    `json:"service_name"`
	Port         int32     `json:"port"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Environment struct {
	EnvironmentID uuid.UUID      `json:"environment_id"`
	DeploymentID  uuid.UUID      `json:"deployment_id"`
	Namespace     string         `json:"namespace"`
	AppLabel      string         `json:"app_label"`
	KeyValues     []KeyValuePair `json:"key_values"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type Issuer struct {
	DeploymentID uuid.UUID `json:"deployment_id"`
	IssuerID     uuid.UUID `json:"issuer_id"`
	Namespace    string    `json:"namespace"`
	UserEmail    string    `json:"user_email"`
	Environment  string    `json:"environment"`
	AppLabel     string    `json:"app_label"`
	IssuerName   string    `json:"issuer_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Organization struct {
	OrganizationID   uuid.UUID `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	Description      string    `json:"description,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type User struct {
	UserID          uuid.UUID `json:"user_id"`
	Email           string    `json:"email"`
	UserName        string    `json:"user_name"`
	OrganizationIDs []string  `json:"organization_ids"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type DnsConfigType string

const (
	DnsConfigDefault DnsConfigType = "default"
	DnsConfigCustom  DnsConfigType = "custom"
)

type APIError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("(%s): %s", e.Code, e.Message)
}

const (
	StatusPending   = "pending"
	StatusCreating  = "creating"
	StatusRunning   = "running"
	StatusFailed    = "failed"
	StatusCompleted = "completed"
)

type DeploymentStatus struct {
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	Progress int    `json:"progress"`
}

type Machine struct {
	MachineID         uuid.UUID `db:"machine_id" json:"machine_id" validate:"required,uuid"`
	MachineName       string    `db:"machine_name" json:"machine_name" validate:"required"`
	MachineTypes      []string  `db:"machine_types" json:"machine_types" validate:"required"`
	OwnerID           uuid.UUID `db:"owner_id" json:"owner_id" validate:"required,uuid"`
	IsVerified        bool      `db:"is_verified" json:"is_verified" validate:"required"`
	MachineRegion     string    `db:"machine_region" json:"machine_region" validate:"required"`
	MachineZone       string    `db:"machine_zone" json:"machine_zone" validate:"required"`
	IpAddr            string    `db:"ip_addr" json:"ip_addr" validate:"required"`
	TalosVersion      string    `db:"talos_version" json:"talos_version" validate:"required"`
	KubernetesVersion string    `db:"kubernetes_version" json:"kubernetes_version" validate:"required"`
	CPUCores          int       `db:"cpu_cores" json:"cpu_cores" validate:"required"`
	MemoryGB          int       `db:"memory_gb" json:"memory_gb" validate:"required"`
	StorageGB         int       `db:"storage_gb" json:"storage_gb" validate:"required"`
	GPUCount          int       `db:"gpu_count" json:"gpu_count" validate:"required"`
	GPUType           string    `db:"gpu_type" json:"gpu_type" validate:"required"`
	BandwidthGbps     int       `db:"bandwidth_gbps" json:"bandwidth_gbps" validate:"required"`
	Brand             string    `db:"brand" json:"brand" validate:"required"`
	Model             string    `db:"model" json:"model" validate:"required"`
	Manufacturer      string    `db:"manufacturer" json:"manufacturer" validate:"required"`
	FormFactor        string    `db:"form_factor" json:"form_factor" validate:"required"`
	Monetized         bool      `db:"monetized" json:"monetized" validate:"required"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

type MachineIDs struct {
	MachineIDs []uuid.UUID `json:"machine_ids" validate:"required"`
}
