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
	TokenID          string    `json:"token_id,omitempty"`
	UserID           string    `json:"user_id,omitempty"`
	Token            string    `json:"token,omitempty"`
	UserEmail        string    `json:"user_email,omitempty"`
	UserConfigKey    string    `json:"user_config_key,omitempty"`
	OrganizationID   string    `json:"organization_id,omitempty"`
	OrganizationName string    `json:"organization_name,omitempty"`
	Namespace        string    `json:"namespace,omitempty"`
	Message          string    `json:"message,omitempty"`
}

// MulticlusterConfig represents multi-cluster deployment configuration
type MulticlusterConfig struct {
	Enabled               bool   `json:"enabled"`
	Mode                  string `json:"mode"`                              // "active-active" or "active-passive"
	TargetClusters        string `json:"target_clusters,omitempty"`         // "all" or comma-separated cluster names
	BackupEnabled         bool   `json:"backup_enabled,omitempty"`          // Enable Velero backups (works for both modes)
	BackupSchedule        string `json:"backup_schedule,omitempty"`         // Cron schedule (e.g., "0 0 * * *")
	BackupRetention       string `json:"backup_retention,omitempty"`        // Duration (e.g., "168h" for 7 days)
	BackupPriorityCluster int    `json:"backup_priority_cluster,omitempty"` // Which priority cluster performs backups (1 = highest priority)
	FailoverEnabled       bool   `json:"failover_enabled,omitempty"`        // Enable automatic failover (active-passive only)
	RestoreOnFailover     bool   `json:"restore_on_failover,omitempty"`     // Restore from backup on failover (active-passive only)
}

// PDBConfig represents PodDisruptionBudget configuration
type PDBConfig struct {
	Enabled      bool   `json:"enabled"`
	Type         string `json:"type"`                    // "auto", "fixed", "percent"
	MinAvailable *int32 `json:"min_available,omitempty"` // For type=fixed
	Percent      *int32 `json:"percent,omitempty"`       // For type=percent
}

// HPAConfig represents HorizontalPodAutoscaler configuration
type HPAConfig struct {
	Enabled                bool   `json:"enabled"`
	MinReplicas            int32  `json:"min_replicas"`
	MaxReplicas            int32  `json:"max_replicas"`
	CPUTarget              *int32 `json:"cpu_target,omitempty"`
	MemoryTarget           *int32 `json:"memory_target,omitempty"`
	ScaleUpStabilization   *int32 `json:"scale_up_stabilization,omitempty"`
	ScaleDownStabilization *int32 `json:"scale_down_stabilization,omitempty"`
	ScaleUpMaxPods         *int32 `json:"scale_up_max_pods,omitempty"`
	ScaleDownMaxPods       *int32 `json:"scale_down_max_pods,omitempty"`
}

// VPAConfig represents VerticalPodAutoscaler configuration
type VPAConfig struct {
	Enabled    bool   `json:"enabled"`
	UpdateMode string `json:"update_mode"`          // "Off", "Initial", "Auto"
	MinCPU     string `json:"min_cpu,omitempty"`    // e.g., "100m"
	MaxCPU     string `json:"max_cpu,omitempty"`    // e.g., "4"
	MinMemory  string `json:"min_memory,omitempty"` // e.g., "128Mi"
	MaxMemory  string `json:"max_memory,omitempty"` // e.g., "8Gi"
}

