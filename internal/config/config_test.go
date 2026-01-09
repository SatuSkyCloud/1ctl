package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"1ctl/internal/context"
)

func TestGetConfig(t *testing.T) {
	cfg := GetConfig()

	if cfg.ApiURL != defaultAPIURL {
		t.Errorf("GetConfig().ApiURL = %v, want %v", cfg.ApiURL, defaultAPIURL)
	}
}

func TestValidateEnvironment(t *testing.T) {
	// Skip on Windows CI - context file operations may have permission issues
	if runtime.GOOS == "windows" {
		t.Skip("Skipping context-dependent test on Windows")
	}

	// Set up isolated test context directory
	tempDir := t.TempDir()
	testConfigDir := filepath.Join(tempDir, ".satusky")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create test config dir: %v", err)
	}

	// Create empty context file with valid JSON
	contextFile := filepath.Join(testConfigDir, "context.json")
	if err := os.WriteFile(contextFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create context file: %v", err)
	}

	// Override HOME to use test directory
	t.Setenv("HOME", tempDir)

	// Override context package's configDir
	context.SetConfigDir(testConfigDir)

	tests := []struct {
		name    string
		setup   func(t *testing.T)
		wantErr bool
	}{
		{
			name: "valid token",
			setup: func(t *testing.T) {
				if err := context.SetToken("test-token"); err != nil {
					t.Fatalf("failed to set token: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "missing token",
			setup: func(t *testing.T) {
				if err := context.SetToken(""); err != nil {
					t.Fatalf("failed to set token: %v", err)
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)
			err := ValidateEnvironment()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEnvironment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
