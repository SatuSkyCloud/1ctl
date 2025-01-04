package commands

import (
	"testing"
)

func TestEnvironmentCommand(t *testing.T) {
	cmd := EnvironmentCommand()

	// Test command structure
	if cmd.Name != "env" {
		t.Errorf("Expected command name 'env', got %s", cmd.Name)
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

// func TestHandleCreateEnvironment(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		flags   map[string]string
// 		envVars []string
// 		wantErr bool
// 	}{
// 		{
// 			name: "valid environment",
// 			flags: map[string]string{
// 				"deployment-id": "3a842a45-aa4e-4e22-bdde-6f952c3cca43",
// 				"name":          "test-env",
// 			},
// 			envVars: []string{"DB_HOST=localhost", "DB_PORT=5432"},
// 			wantErr: false,
// 		},
// 		{
// 			name: "missing deployment-id",
// 			flags: map[string]string{
// 				"name": "test-env",
// 			},
// 			envVars: []string{"DB_HOST=localhost"},
// 			wantErr: true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			app := cli.NewApp()

// 			flags := flag.NewFlagSet("test", flag.ContinueOnError)
// 			for name, value := range tt.flags {
// 				flags.String(name, value, "test flag")
// 			}
// 			flags.String("env", "", "environment variables")
// 			ctx := cli.NewContext(app, flags, nil)

// 			// Set flags
// 			for name, value := range tt.flags {
// 				ctx.Set(name, value)
// 			}

// 			// Set environment variables
// 			for _, env := range tt.envVars {
// 				ctx.Set("env", env)
// 			}

// 			err := handleCreateEnvironment(ctx)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("handleCreateEnvironment() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }
