package machine

import (
	"context"
	"reflect"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestFlagsHaveDestination(t *testing.T) {
	walkCommands(Command(), func(cmd *cli.Command) {
		for _, f := range cmd.Flags {
			if !isRequired(f) {
				continue
			}
			if hasNilDestination(f) {
				t.Errorf("command %q: required flag %q has no Destination — value will be lost", cmd.Name, flagNameFromReflect(f))
			}
		}
	})
}

func walkCommands(cmd *cli.Command, fn func(*cli.Command)) {
	fn(cmd)
	for _, sub := range cmd.Commands {
		walkCommands(sub, fn)
	}
}

func isRequired(f cli.Flag) bool {
	return reflect.ValueOf(f).Elem().FieldByName("Required").Bool()
}

func hasNilDestination(f cli.Flag) bool {
	dest := reflect.ValueOf(f).Elem().FieldByName("Destination")
	if !dest.IsValid() {
		return true
	}
	return dest.IsNil()
}

func flagNameFromReflect(f cli.Flag) string {
	return reflect.ValueOf(f).Elem().FieldByName("Name").String()
}

func TestRequiredPositionalArgumentsReturnNonZeroExit(t *testing.T) {
	tests := [][]string{
		{"machine", "get"},
		{"machine", "update"},
		{"machine", "delete"},
		{"machine", "inspect"},
		{"machine", "logs"},
		{"machine", "events"},
		{"machine", "usage", "get"},
		{"machine", "usage", "cost"},
		{"machine", "labels", "list"},
		{"machine", "labels", "set", "machine-id"},
		{"machine", "labels", "unset", "machine-id"},
	}
	for _, args := range tests {
		err := Command().Run(context.Background(), args)
		assertNonZeroExit(t, args, err)
	}
}

func assertNonZeroExit(t *testing.T, args []string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%v returned nil error", args)
	}
}
