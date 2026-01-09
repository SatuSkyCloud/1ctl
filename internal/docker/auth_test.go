package docker

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"1ctl/internal/testing/helpers"
)

func TestEnsureDockerLogin(t *testing.T) {
	// Save original home dir and restore after test
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }() //nolint:errcheck

	tests := []struct {
		name           string
		configBase64   string
		wantErr        bool
		validateConfig bool
	}{
		{
			name: "valid config",
			configBase64: base64.StdEncoding.EncodeToString([]byte(`{
				"auths": {
					"registry.example.com": {
						"auth": "dGVzdDp0ZXN0"
					}
				}
			}`)),
			wantErr:        false,
			validateConfig: true,
		},
		{
			name:         "empty config",
			configBase64: "",
			wantErr:      true,
		},
		{
			name:         "invalid base64",
			configBase64: "invalid-base64",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp home directory
			homeDir := helpers.CreateTempDir(t)

			// Set HOME for Unix, USERPROFILE for Windows
			t.Setenv("HOME", homeDir)
			if runtime.GOOS == "windows" {
				t.Setenv("USERPROFILE", homeDir)
			}

			// Set test config
			originalConfig := DockerConfigBase64
			DockerConfigBase64 = tt.configBase64
			defer func() { DockerConfigBase64 = originalConfig }()

			err := ensureDockerLogin()
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureDockerLogin() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validateConfig {
				// Check if config file was created with correct permissions (skip on Windows)
				configPath := filepath.Join(homeDir, ".docker", "config.json")
				info, err := os.Stat(configPath)
				if err != nil {
					t.Errorf("Failed to stat config file: %v", err)
				} else if runtime.GOOS != "windows" {
					// Only check permissions on Unix - Windows doesn't support them
					if info.Mode().Perm() != 0600 {
						t.Errorf("Config file has wrong permissions: got %v, want %v", info.Mode().Perm(), 0600)
					}
				}
			}
		})
	}
}
