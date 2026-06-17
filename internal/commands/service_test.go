package commands

import (
	"testing"
)

func TestServiceCommand(t *testing.T) {
	cmd := ServiceCommand()

	// Test command structure
	if cmd.Name != "service" {
		t.Errorf("Expected command name 'service', got %s", cmd.Name)
	}

	// Check subcommands
	expectedCommands := map[string]bool{
		"list":   false,
		"delete": false,
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
