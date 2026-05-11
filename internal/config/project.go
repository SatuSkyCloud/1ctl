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
	Volume       VolumeConfig       `toml:"volume"`
	HPA          HPAConfig          `toml:"hpa"`
	VPA          VPAConfig          `toml:"vpa"`
	PDB          PDBConfig          `toml:"pdb"`
	Multicluster MulticlusterConfig `toml:"multicluster"`
	Path         string             `toml:"-"`
}

type AppConfig struct {
	Name                  string   `toml:"name"`
	Port                  int      `toml:"port"`
	Dockerfile            string   `toml:"dockerfile"`
	CPU                   string   `toml:"cpu"`
	Memory                string   `toml:"memory"`
	Replicas              int      `toml:"replicas"`
	Domain                string   `toml:"domain"`
	Zone                  string   `toml:"zone"`
	Organization          string   `toml:"organization"`
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

type PDBConfig struct {
	Enabled        bool   `toml:"enabled"`
	Type           string `toml:"type"`
	MinAvailable   int32  `toml:"min_available"`
	MaxUnavailable int32  `toml:"max_unavailable"`
	Percent        int32  `toml:"percent"`
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
	return &cfg, nil
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
