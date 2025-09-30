package commands

import (
	"flag"
	"testing"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func TestIngressCommand(t *testing.T) {
	cmd := IngressCommand()

	// Test command structure
	if cmd.Name != "ingress" {
		t.Errorf("Expected command name 'ingress', got %s", cmd.Name)
	}

	// Check subcommands
	expectedSubcommands := map[string]bool{
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

func TestHandleCreateIngress(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		wantErr bool
	}{
		{
			name: "valid ingress",
			flags: map[string]string{
				"deployment-id": uuid.New().String(),
				"service-id":    uuid.New().String(),
				"domain":        "test.example.com",
				"custom-dns":    "true",
				"app-label":     "test-app",
				"namespace":     "test-namespace",
			},
			wantErr: false,
		},
		{
			name: "missing deployment-id",
			flags: map[string]string{
				"domain":     "test.example.com",
				"custom-dns": "true",
			},
			wantErr: true,
		},
		{
			name: "invalid domain",
			flags: map[string]string{
				"deployment-id": uuid.New().String(),
				"domain":        "invalid domain",
				"custom-dns":    "true",
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

			err := handleUpsertIngress(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleUpsertIngress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
