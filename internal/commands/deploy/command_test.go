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

func TestAppDeleteAsyncFlags(t *testing.T) {
	var deleteCommand *cli.Command
	for _, command := range AppCommand().Commands {
		if command.Name == "delete" {
			deleteCommand = command
			break
		}
	}
	if deleteCommand == nil {
		t.Fatal("app delete command not found")
	}
	want := map[string]bool{"purge-retained": false, "retain-volumes": false, "no-wait": false}
	for _, flag := range deleteCommand.Flags {
		if name := getFlagName(flag); name == "purge-retained" || name == "retain-volumes" || name == "no-wait" {
			want[name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing --%s flag", name)
		}
	}
}

func TestValidateInputsRejectsInvalidDeployValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*mergedInput)
	}{
		{"environment", func(m *mergedInput) { m.Env = []string{"BROKEN"} }},
		{"surge", func(m *mergedInput) { m.RollingMaxSurge = "bogus" }},
		{"replicas", func(m *mergedInput) { m.Replicas = -1 }},
		{"VPA mode without VPA", func(m *mergedInput) { m.VPAMode = "Bogus" }},
		{"backup schedule without multi-cluster", func(m *mergedInput) { m.BackupSchedule = "never" }},
		{"PDB type", func(m *mergedInput) { m.PDB = true; m.PDBType = "bogus" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := mergedInput{DeployInput: DeployInput{
				Image: "nginxinc/nginx-unprivileged:1.29-alpine", RollingMaxSurge: "25%",
				RollingMaxUnavail: "25%", VPAMode: "Off", BackupSchedule: "daily", PDBType: "auto",
			}}
			test.mutate(&input)
			if err := validateInputs(input); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestParseEnvVarsPreservesEmptyValues(t *testing.T) {
	values, err := parseEnvVars([]string{"EMPTY=", "NORMAL=value"})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].Key != "EMPTY" || values[0].Value != "" {
		t.Fatalf("unexpected values: %#v", values)
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
