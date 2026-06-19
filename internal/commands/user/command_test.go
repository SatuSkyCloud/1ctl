package user

import (
	"reflect"
	"testing"

	"github.com/urfave/cli/v3"
)

// TestFlagsHaveDestination ensures every Required flag in the user
// command tree has a Destination pointer.  A Required flag without one
// demands user input but discards the value — the handler never sees it.
func TestFlagsHaveDestination(t *testing.T) {
	walkCommands(Command(), func(cmd *cli.Command) {
		for _, f := range cmd.Flags {
			if !isRequired(f) {
				continue
			}
			if hasNilDestination(f) {
				t.Errorf("command %q: required flag %q has no Destination — value will be lost", cmd.Name, getFlagName(f))
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

func getFlagName(f cli.Flag) string {
	return reflect.ValueOf(f).Elem().FieldByName("Name").String()
}
