package domains

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// TestFlagsHaveDestination ensures every Required flag in the domains
// command tree has a Destination pointer.
func TestFlagsHaveDestination(t *testing.T) {
	walkCommands(Command(), func(cmd *cli.Command) {
		for _, f := range cmd.Flags {
			if !isRequired(f) {
				continue
			}
			if hasNilDestination(f) {
				t.Errorf("command %q: required flag %q has no Destination — value will be lost", cmd.Name, flagNameFrom(f))
			}
		}
	})
}

func TestFiniteDomainCommandsRejectExtraArgs(t *testing.T) {
	paths := [][]string{
		{"list"},
		{"add", "example.com"},
		{"delete", "example.com"},
		{"check", "example.com"},
		{"setup", "example.com"},
		{"available"},
		{"search"},
		{"managed", "list"},
		{"managed", "add", "example.com"},
		{"managed", "verify", "example.com"},
		{"managed", "delete", "example.com"},
		{"dns", "list"},
		{"dns", "create"},
		{"dns", "update"},
		{"dns", "delete"},
		{"purchase"},
		{"purchase-status", "intent-id"},
	}

	for _, path := range paths {
		name := strings.Join(path, " ")
		t.Run(name, func(t *testing.T) {
			args := append([]string{"domains"}, path...)
			args = append(args, "unexpected")
			err := Command().Run(context.Background(), args)
			if err == nil {
				t.Fatal("Run() error = nil, want maximum arity error")
			}
			if !strings.Contains(err.Error(), "accepts at most") {
				t.Fatalf("Run() error = %q, want maximum arity error", err)
			}
		})
	}
}

func TestFiniteMaxArgs(t *testing.T) {
	tests := []struct {
		usage  string
		want   int
		finite bool
	}{
		{usage: "", want: 0, finite: true},
		{usage: "<domain>", want: 1, finite: true},
		{usage: "<first> [second]", want: 2, finite: true},
		{usage: "[key=value...]", finite: false},
	}
	for _, tt := range tests {
		got, finite := finiteMaxArgs(tt.usage)
		if got != tt.want || finite != tt.finite {
			t.Errorf("finiteMaxArgs(%q) = (%d, %t), want (%d, %t)", tt.usage, got, finite, tt.want, tt.finite)
		}
	}
}

func TestFiniteMaxArgsPreservesExistingBefore(t *testing.T) {
	called := false
	cmd := &cli.Command{
		Name: "leaf",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			called = true
			return ctx, nil
		},
		Action: func(context.Context, *cli.Command) error { return nil },
	}
	enforceFiniteMaxArgs(cmd)

	if err := cmd.Run(context.Background(), []string{"leaf"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("existing Before hook was not called")
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

func flagNameFrom(f cli.Flag) string {
	return reflect.ValueOf(f).Elem().FieldByName("Name").String()
}
