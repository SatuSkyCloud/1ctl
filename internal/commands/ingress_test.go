package commands

import (
	"testing"
)

func TestIngressCommand(t *testing.T) {
	cmd := IngressCommand()

	// Test command structure
	if cmd.Name != "ingress" {
		t.Errorf("Expected command name 'ingress', got %s", cmd.Name)
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

func TestHandleUpsertIngress_NoActionDefined(t *testing.T) {
	// Verify the IngressCommand has the proper structure
	cmd := IngressCommand()
	if cmd == nil {
		t.Fatal("IngressCommand returned nil")
	}
}
