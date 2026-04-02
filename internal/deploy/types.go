package deploy

import "1ctl/internal/api"

// PDBConfigType represents the type of PodDisruptionBudget configuration
type PDBConfigType string

const (
	PDBTypeAuto    PDBConfigType = "auto"    // 50% rounded up (default)
	PDBTypeFixed   PDBConfigType = "fixed"   // Specific number of pods
	PDBTypePercent PDBConfigType = "percent" // Percentage
)

// PDBConfig represents PodDisruptionBudget configuration
type PDBConfig struct {
	Enabled      bool          `json:"enabled"`       // Whether PDB is enabled
	Type         PDBConfigType `json:"type"`          // auto, fixed, or percent
	MinAvailable *int32        `json:"min_available"` // Fixed number of pods (for type=fixed)
	Percent      *int32        `json:"percent"`       // Percentage (for type=percent)
}

type DeploymentOptions struct {
	CPU            string
	Memory         string
	Domain         string
	Organization   string
	Port           int
	DockerfilePath string
	Hostnames      []string
	Dependencies   []api.Dependency
	VolumeEnabled  bool
	Volume         *api.Volume
	EnvEnabled     bool
	Environment    *api.Environment
	// Multi-cluster deployment options
	MulticlusterEnabled   bool
	MulticlusterMode      string // "active-active" or "active-passive"
	BackupEnabled         bool   // Enable backups (auto-enabled for active-passive, optional for active-active)
	BackupSchedule        string // "hourly", "daily", "weekly"
	BackupRetention       string // "24h", "72h", "168h", "720h"
	BackupPriorityCluster int    // Which cluster performs backups (1 = primary, 2 = secondary)
	// HA settings
	Replicas  int            // Manual replica count override (0 = auto from hostnames)
	PDBConfig *PDBConfig     `json:"pdb_config,omitempty"`
	HPAConfig *api.HPAConfig `json:"hpa_config,omitempty"`
	VPAConfig *api.VPAConfig `json:"vpa_config,omitempty"`
	// PrebuiltImage, when non-empty, skips local Docker build and upload.
	// The image must already exist in the registry and be pullable by the cluster.
	PrebuiltImage string
	// WaitFor declares TCP dependencies that must be reachable before the app starts.
	// The platform injects init containers so the main container never crashes while
	// dependencies are unavailable. Format: [{Host: "postgres", Port: 5432}]
	WaitFor []api.WaitFor
	// Deployment strategy options
	Strategy              string // "rolling" (default), "recreate"
	RollingMaxSurge       string // Rolling update max surge (e.g. "25%" or "1")
	RollingMaxUnavailable string // Rolling update max unavailable (e.g. "25%" or "0")
}
