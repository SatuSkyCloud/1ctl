package commands

import (
	"testing"
)

func TestSecretCommand(t *testing.T) {
	cmd := SecretCommand()

	// Test command structure
	if cmd.Name != "secret" {
		t.Errorf("Expected command name 'secret', got %s", cmd.Name)
	}

	// Check subcommands
	expectedCommands := map[string]bool{
		"create": false,
		"list":   false,
		"get":    false,
		"unset":  false,
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

func TestSecretCommand_Structure(t *testing.T) {
	cmd := SecretCommand()
	if cmd == nil {
		t.Fatal("SecretCommand returned nil")
	}
}
