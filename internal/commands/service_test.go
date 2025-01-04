package commands

import (
	"flag"
	"testing"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func TestServiceCommand(t *testing.T) {
	cmd := ServiceCommand()

	// Test command structure
	if cmd.Name != "service" {
		t.Errorf("Expected command name 'service', got %s", cmd.Name)
	}

	// Check subcommands
	expectedSubcommands := map[string]bool{
		"create": false,
		"list":   false,
		"delete": false,
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

func TestHandleCreateService(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		wantErr bool
	}{
		{
			name: "valid service",
			flags: map[string]string{
				"deployment-id": uuid.New().String(),
				"name":          "test-service",
				"port":          "8080",
				"namespace":     "test-namespace",
			},
			wantErr: false,
		},
		{
			name: "missing deployment-id",
			flags: map[string]string{
				"name":      "test-service",
				"port":      "8080",
				"namespace": "test-namespace",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			flags: map[string]string{
				"deployment-id": uuid.New().String(),
				"name":          "test-service",
				"port":          "invalid",
				"namespace":     "test-namespace",
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

			err := handleCreateService(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleCreateService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
