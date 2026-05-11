package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"1ctl/internal/context"
)

func TestGetConfig(t *testing.T) {
	// Isolate from the real ~/.satusky so the active profile's api_url doesn't leak in.
	tempDir := t.TempDir()
	context.SetDefault(context.NewTestStore(filepath.Join(tempDir, ".satusky")))

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

	// Set up isolated test context directory with a "test" profile
	tempDir := t.TempDir()
	testConfigDir := filepath.Join(tempDir, ".satusky")
	profilesDir := filepath.Join(testConfigDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("Failed to create test profiles dir: %v", err)
	}

	// Create an empty "test" profile
	if err := os.WriteFile(filepath.Join(profilesDir, "test.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create test profile: %v", err)
	}

	// context.json points at the test profile
	contextFile := filepath.Join(testConfigDir, "context.json")
	if err := os.WriteFile(contextFile, []byte(`{"active_profile":"test"}`), 0600); err != nil {
		t.Fatalf("Failed to create context file: %v", err)
	}

	// Override HOME to use test directory
	t.Setenv("HOME", tempDir)

	// Override the default context Store so it operates on the test dir
	context.SetDefault(context.NewTestStore(testConfigDir))

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
