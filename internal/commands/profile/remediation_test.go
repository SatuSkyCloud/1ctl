package profile

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

func TestProfileCommandsRejectMissingName(t *testing.T) {
	tests := []struct {
		name    string
		command func() interface {
			Run(context.Context, []string) error
		}
	}{
		{name: "create", command: func() interface {
			Run(context.Context, []string) error
		} {
			return profileCreateCommand()
		}},
		{name: "use", command: func() interface {
			Run(context.Context, []string) error
		} {
			return profileUseCommand()
		}},
		{name: "delete", command: func() interface {
			Run(context.Context, []string) error
		} {
			return profileDeleteCommand()
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.command().Run(context.Background(), []string{test.name})
			if err == nil || !strings.Contains(err.Error(), "profile name is required") {
				t.Fatalf("error = %v, want profile name required error", err)
			}
		})
	}
}

func TestHandleProfileListJSONEmpty(t *testing.T) {
	setupProfileRemediationContext(t)
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })

	output := captureProfileStdout(t, func() error {
		return handleProfileList(context.Background())
	})
	var profiles []map[string]any
	if err := json.Unmarshal([]byte(output), &profiles); err != nil {
		t.Fatalf("profile list output is not JSON: %q: %v", output, err)
	}
	if len(profiles) != 0 {
		t.Fatalf("profiles = %#v, want empty array", profiles)
	}
}

func TestHandleProfileCurrentJSONEmpty(t *testing.T) {
	setupProfileRemediationContext(t)
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })

	output := captureProfileStdout(t, func() error {
		return handleProfileCurrent(context.Background())
	})
	var current any
	if err := json.Unmarshal([]byte(output), &current); err != nil {
		t.Fatalf("profile current output is not JSON: %q: %v", output, err)
	}
	if current != nil {
		t.Fatalf("current profile = %#v, want null", current)
	}
}

func TestHandleProfileCurrentJSONDoesNotExposeCredentials(t *testing.T) {
	setupProfileRemediationContext(t)
	if err := satuskyctx.CreateProfile("work", "https://api.example.test"); err != nil {
		t.Fatal(err)
	}
	if err := satuskyctx.UseProfile("work"); err != nil {
		t.Fatal(err)
	}
	const token = "secret-token-must-not-appear"
	if err := satuskyctx.SetToken(token); err != nil {
		t.Fatal(err)
	}
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })

	output := captureProfileStdout(t, func() error {
		return handleProfileCurrent(context.Background())
	})
	if strings.Contains(output, token) {
		t.Fatalf("profile current output exposed authentication token: %q", output)
	}
	var current map[string]any
	if err := json.Unmarshal([]byte(output), &current); err != nil {
		t.Fatalf("profile current output is not JSON: %q: %v", output, err)
	}
	if current["profile"] != "work" {
		t.Fatalf("profile = %#v, want work", current["profile"])
	}
	if _, exists := current["token"]; exists {
		t.Fatalf("profile current output contains token field: %#v", current)
	}
}

func setupProfileRemediationContext(t *testing.T) {
	t.Helper()
	configDir := filepath.Join(t.TempDir(), ".satusky")
	if err := os.MkdirAll(filepath.Join(configDir, "profiles"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "context.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	originalStore := satuskyctx.Default()
	satuskyctx.SetDefault(satuskyctx.NewTestStore(configDir))
	t.Cleanup(func() { satuskyctx.SetDefault(originalStore) })
}

func captureProfileStdout(t *testing.T, run func() error) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	runErr := run()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = original
	if runErr != nil {
		t.Fatal(runErr)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
