package packagecmd

import (
	"context"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestCreateCommandExposesChartFlag(t *testing.T) {
	for _, flag := range createCommand().Flags {
		if flag.Names()[0] == flagChart {
			return
		}
	}
	t.Fatal("package create does not expose --chart")
}

func TestDeleteCommandHasExactReleaseIDArityAndYesFlag(t *testing.T) {
	command := deleteCommand()
	if command.ArgsUsage != "<release-id>" {
		t.Fatalf("delete ArgsUsage = %q, want <release-id>", command.ArgsUsage)
	}
	if !hasPackageFlag(command, flagYes) {
		t.Fatal("package delete does not expose --yes")
	}

	for _, args := range [][]string{
		{"package", "delete"},
		{"package", "delete", "first", "second", "--yes"},
	} {
		err := Command().Run(context.Background(), args)
		if err == nil || !strings.Contains(err.Error(), "requires exactly one release-id") {
			t.Fatalf("Run(%v) error = %v, want exact release-id arity error", args, err)
		}
	}
}

func hasPackageFlag(command *cli.Command, name string) bool {
	for _, flag := range command.Flags {
		if flag.Names()[0] == name {
			return true
		}
	}
	return false
}
