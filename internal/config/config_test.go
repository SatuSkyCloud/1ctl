package config

import (
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
	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "valid token",
			setup: func() {
				_ = context.SetToken("test-token")
			},
			wantErr: false,
		},
		{
			name: "missing token",
			setup: func() {
				_ = context.SetToken("")
			},
			wantErr: true,
		},
	}

	// Save original token and restore after tests
	originalToken := context.GetToken()
	defer func() {
		_ = context.SetToken(originalToken)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := ValidateEnvironment()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEnvironment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
