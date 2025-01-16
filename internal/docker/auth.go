package docker

import (
	"1ctl/internal/utils"
	"encoding/base64"
	"os"
	"path/filepath"
)

// These will be set during compilation
var (
	// DockerConfigBase64 is the base64 encoded docker config.json
	DockerConfigBase64 = ""
)

// ensureDockerLogin ensures Docker is logged into the registry
func ensureDockerLogin() error {
	if DockerConfigBase64 == "" {
		return utils.NewError("docker configuration not found in binary", nil)
	}

	// Decode the base64 config
	configJSON, err := base64.StdEncoding.DecodeString(DockerConfigBase64)
	if err != nil {
		return utils.NewError("failed to decode docker config: %w", err)
	}

	// Create docker config directory if it doesn't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return utils.NewError("failed to get home directory: %w", err)
	}

	dockerDir := filepath.Join(homeDir, ".docker")
	if err := os.MkdirAll(dockerDir, 0700); err != nil {
		return utils.NewError("failed to create docker config directory: %w", err)
	}

	// Write the config file
	configPath := filepath.Join(dockerDir, "config.json")
	if err := os.WriteFile(configPath, configJSON, 0600); err != nil {
		return utils.NewError("failed to write docker config: %w", err)
	}

	return nil
}
