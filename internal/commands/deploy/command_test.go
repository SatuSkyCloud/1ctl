package deploy

import (
	"reflect"
	"testing"

	"github.com/urfave/cli/v3"
)

// TestFlagsHaveDestination ensures every Required flag has a Destination pointer.
func TestFlagsHaveDestination(t *testing.T) {
	walkCommands(Command(), func(cmd *cli.Command) {
		for _, f := range cmd.Flags {
			if !isRequired(f) {
				continue
			}
			if hasNilDestination(f) {
				t.Errorf("command %q: required flag %q has no Destination", cmd.Name, getFlagName(f))
			}
		}
	})
}

func TestDeployCommand_Subcommands(t *testing.T) {
	cmd := Command()
	if cmd.Name != "deploy" {
		t.Errorf("expected name 'deploy', got %s", cmd.Name)
	}

	// Deploy should no longer have lifecycle subcommands — those moved to '1ctl app'
	if len(cmd.Commands) != 0 {
		t.Errorf("expected no subcommands on deploy, got %d: %v", len(cmd.Commands), cmd.Commands)
	}
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
