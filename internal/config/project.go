package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const DefaultConfigFile = "satusky.toml"

// ProjectConfig is the in-memory representation of satusky.toml.
//
// Field-by-field precedence at deploy time (highest wins):
//
//	CLI flag (explicit, c.IsSet) > satusky.toml > platform default (Value: on flag)
//
// Fields with the zero value are treated as "not set" by the merge.
type ProjectConfig struct {
	App          AppConfig          `toml:"app"`
	Build        BuildConfig        `toml:"build"`
	Checks       ChecksConfig       `toml:"checks"`
	Deploy       DeployConfig       `toml:"deploy"`
	Volume       VolumeConfig       `toml:"volume"`
	HPA          HPAConfig          `toml:"hpa"`
	VPA          VPAConfig          `toml:"vpa"`
	PDB          PDBConfig          `toml:"pdb"`
	Multicluster MulticlusterConfig `toml:"multicluster"`
	Path         string             `toml:"-"`
}

// AppConfig holds app identity and resource fields.
// For build, health-check, and deploy-strategy settings see [build], [checks], [deploy].
type AppConfig struct {
	Name                  string   `toml:"name"`
	Port                  int      `toml:"port"`
	CPU                   string   `toml:"cpu"` // Deprecated: legacy burst CPU alias.
	CPURequest            string   `toml:"cpu_request"`
	CPULimit              string   `toml:"cpu_limit"`
	Memory                string   `toml:"memory"`
	Replicas              int      `toml:"replicas"`
	Domain                string   `toml:"domain"`
	Zone                  string   `toml:"zone"`
	Organization          string   `toml:"organization"`

	// Backward-compat: fields below were moved to [build], [checks], or [deploy] in the v2 schema.
	// Normalize() copies them to the preferred location when the target section is empty.
	Dockerfile            string   `toml:"dockerfile"`
	FastBuild             bool     `toml:"fast_build"`
	HealthPath            string   `toml:"health_path"`
	Strategy              string   `toml:"strategy"`
	RollingMaxSurge       string   `toml:"rolling_max_surge"`
	RollingMaxUnavailable string   `toml:"rolling_max_unavailable"`
	MachineTag            string   `toml:"machine_tag"`
	WaitFor               []string `toml:"wait_for"`
}

// BuildConfig controls how the container image is built.
type BuildConfig struct {
	Dockerfile string `toml:"dockerfile"`
	FastBuild  bool   `toml:"fast_build"`
}

// ChecksConfig controls deployment health checks and smoke testing.
type ChecksConfig struct {
	HealthPath string `toml:"health_path"`
}

// DeployConfig controls deployment strategy and runtime placement.
type DeployConfig struct {
	Strategy              string   `toml:"strategy"`
	RollingMaxSurge       string   `toml:"rolling_max_surge"`
	RollingMaxUnavailable string   `toml:"rolling_max_unavailable"`
	MachineTag            string   `toml:"machine_tag"`
	WaitFor               []string `toml:"wait_for"`
}

type VolumeConfig struct {
	Size  string `toml:"size"`
	Mount string `toml:"mount"`
}

type HPAConfig struct {
	Enabled      bool  `toml:"enabled"`
	MinReplicas  int32 `toml:"min_replicas"`
	MaxReplicas  int32 `toml:"max_replicas"`
	CPUTarget    int32 `toml:"cpu_target"`
	MemoryTarget int32 `toml:"memory_target"`
}

type VPAConfig struct {
	Enabled   bool   `toml:"enabled"`
	Mode      string `toml:"mode"`
	MinCPU    string `toml:"min_cpu"`
	MaxCPU    string `toml:"max_cpu"`
	MinMemory string `toml:"min_memory"`
	MaxMemory string `toml:"max_memory"`
}

// PDBConfig fields match the API surface (api.PDBConfig): only MinAvailable
// and Percent are accepted today. MaxUnavailable is intentionally not on the
// struct — the platform doesn't yet support it and silently dropping the
// field would surprise users who set it expecting it to work.
type PDBConfig struct {
	Enabled      bool   `toml:"enabled"`
	Type         string `toml:"type"`
	MinAvailable int32  `toml:"min_available"`
	Percent      int32  `toml:"percent"`
}

