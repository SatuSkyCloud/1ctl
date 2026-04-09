package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const DefaultConfigFile = "satusky.toml"

type ProjectConfig struct {
	App  AppConfig `toml:"app"`
	Path string    `toml:"-"`
}

type AppConfig struct {
	Name         string `toml:"name"`
	Org          string `toml:"org"`
	Port         int    `toml:"port"`
	Dockerfile   string `toml:"dockerfile"`
	CPU          string `toml:"cpu"`
	Memory       string `toml:"memory"`
	Replicas     int    `toml:"replicas"`
	Domain       string `toml:"domain"`
	DeploymentID string `toml:"deployment_id"`
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

// ResolveDeploymentID returns the deployment ID to use for a command.
// Precedence: explicit --deployment-id flag > config file.
func ResolveDeploymentID(flagValue string, configArg string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	cfg, err := LoadConfig(configArg)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no --deployment-id provided and no satusky.toml found\nRun '1ctl init' to create one")
		}
		return "", err
	}
	if cfg.App.DeploymentID == "" {
		configName := "satusky.toml"
		if configArg != "" && !strings.HasSuffix(configArg, ".toml") {
			configName = fmt.Sprintf("satusky.%s.toml", configArg)
		} else if configArg != "" {
			configName = configArg
		}
		suffix := ""
		if configArg != "" {
			suffix = " --config " + configArg
		}
		return "", fmt.Errorf("no deployment_id in %s\nRun '1ctl deploy%s' first", configName, suffix)
	}
	return cfg.App.DeploymentID, nil
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
