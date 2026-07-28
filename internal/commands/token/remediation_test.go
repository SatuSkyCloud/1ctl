package token

import (
	"context"
	"strings"
	"testing"
)

func TestTokenCommandsRejectMissingRequiredInput(t *testing.T) {
	tests := []struct {
		name    string
		command interface {
			Run(context.Context, []string) error
		}
		message string
	}{
		{name: "create", command: tokenCreateCommand(), message: "token name is required"},
		{name: "get", command: tokenGetCommand(), message: "token ID is required"},
		{name: "enable", command: tokenEnableCommand(), message: "token ID is required"},
		{name: "disable", command: tokenDisableCommand(), message: "token ID is required"},
		{name: "delete", command: tokenDeleteCommand(), message: "token ID is required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.command.Run(context.Background(), []string{test.name})
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error = %v, want %q", err, test.message)
			}
		})
	}
}
