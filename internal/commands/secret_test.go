package commands

import (
	"flag"
	"testing"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func TestSecretCommand(t *testing.T) {
	cmd := SecretCommand()

	// Test command structure
	if cmd.Name != "secret" {
		t.Errorf("Expected command name 'secret', got %s", cmd.Name)
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

func TestHandleCreateSecret(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		envVars []string
		wantErr bool
	}{
		{
			name: "valid secret",
			flags: map[string]string{
				"deployment-id": uuid.New().String(),
				"name":          "test-secret",
			},
			envVars: []string{"KEY1=value1", "KEY2=value2"},
			wantErr: false,
		},
		{
			name: "missing deployment-id",
			flags: map[string]string{
				"name": "test-secret",
			},
			envVars: []string{"KEY=value"},
			wantErr: true,
		},
		{
			name: "invalid env format",
			flags: map[string]string{
				"deployment-id": uuid.New().String(),
				"name":          "test-secret",
			},
			envVars: []string{"invalid-format"},
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
			flags.String("env", "", "test env vars")

			ctx := cli.NewContext(app, flags, nil)
			for name, value := range tt.flags {
				ctx.Set(name, value)
			}

			err := handleCreateSecret(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleCreateSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
