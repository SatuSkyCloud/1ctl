package commands

import (
	"1ctl/internal/api"
	"testing"
)

func TestMachineCommand(t *testing.T) {
	cmd := MachineCommand()

	// Test command structure
	if cmd.Name != "machine" {
		t.Errorf("Expected command name 'machine', got %s", cmd.Name)
	}

	if cmd.Usage != "Check your machines and their status" {
		t.Errorf("Expected usage 'Check your machines and their status', got %s", cmd.Usage)
	}

	// Check subcommands
	expectedSubcommands := map[string]bool{
		"list": false,
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

	// Test list subcommand structure
	listCmd := cmd.Subcommands[0]
	if listCmd.Name != "list" {
		t.Errorf("Expected subcommand name 'list', got %s", listCmd.Name)
	}

	if listCmd.Usage != "List all machines owned by the current user" {
		t.Errorf("Expected usage 'List all machines owned by the current user', got %s", listCmd.Usage)
	}

	if listCmd.Action == nil {
		t.Error("List command action should not be nil")
	}
}

func TestPrintMachineDetails(t *testing.T) {
	// Create a sample machine for testing
	machine := &api.Machine{
		MachineName:   "test-machine",
		MachineTypes:  []string{"compute", "gpu"},
		MachineRegion: "us-west",
		MachineZone:   "us-west-1",
		IpAddr:        "192.168.1.1",
		CPUCores:      8,
		MemoryGB:      32,
		StorageGB:     500,
		GPUCount:      2,
		GPUType:       "NVIDIA A100",
		BandwidthGbps: 10,
		Brand:         "SuperMicro",
		Model:         "X11DPH-T",
		Manufacturer:  "SuperMicro",
		FormFactor:    "2U",
		Monetized:     true,
	}

	// This is primarily a visual test - we're just ensuring it doesn't panic
	printMachineDetails(machine)
}
