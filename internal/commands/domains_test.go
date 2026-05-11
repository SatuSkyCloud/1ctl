package commands

import (
	"testing"

	"github.com/urfave/cli/v2"
)

func TestDomainsCommandStructure(t *testing.T) {
	cmd := DomainsCommand()
	if cmd.Name != "domains" {
		t.Errorf("Name = %q, want domains", cmd.Name)
	}
	if !containsString(cmd.Aliases, "domain") {
		t.Errorf("Aliases = %v, want to include 'domain'", cmd.Aliases)
	}
	wantSubs := []string{"list", "add", "remove", "check"}
	got := subNames(cmd.Subcommands)
	for _, w := range wantSubs {
		if !containsString(got, w) {
			t.Errorf("Subcommands missing %q (have %v)", w, got)
		}
	}
}

func TestSurfaceReframe_HiddenCommands(t *testing.T) {
	// Issue #28 hide-only: service stays callable but invisible in --help.
	// Issue #30 reframe: ingress upsert and issuer same treatment.
	tests := []struct {
		name string
		cmd  *cli.Command
	}{
		{"service", ServiceCommand()},
		{"ingress", IngressCommand()},
		{"issuer", IssuerCommand()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.cmd.Hidden {
				t.Errorf("%s command should be Hidden:true after surface reframe", tt.name)
			}
		})
	}
}

func containsString(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func subNames(subs []*cli.Command) []string {
	names := make([]string, 0, len(subs))
	for _, s := range subs {
		names = append(names, s.Name)
	}
	return names
}
