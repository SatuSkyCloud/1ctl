package commands

import (
	"flag"
	"testing"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func TestIssuerCommand(t *testing.T) {
	cmd := IssuerCommand()

	// Test command structure
	if cmd.Name != "issuer" {
		t.Errorf("Expected command name 'issuer', got %s", cmd.Name)
	}

	// Check subcommands
	expectedSubcommands := map[string]bool{
		"create": false,
		"list":   false,
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

func TestHandleCreateIssuer(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		wantErr bool
	}{
		{
			name: "valid issuer",
			flags: map[string]string{
				"deployment-id": uuid.New().String(),
				"email":         "test@example.com",
				"environment":   "staging",
			},
			wantErr: false,
		},
		{
			name: "missing deployment-id",
			flags: map[string]string{
				"email":       "test@example.com",
				"environment": "staging",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			flags: map[string]string{
				"deployment-id": uuid.New().String(),
				"email":         "invalid-email",
				"environment":   "staging",
			},
			wantErr: true,
		},
		{
			name: "invalid environment",
			flags: map[string]string{
				"deployment-id": uuid.New().String(),
				"email":         "test@example.com",
				"environment":   "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.NewApp()
			flags := flag.NewFlagSet("test", flag.ContinueOnError)

			// Set up flags
			for name, value := range tt.flags {
				flags.String(name, value, "test flag")
			}

			ctx := cli.NewContext(app, flags, nil)
			for name, value := range tt.flags {
				ctx.Set(name, value)
			}

			err := handleCreateIssuer(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleCreateIssuer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
