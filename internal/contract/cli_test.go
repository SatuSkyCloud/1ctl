package contract

import (
	"testing"

	"1ctl/internal/cliapp"
)

func TestCLIIncludesCurrentHighRiskCommands(t *testing.T) {
	manifest, err := CLI(cliapp.New())
	if err != nil {
		t.Fatal(err)
	}

	commands := make(map[string]CommandContract, len(manifest.Commands))
	for _, command := range manifest.Commands {
		commands[joinPath(command.Path)] = command
	}
	for _, path := range []string{"profile create", "logs stream", "nats create", "volumes delete"} {
		if _, ok := commands[path]; !ok {
			t.Errorf("manifest is missing %q", path)
		}
	}

	profileCreate := commands["profile create"]
	if profileCreate.MinArgs != 1 || profileCreate.MaxArgs == nil || *profileCreate.MaxArgs != 1 {
		t.Errorf("profile create arity = %d..%v, want 1..1", profileCreate.MinArgs, profileCreate.MaxArgs)
	}
	if hasFlag(profileCreate, "name") {
		t.Error("profile create unexpectedly exports obsolete --name flag")
	}
	if !hasFlag(profileCreate, "url") {
		t.Error("profile create is missing --url")
	}
	if hasFlag(commands["logs"], "follow") {
		t.Error("logs unexpectedly exports obsolete --follow flag")
	}
	if !hasFlag(commands[""], "version") {
		t.Error("root command is missing built-in --version")
	}
	if !hasFlag(commands["profile create"], "help") {
		t.Error("subcommand is missing built-in --help")
	}
	secretCreate := commands["secret create"]
	if secretCreate.MinArgs != 0 || secretCreate.MaxArgs != nil {
		t.Errorf("secret create arity = %d..%v, want 0..unbounded", secretCreate.MinArgs, secretCreate.MaxArgs)
	}
	for _, path := range []string{"service", "service list", "ingress", "ingress delete"} {
		if !commands[path].Hidden {
			t.Errorf("%q should be hidden because it or its parent is hidden", path)
		}
	}
}

func TestCLIContractDoesNotEmbedBuildVersion(t *testing.T) {
	manifest, err := CLI(cliapp.New())
	if err != nil {
		t.Fatal(err)
	}
	if manifest.CLI.Version != "" {
		t.Fatalf("CLI version = %q, want empty deterministic contract identity", manifest.CLI.Version)
	}
}

func TestAllArgsUsageIsMachineReadable(t *testing.T) {
	if _, err := CLI(cliapp.New()); err != nil {
		t.Fatal(err)
	}
}

func joinPath(path []string) string {
	result := ""
	for i, value := range path {
		if i > 0 {
			result += " "
		}
		result += value
	}
	return result
}

func hasFlag(command CommandContract, name string) bool {
	for _, flag := range command.Flags {
		if flag.Name == name {
			return true
		}
	}
	return false
}
