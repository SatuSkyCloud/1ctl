package commands

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
)

// TestCaptureUserSetFlags_BeforeAndAfterCSet locks down the invariant that
// applyConfigScalar's c.Set call should NOT make the captured snapshot
// report a user-set flag. This regression existed in pre-review code: a
// satusky.toml carrying rolling_max_surge would flip
// opts.RollingFlagsExplicit and force strategy config onto requests that
// would otherwise have been omitted. The fix moved the IsSet capture to
// the top of handleDeploy. Test guards that.
func TestCaptureUserSetFlags_NotPoisonedByCSet(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("rolling-max-surge", "25%", "")
	// Do NOT call fs.Set / c.Set for "rolling-max-surge" — simulating the
	// "user did not pass --rolling-max-surge" case.
	ctx := cli.NewContext(nil, fs, nil)

	snapshot := captureUserSetFlags(ctx, "rolling-max-surge")
	if snapshot["rolling-max-surge"] {
		t.Fatalf("snapshot pre-c.Set: want false, got true")
	}

	// Simulate applyConfigScalar's effect.
	if err := ctx.Set("rolling-max-surge", "50%"); err != nil {
		t.Fatalf("ctx.Set: %v", err)
	}

	// The snapshot must still report user did not set it. c.IsSet would
	// return true here — that's exactly the trap the snapshot avoids.
	if snapshot["rolling-max-surge"] {
		t.Errorf("snapshot post-c.Set mutated: want false, got true")
	}
	if !ctx.IsSet("rolling-max-surge") {
		t.Log("note: c.IsSet returns true after c.Set — this is the trap captureUserSetFlags exists to side-step")
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
		"destroy":  false,
		"restart":  false,
		"releases": false,
		"rollback": false,
		"open":     false, // #3 D-02
		"scale":    false, // #3 F-05
	}

	for _, subcmd := range cmd.Subcommands {
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

func TestHandleDeploy(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		wantErr bool
	}{
		{
			name: "valid deployment",
			flags: map[string]string{
				"cpu":        "1",
				"memory":     "512Mi",
				"project":    "test-project",
				"dockerfile": "testdata/Dockerfile",
			},
			wantErr: true,
		},
		{
			name: "invalid cpu",
			flags: map[string]string{
				"cpu":        "invalid",
				"memory":     "512Mi",
				"project":    "test-project",
				"dockerfile": "testdata/Dockerfile",
			},
			wantErr: true,
		},
		{
			name: "invalid memory",
			flags: map[string]string{
				"cpu":        "1",
				"memory":     "invalid",
				"project":    "test-project",
				"dockerfile": "testdata/Dockerfile",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.NewApp()
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			for name, value := range tt.flags {
				flags.String(name, value, "test flag")
			}
			ctx := cli.NewContext(app, flags, nil)
			for name, value := range tt.flags {
				if err := ctx.Set(name, value); err != nil {
					t.Fatalf("failed to set flag %s: %v", name, err)
				}
			}

			err := handleDeploy(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleDeploy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateInputs_MulticlusterCustomDomain ensures that combining
// --multicluster with a custom (non-*.satusky.com) --domain is rejected
// at the client side, before any backend round trip. This guards the
// known operator limitation: zone-specific ingress classes are blocked
// from multi-cluster replication so the user would silently get a
// single-cluster deployment with broken HA expectations.
func TestValidateInputs_MulticlusterCustomDomain(t *testing.T) {
	tests := []struct {
		name         string
		multicluster bool
		domain       string
		wantErr      bool
	}{
		{"multicluster + custom domain", true, "app.example.com", true},
		{"multicluster + custom apex", true, "example.com", true},
		{"multicluster + custom wildcard", true, "*.example.com", true},
		{"multicluster + managed subdomain", true, "myapp.satusky.com", false},
		{"multicluster + managed wildcard", true, "*.satusky.com", false},
		{"multicluster + managed apex", true, "satusky.com", false},
		{"multicluster + no domain", true, "", false},
		{"no multicluster + custom domain", false, "app.example.com", false},
		{"no multicluster + managed domain", false, "myapp.satusky.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.NewApp()
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			// Required flags so we don't trip the earlier validations.
			flags.String("cpu", "1", "")
			flags.String("memory", "512Mi", "")
			flags.String("image", "registry.example.com/myapp:latest", "")
			flags.Bool("multicluster", tt.multicluster, "")
			if tt.domain != "" {
				flags.String("domain", tt.domain, "")
			}

			ctx := cli.NewContext(app, flags, nil)
			if err := ctx.Set("cpu", "1"); err != nil {
				t.Fatalf("set cpu: %v", err)
			}
			if err := ctx.Set("memory", "512Mi"); err != nil {
				t.Fatalf("set memory: %v", err)
			}
			if err := ctx.Set("image", "registry.example.com/myapp:latest"); err != nil {
				t.Fatalf("set image: %v", err)
			}
			if tt.multicluster {
				if err := ctx.Set("multicluster", "true"); err != nil {
					t.Fatalf("set multicluster: %v", err)
				}
			}
			if tt.domain != "" {
				if err := ctx.Set("domain", tt.domain); err != nil {
					t.Fatalf("set domain: %v", err)
				}
			}

			err := validateInputs(ctx)
			if tt.wantErr && err == nil {
				t.Errorf("validateInputs() expected error for multicluster=%v domain=%q, got nil", tt.multicluster, tt.domain)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateInputs() unexpected error for multicluster=%v domain=%q: %v", tt.multicluster, tt.domain, err)
			}
		})
	}
}

func TestShouldWaitForPlatformDNS(t *testing.T) {
	tests := []struct {
		domain string
		want   bool
	}{
		{"angrypanda-k28aur1.satusky.com", true},
		{"Satusky.com", true},
		{"*.satusky.com", true},
		{"example.com", false},
		{"app.example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			if got := shouldWaitForPlatformDNS(tt.domain); got != tt.want {
				t.Fatalf("shouldWaitForPlatformDNS(%q) = %v, want %v", tt.domain, got, tt.want)
			}
		})
	}
}