type Deployment struct {
	DeploymentID       uuid.UUID           `json:"deployment_id,omitempty"`
	UserID             uuid.UUID           `json:"user_id"`
	Hostnames          []string            `json:"hostnames"`
	Type               string              `json:"type"`
	Zone               string              `json:"zone"`
	Region             string              `json:"region"`
	SSD                string              `json:"ssd"`
	GPU                string              `json:"gpu"`
	Namespace          string              `json:"namespace"`
	Replicas           int32               `json:"replicas"`
	Image              string              `json:"image"`
	AppLabel           string              `json:"app_label"`
	Port               int32               `json:"port"`
	CpuRequest         string              `json:"cpu_request"`
	MemoryRequest      string              `json:"memory_request"`
	MemoryLimit        string              `json:"memory_limit"`
	RepoURL            string              `json:"repo_url,omitempty"`
	BranchName         string              `json:"branch_name,omitempty"`
	DockerfilePath     string              `json:"dockerfile_path,omitempty"`
	EnvEnabled         bool                `json:"env_enabled"`
	SecretEnabled      bool                `json:"secret_enabled"`
	VolumeEnabled      bool                `json:"volume_enabled"`
	Status             string              `json:"status"`
	Environment        string              `json:"environment"`
	MarketplaceAppName string              `json:"marketplace_app_name"`
	MulticlusterConfig *MulticlusterConfig `json:"multicluster_config,omitempty"`
	PDBConfig          *PDBConfig          `json:"pdb_config,omitempty"`
	HPAConfig          *HPAConfig          `json:"hpa_config,omitempty"`
	VPAConfig          *VPAConfig          `json:"vpa_config,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
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

type UserProfile struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Organization string    `json:"organization"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
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
	StatusPending     = "pending"
	StatusCreating    = "creating"
	StatusRunning     = "running"
	StatusFailed      = "failed"
	StatusCompleted   = "completed"
	StatusNotReady    = "NotReady"
	StatusRunningK8s  = "Running"
	StatusFailedK8s   = "Failed"
	StatusProgressing = "Progressing"
	StatusUnknown     = "Unknown"
)

type DeploymentStatus struct {
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	Progress int    `json:"progress"`
}

type Machine struct {
	ID                  int64      `db:"id" json:"id"`
	MachineID           string     `db:"machine_id" json:"machine_id" validate:"required"`
	MachineName         string     `db:"machine_name" json:"machine_name" validate:"required"`
	MachineTypes        []string   `db:"machine_types" json:"machine_types" validate:"required"`
	OwnerID             uuid.UUID  `db:"owner_id" json:"owner_id" validate:"required,uuid"`
	IsVerified          bool       `db:"is_verified" json:"is_verified" validate:"required"`
	MachineRegion       string     `db:"machine_region" json:"machine_region" validate:"required"`
	MachineZone         string     `db:"machine_zone" json:"machine_zone" validate:"required"`
	IpAddr              string     `db:"ip_addr" json:"ip_addr" validate:"required"`
	TalosVersion        string     `db:"talos_version" json:"talos_version" validate:"required"`
	KubernetesVersion   string     `db:"kubernetes_version" json:"kubernetes_version" validate:"required"`
	CPUCores            int        `db:"cpu_cores" json:"cpu_cores" validate:"required"`
	MemoryGB            int        `db:"memory_gb" json:"memory_gb" validate:"required"`
	StorageGB           int        `db:"storage_gb" json:"storage_gb" validate:"required"`
	GPUCount            int        `db:"gpu_count" json:"gpu_count" validate:"required"`
	GPUType             string     `db:"gpu_type" json:"gpu_type" validate:"required"`
	BandwidthGbps       int        `db:"bandwidth_gbps" json:"bandwidth_gbps" validate:"required"`
	Brand               string     `db:"brand" json:"brand" validate:"required"`
	Model               string     `db:"model" json:"model" validate:"required"`
	Manufacturer        string     `db:"manufacturer" json:"manufacturer" validate:"required"`
	FormFactor          string     `db:"form_factor" json:"form_factor" validate:"required"`
	Monetized           bool       `db:"monetized" json:"monetized" validate:"required"`
	Status              string     `db:"status" json:"status"`
	LastHealthCheck     *time.Time `db:"last_health_check" json:"last_health_check"`
	Recommended         bool       `db:"recommended" json:"recommended"`
	ResourceScore       *float64   `db:"resource_score" json:"resource_score"`
	CPUUsagePercent     *float64   `db:"cpu_usage_percent" json:"cpu_usage_percent"`
	MemoryUsagePercent  *float64   `db:"memory_usage_percent" json:"memory_usage_percent"`
	StorageUsagePercent *float64   `db:"storage_usage_percent" json:"storage_usage_percent"`
	NetworkUsageGbps    *float64   `db:"network_usage_gbps" json:"network_usage_gbps"`
	NetworkMetricsType  string     `db:"network_metrics_type" json:"network_metrics_type"`
	ConnectionMode      *string    `db:"connection_mode" json:"connection_mode,omitempty"`
	VMState             *string    `db:"vm_state" json:"vm_state,omitempty"`
	UptimePercent       *float64   `db:"uptime_percent" json:"uptime_percent"`
	ResponseTimeMs      *int       `db:"response_time_ms" json:"response_time_ms"`
	NodeType            string     `db:"node_type" json:"node_type"`
	HasGPU              bool       `db:"has_gpu" json:"has_gpu"`
	HasHDD              bool       `db:"has_hdd" json:"has_hdd"`
	HasNVME             bool       `db:"has_nvme" json:"has_nvme"`
	PricingTier         string     `db:"pricing_tier" json:"pricing_tier"`
	HourlyCost          float64    `db:"hourly_cost" json:"hourly_cost"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at" json:"updated_at"`
}

