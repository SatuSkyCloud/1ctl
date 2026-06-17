package commands

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"1ctl/internal/config"

	"github.com/urfave/cli/v3"
)

// TestCaptureUserSetFlags_NotPoisonedByCSet locks down the invariant that
// applyConfigScalar's c.Set call should NOT make the captured snapshot
// report a user-set flag.
func TestCaptureUserSetFlags_NotPoisonedByCSet(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "rolling-max-surge", Value: "25%"},
		},
	}

	snapshot := captureUserSetFlags(cmd, "rolling-max-surge")
	if snapshot["rolling-max-surge"] {
		t.Fatalf("snapshot pre-c.Set: want false, got true")
	}

	// Simulate applyConfigScalar's effect.
	if err := cmd.Set("rolling-max-surge", "50%"); err != nil {
		t.Fatalf("cmd.Set: %v", err)
	}

	// The snapshot must still report user did not set it.
	if snapshot["rolling-max-surge"] {
		t.Errorf("snapshot post-c.Set mutated: want false, got true")
	}
	if !cmd.IsSet("rolling-max-surge") {
		t.Log("note: cmd.IsSet returns true after cmd.Set — this is the trap captureUserSetFlags exists to side-step")
	}
}

func TestShouldShowDeployHelp_UsesFlagDefaults(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "cpu-request", Value: "250m"},
			&cli.StringFlag{Name: "memory", Value: "256Mi"},
			&cli.StringFlag{Name: "image", Value: ""},
		},
	}

	if shouldShowDeployHelp(cmd, &config.ProjectConfig{}) {
		t.Fatal("deploy help guard ignored cpu/memory flag defaults")
	}
}

func TestShouldShowDeployHelp_EmptyResourceDefaults(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "cpu-request", Value: ""},
			&cli.StringFlag{Name: "memory", Value: ""},
			&cli.StringFlag{Name: "image", Value: ""},
		},
	}

	if !shouldShowDeployHelp(cmd, &config.ProjectConfig{}) {
		t.Fatal("deploy help guard should show help when no image, cpu, or memory defaults exist")
	}
}

func TestDeployCommand(t *testing.T) {
	cmd := DeployCommand()

	// Test command structure
	if cmd.Name != "deploy" {
		t.Errorf("Expected command name 'deploy', got %s", cmd.Name)
	}

	// Check subcommands
	expectedSubcommands := map[string]bool{
		"list":     false,
		"get":      false,
		"status":   false,
		"delete":   false,
		"restart":  false,
		"releases": false,
		"rollback": false,
		"open":     false,
		"scale":    false,
	}

	for _, subcmd := range cmd.Commands {
		if _, exists := expectedSubcommands[subcmd.Name]; !exists {
			t.Errorf("Unexpected subcommand: %s", subcmd.Name)
		}
		expectedSubcommands[subcmd.Name] = true
	}

	for name, found := range expectedSubcommands {
		if !found {
			t.Errorf("Missing subcommand: %s", name)
		}
	}
}

func TestDeployCommand_Flags(t *testing.T) {
	cmd := DeployCommand()

	expectedFlags := []string{
		"name", "cpu", "cpu-request", "cpu-limit", "memory",
		"machine", "machine-tag", "domain", "organization",
		"health-path", "dockerfile", "image", "fast", "port",
		"env", "volume-size", "volume-mount", "volume-storage-class",
		"zone", "multicluster", "multicluster-mode", "backup-enabled",
		"backup-schedule", "backup-retention", "backup-priority-cluster",
		"replicas", "pdb", "pdb-type", "pdb-min-available", "pdb-percent",
		"hpa", "hpa-min-replicas", "hpa-max-replicas", "hpa-cpu-target",
		"hpa-memory-target", "vpa", "vpa-mode", "vpa-min-cpu", "vpa-max-cpu",
		"vpa-min-memory", "vpa-max-memory", "wait-for", "strategy",
		"rolling-max-surge", "rolling-max-unavailable", "config", "wait",
	}

	flagMap := make(map[string]bool)
	for _, f := range cmd.Flags {
		flagMap[f.Names()[0]] = true
	}

	for _, name := range expectedFlags {
		if !flagMap[name] {
			t.Errorf("Missing flag: --%s", name)
		}
	}
}

func TestCheckPublicURLSmokeAtPath_NonStrict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	result := checkPublicURLSmoke(server.URL, []string{"/"}, false)

	if !result.Ready {
		t.Errorf("Expected 401 to pass non-strict smoke check, got: %s", result.Reason)
	}
	if result.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", result.StatusCode)
	}
}

func TestCheckPublicURLSmokeAtPath_Strict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	result := checkPublicURLSmoke(server.URL, []string{"/"}, true)

	if result.Ready {
		t.Error("Expected 401 to fail strict smoke check")
	}
}
