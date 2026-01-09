package config

import (
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

	// Save original token and restore after tests
	originalToken := context.GetToken()
	defer func() {
		_ = context.SetToken(originalToken) //nolint:errcheck
	}()

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
