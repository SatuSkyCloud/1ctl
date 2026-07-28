package volumes

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
				t.Errorf("command %q: required flag %q has no Destination", cmd.Name, flagName(f))
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

func flagName(f cli.Flag) string {
	return reflect.ValueOf(f).Elem().FieldByName("Name").String()
}

func TestRequiredPositionalArgumentsReturnNonZeroExit(t *testing.T) {
	for _, args := range [][]string{
		{"volumes", "detach"},
		{"volumes", "delete"},
	} {
		err := Command().Run(context.Background(), args)
		if err == nil {
			t.Fatalf("%v returned nil error", args)
		}
	}
}
