package commands

import (
	"testing"

	"github.com/urfave/cli/v2"
)

func TestOrgCommand(t *testing.T) {
	cmd := OrgCommand()

	// Test command structure
	if cmd.Name != "org" {
		t.Errorf("Expected command name 'org', got %s", cmd.Name)
	}

	// Check aliases
	expectedAliases := []string{"organization"}
	if len(cmd.Aliases) != len(expectedAliases) {
		t.Errorf("Expected %d aliases, got %d", len(expectedAliases), len(cmd.Aliases))
	}
	for i, alias := range expectedAliases {
		if i >= len(cmd.Aliases) || cmd.Aliases[i] != alias {
			t.Errorf("Expected alias %s at position %d", alias, i)
		}
	}

	// Check subcommands
	expectedSubcommands := map[string]bool{
		"current": false,
		"switch":  false,
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

func TestOrgSwitchCommand(t *testing.T) {
	cmd := OrgCommand()

	// Find the switch subcommand
	var switchCmd *cli.Command
	for _, subcmd := range cmd.Subcommands {
		if subcmd.Name == "switch" {
			switchCmd = subcmd
			break
		}
	}

	if switchCmd == nil {
		t.Fatal("switch subcommand not found")
	}

	// Check flags
	expectedFlags := map[string]bool{
		"org-id":   false,
		"org-name": false,
	}

	for _, flag := range switchCmd.Flags {
		flagName := flag.Names()[0]
		if _, exists := expectedFlags[flagName]; exists {
			expectedFlags[flagName] = true
		}
	}

	for name, found := range expectedFlags {
		if !found {
			t.Errorf("Missing flag: %s", name)
		}
	}
}

func TestOrgCurrentCommand(t *testing.T) {
	cmd := OrgCommand()

	// Find the current subcommand
	var currentCmd *cli.Command
	for _, subcmd := range cmd.Subcommands {
		if subcmd.Name == "current" {
			currentCmd = subcmd
			break
		}
	}

	if currentCmd == nil {
		t.Fatal("current subcommand not found")
	}

	// Verify it has an action
	if currentCmd.Action == nil {
		t.Error("current subcommand should have an action")
	}
}
