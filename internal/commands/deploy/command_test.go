package deploy

import (
	"context"
	"reflect"
	"testing"

	"1ctl/internal/config"
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

func TestDeployExplicitFalseAndDefaultFlagsOverrideConfig(t *testing.T) {
	var in DeployInput
	cmd := &cli.Command{
		Name:  "deploy",
		Flags: deployFlags(&in),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			captureDeploySetFlags(cmd, &in)
			return nil
		},
	}
	err := cmd.Run(context.Background(), []string{
		"deploy", "--image", "registry.example/api:v1",
		"--hpa=false", "--hpa-min-replicas", "1", "--hpa-max-replicas", "10",
		"--hpa-cpu-target", "80", "--hpa-memory-target", "0",
		"--vpa=false", "--vpa-mode", "Off",
		"--pdb=false", "--pdb-type", "auto",
		"--fast=false", "--multi-cluster=false",
	})
	if err != nil {
		t.Fatalf("parse deploy flags: %v", err)
	}

	for _, name := range []string{
		flagHPA, flagHPAMinReplicas, flagHPAMaxReplicas, flagHPACPUCoreTarget,
		flagHPAMemoryTarget, flagVPA, flagVPAMode, flagPDB, flagPDBType,
		flagFast, flagMulticluster,
	} {
		if !in.SetFlags[name] {
			t.Errorf("SetFlags[%q] = false after explicit CLI flag", name)
		}
	}

	cfg := &config.ProjectConfig{
		Build:        config.BuildConfig{FastBuild: true},
		HPA:          config.HPAConfig{Enabled: true, MinReplicas: 3, MaxReplicas: 20, CPUTarget: 60, MemoryTarget: 70},
		VPA:          config.VPAConfig{Enabled: true, Mode: "Auto"},
		PDB:          config.PDBConfig{Enabled: true, Type: "fixed", MinAvailable: 2},
		Multicluster: config.MulticlusterConfig{Enabled: true, Mode: "active-active"},
	}
	merged := mergeConfig(in, cfg)
	if merged.HPA || merged.VPA || merged.PDB || merged.Fast || merged.Multicluster {
		t.Fatalf("explicit false flags were overwritten: hpa=%v vpa=%v pdb=%v fast=%v multicluster=%v",
			merged.HPA, merged.VPA, merged.PDB, merged.Fast, merged.Multicluster)
	}
	if merged.HPAMinReplicas != 1 || merged.HPAMaxReplicas != 10 || merged.HPACPUCoreTarget != 80 || merged.HPAMemoryTarget != 0 {
		t.Fatalf("explicit HPA defaults were overwritten: min=%d max=%d cpu=%d memory=%d",
			merged.HPAMinReplicas, merged.HPAMaxReplicas, merged.HPACPUCoreTarget, merged.HPAMemoryTarget)
	}
	if merged.VPAMode != "Off" || merged.PDBType != "auto" {
		t.Fatalf("explicit VPA/PDB defaults were overwritten: mode=%q type=%q", merged.VPAMode, merged.PDBType)
	}

	opts, err := prepareDeploymentOptions(merged, cfg)
	if err != nil {
		t.Fatalf("prepareDeploymentOptions: %v", err)
	}
	if opts.HPAConfig != nil || opts.VPAConfig != nil || opts.PDBConfig != nil || opts.FastBuild || opts.MulticlusterEnabled {
		t.Fatalf("explicit false flags were re-enabled while preparing options: %+v", opts)
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

// Deleting an app deletes its volumes. This was the reverse: `1ctl app delete`
// removed the workload, the routes and the services, and left every PVC bound.
// Nothing in the confirmation named them, and once the deployment record was
// gone no CLI command could reach them -- the storage stayed billed and
// invisible, reclaimable only with kubectl.
func TestDestroyPurgesRetainedResourcesByDefault(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   DestroyInput
		want bool
	}{
		{"bare delete purges", DestroyInput{}, true},
		{"explicit purge purges", DestroyInput{PurgeRetained: true, PurgeExplicit: true}, true},
		{"retain-volumes opts out", DestroyInput{RetainVolumes: true}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.PurgeRetainedResources(); got != tt.want {
				t.Fatalf("PurgeRetainedResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --retain-volumes is the only way to keep them, so it must keep working
// regardless of whether --purge-retained is also present (that pair is
// rejected earlier as a contradiction).
func TestRetainVolumesAlwaysWinsOverTheDefault(t *testing.T) {
	in := DestroyInput{RetainVolumes: true, PurgeRetained: false}
	if in.PurgeRetainedResources() {
		t.Fatal("--retain-volumes must suppress the purge default")
	}
}
