package integration

import (
	"1ctl/internal/context"
	"testing"
)

func TestAuthFlow(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   "test_token",
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid_token",
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context.SetToken("")

			// Test setting token
			err := context.SetToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Error("SetToken() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("SetToken() error = %v, want nil", err)
				return
			}

			// Verify token was set correctly
			savedToken := context.GetToken()
			if savedToken != tt.token {
				t.Errorf("GetToken() = %v, want %v", savedToken, tt.token)
			}
		})
	}
}
