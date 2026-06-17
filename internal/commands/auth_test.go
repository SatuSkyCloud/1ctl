package commands

import (
	"os"
	"path/filepath"
	"testing"

	"1ctl/internal/context"
	"1ctl/internal/testing/helpers"

	"github.com/urfave/cli/v3"
)

func TestAuthCommand(t *testing.T) {
	cmd := AuthCommand()

	// Test command structure
	if cmd.Name != "auth" {
		t.Errorf("Expected command name 'auth', got %s", cmd.Name)
	}

	// Check subcommands
	expectedCommands := map[string]bool{
		"login":  false,
		"logout": false,
		"status": false,
	}

	for _, subcmd := range cmd.Commands {
		if _, exists := expectedCommands[subcmd.Name]; !exists {
			t.Errorf("Unexpected command: %s", subcmd.Name)
		}
		expectedCommands[subcmd.Name] = true
	}

	for name, found := range expectedCommands {
		if !found {
			t.Errorf("Missing command: %s", name)
		}
	}
}

func TestHandleLogin(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "no token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Isolate from the real profile so the stored token doesn't leak in.
			dir := helpers.SetupTestContext(t)
			defer func() { _ = os.RemoveAll(dir) }() //nolint:errcheck
			context.SetDefault(context.NewTestStore(filepath.Join(dir, ".satusky")))

			// Set up CLI command with flags
			cmd := &cli.Command{
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "token"},
				},
			}
			if tt.token != "" {
				if err := cmd.Set("token", tt.token); err != nil {
					t.Fatalf("failed to set token flag: %v", err)
				}
			}

			err := handleLogin(t.Context(), cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleLogin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleLogout(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "successful logout",
			setup: func(t *testing.T) string {
				dir := helpers.SetupTestContext(t)
				helpers.CreateContextFile(t, dir, `{"token": "test-token"}`)
				return dir
			},
			wantErr: false,
		},
		{
			name: "logout with no existing token",
			setup: func(t *testing.T) string {
				return helpers.SetupTestContext(t)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			defer func() { _ = os.RemoveAll(dir) }() //nolint:errcheck
			// Point context package at the temp dir so writes go to the test profile
			context.SetDefault(context.NewTestStore(filepath.Join(dir, ".satusky")))
			err := handleLogout(t.Context(), &cli.Command{})
			if (err != nil) != tt.wantErr {
				t.Errorf("handleLogout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleAuthStatus(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "no token",
			setup: func(t *testing.T) string {
				return helpers.SetupTestContext(t)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			defer func() { _ = os.RemoveAll(dir) }() //nolint:errcheck
			context.SetDefault(context.NewTestStore(filepath.Join(dir, ".satusky")))
			err := handleAuthStatus(t.Context(), &cli.Command{})
			if (err != nil) != tt.wantErr {
				t.Errorf("handleAuthStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
