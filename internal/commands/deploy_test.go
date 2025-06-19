package commands

import (
	"flag"
	"testing"

	"1ctl/internal/docker"

	"github.com/urfave/cli/v2"
)

func TestDeployCommand(t *testing.T) {
	cmd := DeployCommand()

	// Test command structure
	if cmd.Name != "deploy" {
		t.Errorf("Expected command name 'deploy', got %s", cmd.Name)
	}

	// Check subcommands
	expectedSubcommands := map[string]bool{
		"list":   false,
		"get":    false,
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

func TestHandleDeploy(t *testing.T) {
	tests := []struct {
		name      string
		flags     map[string]string
		mockBuild func(opts docker.BuildOptions) error
		wantErr   bool
	}{
		{
			name: "valid deployment",
			flags: map[string]string{
				"cpu":        "1",
				"memory":     "512Mi",
				"project":    "test-project",
				"dockerfile": "testdata/Dockerfile",
			},
			mockBuild: func(opts docker.BuildOptions) error {
				return nil
			},
			wantErr: true,
		},
		{
			name: "invalid cpu",
			flags: map[string]string{
				"cpu":        "invalid",
				"memory":     "512Mi",
				"project":    "test-project",
				"dockerfile": "testdata/Dockerfile",
			},
			mockBuild: nil,
			wantErr:   true,
		},
		{
			name: "invalid memory",
			flags: map[string]string{
				"cpu":        "1",
				"memory":     "invalid",
				"project":    "test-project",
				"dockerfile": "testdata/Dockerfile",
			},
			mockBuild: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.NewApp()
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			for name, value := range tt.flags {
				flags.String(name, value, "test flag")
			}
			ctx := cli.NewContext(app, flags, nil)
			for name, value := range tt.flags {
				ctx.Set(name, value)
			}

			err := handleDeploy(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleDeploy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
