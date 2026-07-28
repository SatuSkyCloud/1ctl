package postgres

import (
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

func TestPostgresUsersOnlySupportsList(t *testing.T) {
	users := postgresUsersCommand()
	if len(users.Commands) != 1 || users.Commands[0].Name != "list" {
		t.Fatalf("postgres users commands = %v, want only list", commandNames(users.Commands))
	}
}

func commandNames(commands []*cli.Command) []string {
	names := make([]string, 0, len(commands))
	for _, command := range commands {
		names = append(names, command.Name)
	}
	return names
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
