package commands

import (
	"1ctl/internal/testing/helpers"
	"flag"
	"os"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestAuthCommand(t *testing.T) {
	cmd := AuthCommand()

	// Test command structure
	if cmd.Name != "auth" {
		t.Errorf("Expected command name 'auth', got %s", cmd.Name)
	}

	// Check subcommands
	expectedSubcommands := map[string]bool{
		"login":  false,
		"logout": false,
		"status": false,
	}

	for _, subcmd := range cmd.Subcommands {
		if _, exists := expectedSubcommands[subcmd.Name]; !exists {
			t.Errorf("Unexpected subcommand: %s", subcmd.Name)
		}
		expectedSubcommands[subcmd.Name] = true
	}

	for name, found := range expectedSubcommands {
		if !found {
			t.Errorf("Missing subcommand: %s", name)
		}
	}
}

func TestHandleLogin(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		// {
		// 	name:    "valid token",
		// 	token:   "test-token",
		// 	wantErr: false,
		// },
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up CLI context
			app := cli.NewApp()
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			flags.String("token", tt.token, "test token flag")
			ctx := cli.NewContext(app, flags, nil)
			if tt.token != "" {
				if err := ctx.Set("token", tt.token); err != nil {
					t.Fatalf("failed to set token flag: %v", err)
				}
			}

			err := handleLogin(ctx)
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
			err := handleLogout(cli.NewContext(nil, nil, nil))
			if (err != nil) != tt.wantErr {
				t.Errorf("handleLogout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandleStatus(t *testing.T) {
	tests := []struct {
		name       string
		wantErr    bool
		wantOutput string
	}{
		// {
		// 	name:       "authenticated status",
		// 	wantErr:    false,
		// 	wantOutput: "Authenticated as test@example.com in organization test-org",
		// },
		// {
		// 	name:       "authenticated status",
		// 	wantErr:    false,
		// 	wantOutput: "Authenticated as test@example.com in organization test-org",
		// },
		{
			name:       "not authenticated status",
			wantErr:    true,
			wantOutput: "Not authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Run test
			err := handleAuthStatus(cli.NewContext(nil, nil, nil))

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("handleStatus() error = %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}
