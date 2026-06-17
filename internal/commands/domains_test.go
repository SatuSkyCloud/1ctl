package commands

import (
	"testing"

	"github.com/urfave/cli/v3"
)

func TestDomainsCommandStructure(t *testing.T) {
	cmd := DomainsCommand()
	if cmd.Name != "domains" {
		t.Errorf("Name = %q, want domains", cmd.Name)
	}
	if !containsString(cmd.Aliases, "domain") {
		t.Errorf("Aliases = %v, want to include 'domain'", cmd.Aliases)
	}
	wantSubs := []string{"list", "add", "delete", "check", "setup", "available", "search", "managed", "dns", "purchase", "purchase-status"}
	got := subNames(cmd.Commands)
	for _, w := range wantSubs {
		if !containsString(got, w) {
			t.Errorf("Subcommands missing %q (have %v)", w, got)
		}
	}

	add := findSubcommand(cmd, "add")
	if add == nil {
		t.Fatalf("add subcommand missing")
	}
	if !hasFlag(add, "with-www") {
		t.Errorf("add command missing with-www flag")
	}

	managed := findSubcommand(cmd, "managed")
	if managed == nil {
		t.Fatalf("managed subcommand missing")
	}
	for _, w := range []string{"list", "add", "verify", "delete"} {
		if findSubcommand(managed, w) == nil {
			t.Errorf("managed subcommands missing %q", w)
		}
	}

	dns := findSubcommand(cmd, "dns")
	if dns == nil {
		t.Fatalf("dns subcommand missing")
	}
	for _, w := range []string{"list", "create", "update", "delete"} {
		if findSubcommand(dns, w) == nil {
			t.Errorf("dns subcommands missing %q", w)
		}
	}
}

func TestNormalizeDomainArg(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "lowercase and trim", input: " App.Example.COM. ", want: "app.example.com"},
		{name: "reject url", input: "https://example.com", wantErr: true},
		{name: "reject wildcard", input: "*.example.com", wantErr: true},
		{name: "reject path", input: "example.com/path", wantErr: true},
		{name: "reject invalid", input: "not a domain", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeDomainArg(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeDomainArg() err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("normalizeDomainArg() = %q, want %q", got, tt.want)
			}
		})
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

func findSubcommand(cmd *cli.Command, name string) *cli.Command {
	for _, sub := range cmd.Commands {
		if sub.Name == name {
			return sub
		}
	}
	return nil
}

func hasFlag(cmd *cli.Command, name string) bool {
	for _, flag := range cmd.Flags {
		if flag.Names()[0] == name {
			return true
		}
	}
	return false
}