type MachineIDs struct {
	MachineIDs []string `json:"machine_ids" validate:"required"`
}

// Mac agent command type constants
const (
	CmdStart         = "CMD_START"
	CmdStop          = "CMD_STOP"
	CmdForceStop     = "CMD_FORCE_STOP"
	CmdReboot        = "CMD_REBOOT"
	CmdResize        = "CMD_RESIZE"
	CmdApplyConfig   = "CMD_APPLY_TALOS_CONFIG"
	CmdStreamConsole = "CMD_STREAM_CONSOLE"
)

// Domain management types

// Domain represents a domain registered in the system
type Domain struct {
	DomainID  string    `json:"domain_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	TTL       int       `json:"ttl"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DNSRecord represents a DNS record for a domain
type DNSRecord struct {
	RecordID  string    `json:"record_id"`
	DomainID  string    `json:"domain_id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Data      string    `json:"data"`
	Priority  *int      `json:"priority,omitempty"`
	Port      *int      `json:"port,omitempty"`
	Weight    *int      `json:"weight,omitempty"`
	TTL       int       `json:"ttl"`
	Flags     *int      `json:"flags,omitempty"`
	Tag       *string   `json:"tag,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DomainPrice represents pricing information for a domain
type DomainPrice struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// DomainAvailabilityResult is the result of a domain availability check
type DomainAvailabilityResult struct {
	Domain       string       `json:"domain"`
	Available    bool         `json:"available"`
	Status       string       `json:"status"`
	Reason       string       `json:"reason,omitempty"`
	IsPremium    bool         `json:"is_premium,omitempty"`
	Price        *DomainPrice `json:"price,omitempty"`
	PremiumPrice float64      `json:"premium_price,omitempty"`
}

// DomainSearchResult is a single domain search result
type DomainSearchResult struct {
	Domain    string  `json:"domain"`
	Extension string  `json:"extension"`
	Available bool    `json:"available"`
	Status    string  `json:"status"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
	IsPremium bool    `json:"is_premium"`
	Period    int     `json:"period"`
}

// NameserverStatus is the result of a nameserver verification
type NameserverStatus struct {
	Verified            bool     `json:"verified"`
	CurrentNameservers  []string `json:"current_nameservers"`
	ExpectedNameservers []string `json:"expected_nameservers"`
	Message             string   `json:"message"`
}

// DomainContactInfo contains contact information for domain registration
type DomainContactInfo struct {
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	Email            string `json:"email"`
	PhoneCountryCode string `json:"phone_country_code"`
	PhoneNumber      string `json:"phone_number"`
	Street           string `json:"street"`
	StreetNumber     string `json:"street_number"`
	PostalCode       string `json:"postal_code"`
	City             string `json:"city"`
	State            string `json:"state,omitempty"`
	Country          string `json:"country"`
	CompanyName      string `json:"company_name,omitempty"`
}

// DomainCreateRequest is the request for creating a new domain
type DomainCreateRequest struct {
	Name      string `json:"name"`
	IPAddress string `json:"ip_address,omitempty"`
}

// DomainCheckRequest is the request for checking domain availability
type DomainCheckRequest struct {
	Domains   []string `json:"domains"`
	WithPrice bool     `json:"with_price"`
}

// DomainSearchRequest is the request for searching domain availability
type DomainSearchRequest struct {
	DomainName string   `json:"domain_name"`
	Extensions []string `json:"extensions,omitempty"`
	Period     int      `json:"period,omitempty"`
}

// DomainPurchaseRequest is the request for purchasing a domain
type DomainPurchaseRequest struct {
	Domain  string             `json:"domain"`
	Period  int                `json:"period,omitempty"`
	Contact *DomainContactInfo `json:"contact"`
}

// DomainPurchaseIntentResponse is returned when a domain purchase intent is created
type DomainPurchaseIntentResponse struct {
	SessionID   string  `json:"session_id"`
	RedirectURL string  `json:"redirect_url"`
	Domain      string  `json:"domain"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	IntentID    string  `json:"intent_id"`
}

// DomainPurchaseIntentStatus is the status of a domain purchase intent
type DomainPurchaseIntentStatus struct {
	IntentID string `json:"intent_id"`
	Domain   string `json:"domain"`
	Status   string `json:"status"`
}

// DNSRecordCreateRequest is the request for creating a new DNS record
type DNSRecordCreateRequest struct {
	Type     string  `json:"type"`
	Name     string  `json:"name"`
	Data     string  `json:"data"`
	Priority *int    `json:"priority,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Weight   *int    `json:"weight,omitempty"`
	TTL      *int    `json:"ttl,omitempty"`
	Flags    *int    `json:"flags,omitempty"`
	Tag      *string `json:"tag,omitempty"`
}

// DNSRecordUpdateRequest is the request for updating a DNS record
type DNSRecordUpdateRequest struct {
	Type     *string `json:"type,omitempty"`
	Name     *string `json:"name,omitempty"`
	Data     *string `json:"data,omitempty"`
	Priority *int    `json:"priority,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Weight   *int    `json:"weight,omitempty"`
	TTL      *int    `json:"ttl,omitempty"`
	Flags    *int    `json:"flags,omitempty"`
	Tag      *string `json:"tag,omitempty"`
}

// ============================================================
// Machine Usage (user-facing)
// ============================================================

// MachineUsageRecord represents a raw machine usage record from the usage tracking API
type MachineUsageRecord struct {
	UsageID        string  `json:"usage_id"`
	DeploymentID   string  `json:"deployment_id"`
	MachineID      string  `json:"machine_id"`
	UserID         string  `json:"user_id"`
	OrganizationID string  `json:"organization_id"`
	StartTime      string  `json:"start_time"`
	EndTime        *string `json:"end_time,omitempty"`
	HourlyRate     float64 `json:"hourly_rate"`
	Status         string  `json:"status"`
	IsBilled       bool    `json:"is_billed"`
	LastBilledAt   *string `json:"last_billed_at,omitempty"`
	MachineRefID   int64   `json:"machine_ref_id,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// UsageCostResponse is the response from GET /machine-usage/:usageId/cost
type UsageCostResponse struct {
	UsageID string  `json:"usage_id"`
	Cost    float64 `json:"cost"`
}

// ============================================================
// Pricing Configuration
// ============================================================

// PricingConfig represents a pricing configuration entry
type PricingConfig struct {
	ConfigID          string  `json:"config_id"`
	Region            string  `json:"region"`
	MachineType       string  `json:"machine_type"`
	SLATier           string  `json:"sla_tier"`
	BaseHourlyRate    float64 `json:"base_hourly_rate"`
	CPUCoreRate       float64 `json:"cpu_core_rate"`
	MemoryGBRate      float64 `json:"memory_gb_rate"`
	StorageGBRate     float64 `json:"storage_gb_rate"`
	GPURate           float64 `json:"gpu_rate"`
	BandwidthGBPSRate float64 `json:"bandwidth_gbps_rate"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

// CostCalculationRequest is the request body for POST /pricing/calculate/:machine_ref_id/:machine_id
type CostCalculationRequest struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// CostBreakdown contains per-component cost details
type CostBreakdown struct {
	BaseCost      float64 `json:"base_cost"`
	CPUCost       float64 `json:"cpu_cost"`
	MemoryCost    float64 `json:"memory_cost"`
	StorageCost   float64 `json:"storage_cost"`
	GPUCost       float64 `json:"gpu_cost"`
	BandwidthCost float64 `json:"bandwidth_cost"`
}

// CostCalculationResponse is the response from POST /pricing/calculate
type CostCalculationResponse struct {
	MachineID    string        `json:"machine_id"`
	MachineRefID int64         `json:"machine_ref_id"`
	StartTime    string        `json:"start_time"`
	EndTime      string        `json:"end_time"`
	TotalCost    float64       `json:"total_cost"`
	Breakdown    CostBreakdown `json:"breakdown"`
}

// ============================================================
// Billing Settings
// ============================================================

// AutoTopupSettings represents the auto-topup configuration for an organization
type AutoTopupSettings struct {
	SettingsID         string  `json:"settings_id"`
	OrganizationID     string  `json:"organization_id"`
	Enabled            bool    `json:"enabled"`
	ThresholdAmount    float64 `json:"threshold_amount"`
	TopupAmount        float64 `json:"topup_amount"`
	HasPaymentMethod   bool    `json:"has_payment_method"`
	PaymentMethodLast4 string  `json:"payment_method_last4,omitempty"`
	PaymentMethodBrand string  `json:"payment_method_brand,omitempty"`
	LastTopupAt        *string `json:"last_topup_at,omitempty"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// AutoTopupSettingsRequest is the request to update auto-topup settings
type AutoTopupSettingsRequest struct {
	Enabled         bool    `json:"enabled"`
	ThresholdAmount float64 `json:"threshold_amount"`
	TopupAmount     float64 `json:"topup_amount"`
}

// NotificationPreferences represents the billing notification preferences
type NotificationPreferences struct {
	ID                         string   `json:"id"`
	OrganizationID             string   `json:"organization_id"`
	EmailEnabled               bool     `json:"email_enabled"`
	InAppEnabled               bool     `json:"in_app_enabled"`
	WebhookEnabled             bool     `json:"webhook_enabled"`
	WebhookURL                 string   `json:"webhook_url,omitempty"`
	EnabledTypes               []string `json:"enabled_types"`
	QuietHoursEnabled          bool     `json:"quiet_hours_enabled"`
	QuietHoursTimezone         string   `json:"quiet_hours_timezone,omitempty"`
	DigestEnabled              bool     `json:"digest_enabled"`
	DigestFrequency            string   `json:"digest_frequency,omitempty"`
	LowBalanceThresholdPercent int      `json:"low_balance_threshold_percent"`
	LowBalanceThresholdAmount  float64  `json:"low_balance_threshold_amount"`
	AlertCooldownHours         int      `json:"alert_cooldown_hours"`
}

// NotificationPreferencesRequest is the request to update notification preferences
type NotificationPreferencesRequest struct {
	EmailEnabled               bool     `json:"email_enabled"`
	InAppEnabled               bool     `json:"in_app_enabled"`
	WebhookEnabled             bool     `json:"webhook_enabled"`
	WebhookURL                 string   `json:"webhook_url,omitempty"`
	EnabledTypes               []string `json:"enabled_types,omitempty"`
	QuietHoursEnabled          bool     `json:"quiet_hours_enabled"`
	QuietHoursStart            string   `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd              string   `json:"quiet_hours_end,omitempty"`
	QuietHoursTimezone         string   `json:"quiet_hours_timezone,omitempty"`
	DigestEnabled              bool     `json:"digest_enabled"`
	DigestFrequency            string   `json:"digest_frequency,omitempty"`
	LowBalanceThresholdPercent int      `json:"low_balance_threshold_percent"`
	LowBalanceThresholdAmount  float64  `json:"low_balance_threshold_amount"`
	AlertCooldownHours         int      `json:"alert_cooldown_hours"`
}

// SendCommandRequest is the request body for POST /machines/:machineId/command
type SendCommandRequest struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// SendCommandResponse is the response from POST /machines/:machineId/command
type SendCommandResponse struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
}
