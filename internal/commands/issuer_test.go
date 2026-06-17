package commands

import (
	"testing"
)

func TestIssuerCommand(t *testing.T) {
	cmd := IssuerCommand()

	// Test command structure
	if cmd.Name != "issuer" {
		t.Errorf("Expected command name 'issuer', got %s", cmd.Name)
	}

	// Check subcommands
	expectedCommands := map[string]bool{
		"create": false,
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

func TestIssuerCommand_Structure(t *testing.T) {
	cmd := IssuerCommand()
	if cmd == nil {
		t.Fatal("IssuerCommand returned nil")
	}
}
