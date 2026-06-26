package main

import (
	"testing"
)

func TestCreateCommand(t *testing.T) {
	cmd := createCommand()
	if cmd.Name != "1ctl" {
		t.Errorf("Expected command name '1ctl', got %s", cmd.Name)
	}
}
