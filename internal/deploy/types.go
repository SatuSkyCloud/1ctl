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
	Name           string // App name (from satusky.toml or git remote fallback)
	CPU            string // Deprecated: legacy burst CPU flag alias.
	CPURequest     string
	CPULimit       string
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
	// Zone targeting. Region is intentionally not on the struct: the
	// legacy "zone-in-region" backward-compat shim was retired with
	// issue #24's backend-coordinated change.
	Zone string // Target zone (e.g., "my-kul-1b", "my-bki-1a")
	// PrebuiltImage, when non-empty, skips local Docker build and upload.
	// The image must already exist in the registry and be pullable by the cluster.
	PrebuiltImage string
	// FastBuild requests the backend's accelerated build backend. It is ignored
	// when PrebuiltImage is set and is intentionally separate for future billing.
	FastBuild bool
	// WaitFor declares TCP dependencies that must be reachable before the app starts.
	// The platform injects init containers so the main container never crashes while
	// dependencies are unavailable. Format: [{Host: "postgres", Port: 5432}]
	WaitFor []api.WaitFor
	// Deployment strategy options
	Strategy              string // "rolling" (default), "recreate"
	RollingMaxSurge       string // Rolling update max surge (e.g. "25%" or "1")
	RollingMaxUnavailable string // Rolling update max unavailable (e.g. "25%" or "0")
	// RollingFlagsExplicit is true when the user explicitly set either of the
	// rolling-* flags on the CLI. Used by buildStrategyConfig to decide whether
	// to omit the strategy config (default-suppression optimisation) or send
	// it through unchanged (so audit logs / version history capture the
	// user-specified value).
	RollingFlagsExplicit bool
	// TargetArch is the CPU architecture the image was built for ("amd64", "arm64", or "").
	// Empty means multi-arch or unknown — no arch-based machine filtering is applied.
	TargetArch string
	// Wait blocks after deploy until pods are Running or timeout is reached.
	Wait bool
}
