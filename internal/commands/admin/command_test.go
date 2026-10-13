package admin

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func TestDeploymentAdoptRequiresExactlyOnePositionalArgument(t *testing.T) {
	validFlags := []string{
		"--reason", "canary",
		"--expected-uid", uuid.NewString(),
		"--expected-resource-version", "12",
		"--expected-generation", "3",
		"--request-id", uuid.NewString(),
		"--yes",
	}
	for _, args := range [][]string{
		append([]string{"admin", "deployment", "adopt"}, validFlags...),
		append([]string{"admin", "deployment", "adopt", uuid.NewString(), uuid.NewString()}, validFlags...),
	} {
		err := Command().Run(context.Background(), args)
		if err == nil {
			t.Fatalf("%v returned nil error", args)
		}
	}
}

func findDeploymentCommand(t *testing.T, name string) *cli.Command {
	t.Helper()
	root := Command()
	if len(root.Commands) != 1 || root.Commands[0].Name != "deployment" {
		t.Fatalf("unexpected admin command tree")
	}
	deployment := root.Commands[0]
	for _, command := range deployment.Commands {
		if command.Name == name {
			return command
		}
	}
	t.Fatalf("deployment command %q not found", name)
	return nil
}

func replaceFlagValue(args []string, flag, value string) []string {
	for i := range args {
		if args[i] == flag && i+1 < len(args) {
			args[i+1] = value
			return args
		}
	}
	return args
}

func removeFlag(args []string, flag string) []string {
	for i := range args {
		if args[i] == flag && i+1 < len(args) {
			return append(args[:i], args[i+2:]...)
		}
	}
	return args
}

func removeBoolFlag(args []string, flag string) []string {
	for i := range args {
		if args[i] == flag {
			return append(args[:i], args[i+1:]...)
		}
	}
	return args
}