type MulticlusterConfig struct {
	Enabled               bool   `toml:"enabled"`
	Mode                  string `toml:"mode"`
	BackupEnabled         bool   `toml:"backup_enabled"`
	BackupSchedule        string `toml:"backup_schedule"`
	BackupRetention       string `toml:"backup_retention"`
	BackupPriorityCluster int    `toml:"backup_priority_cluster"`
}

// LoadConfig resolves and loads the config file. Returns an error if not found.
func LoadConfig(configArg string) (*ProjectConfig, error) {
	path, err := resolveConfigPath(configArg)
	if err != nil {
		return nil, err
	}
	var cfg ProjectConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", path, err)
	}
	cfg.Path = path
	cfg.Normalize()
	return &cfg, nil
}

// FindConfig looks for a config file without requiring one to exist. Returns nil, nil if not found.
func FindConfig(configArg string) (*ProjectConfig, error) {
	path, err := resolveConfigPath(configArg)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg ProjectConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", path, err)
	}
	cfg.Path = path
	cfg.Normalize()
	return &cfg, nil
}

// Normalize applies backward-compatible field migrations.
// When the preferred section ([build], [checks], [deploy]) is empty/zero,
// values are copied from the legacy [app] location so old satusky.toml files
// continue to work without changes. The legacy [app] fields are then cleared
// so downstream consumers always read from the canonical v2 sections.
func (cfg *ProjectConfig) Normalize() {
	// Build: prefer [build] over [app].
	if cfg.Build.Dockerfile == "" {
		cfg.Build.Dockerfile = cfg.App.Dockerfile
	}
	if !cfg.Build.FastBuild {
		cfg.Build.FastBuild = cfg.App.FastBuild
	}
	// Checks: prefer [checks] over [app].
	if cfg.Checks.HealthPath == "" {
		cfg.Checks.HealthPath = cfg.App.HealthPath
	}
	// Deploy: prefer [deploy] over [app].
	if cfg.Deploy.Strategy == "" {
		cfg.Deploy.Strategy = cfg.App.Strategy
	}
	if cfg.Deploy.RollingMaxSurge == "" {
		cfg.Deploy.RollingMaxSurge = cfg.App.RollingMaxSurge
	}
	if cfg.Deploy.RollingMaxUnavailable == "" {
		cfg.Deploy.RollingMaxUnavailable = cfg.App.RollingMaxUnavailable
	}
	if cfg.Deploy.MachineTag == "" {
		cfg.Deploy.MachineTag = cfg.App.MachineTag
	}
	if len(cfg.Deploy.WaitFor) == 0 {
		cfg.Deploy.WaitFor = cfg.App.WaitFor
	}

	// Clear legacy [app] fields so downstream consumers always read from
	// the canonical v2 sections.
	cfg.App.Dockerfile = ""
	cfg.App.FastBuild = false
	cfg.App.HealthPath = ""
	cfg.App.Strategy = ""
	cfg.App.RollingMaxSurge = ""
	cfg.App.RollingMaxUnavailable = ""
	cfg.App.MachineTag = ""
	cfg.App.WaitFor = nil
}

// Save writes the config back to its original path.
func (cfg *ProjectConfig) Save() error {
	f, err := os.Create(cfg.Path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }() //nolint:errcheck
	return toml.NewEncoder(f).Encode(cfg)
}

func resolveConfigPath(configArg string) (string, error) {
	if configArg != "" {
		if strings.HasSuffix(configArg, ".toml") {
			if _, err := os.Stat(configArg); err != nil {
				return "", err
			}
			return configArg, nil
		}
		base, err := findDefaultConfigDir()
		if err != nil {
			return "", fmt.Errorf("satusky.toml not found; cannot resolve --config %s", configArg)
		}
		path := filepath.Join(base, fmt.Sprintf("satusky.%s.toml", configArg))
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("%s not found", path)
		}
		return path, nil
	}
	dir, err := findDefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultConfigFile), nil
}

func findDefaultConfigDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, DefaultConfigFile)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}
