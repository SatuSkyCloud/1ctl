package org

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

func TestOrgCommandsRejectMissingRequiredInput(t *testing.T) {
	tests := []struct {
		name    string
		command interface {
			Run(context.Context, []string) error
		}
		message string
	}{
		{name: "switch", command: orgSwitchCommand(), message: "organization name or ID is required"},
		{name: "create", command: orgCreateCommand(), message: "organization name is required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.command.Run(context.Background(), []string{test.name})
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error = %v, want %q", err, test.message)
			}
		})
	}
}

func TestHandleOrgCurrentJSON(t *testing.T) {
	setupOrgRemediationContext(t, "", "org-123", "Example Org")
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })

	output := captureOrgStdout(t, func() error {
		return handleOrgCurrent(context.Background())
	})
	var current struct {
		Organization   string `json:"organization"`
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal([]byte(output), &current); err != nil {
		t.Fatalf("org current output is not JSON: %q: %v", output, err)
	}
	if current.Organization != "Example Org" || current.OrganizationID != "org-123" {
		t.Fatalf("current organization = %#v", current)
	}
}

func TestHandleOrgTeamListJSONEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/cli/organizations/id/org-123/team" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"error":false,"data":[]}`)
	}))
	defer server.Close()

	setupOrgRemediationContext(t, server.URL, "org-123", "Example Org")
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })

	output := captureOrgStdout(t, func() error {
		return handleOrgTeamList(context.Background())
	})
	var members []map[string]any
	if err := json.Unmarshal([]byte(output), &members); err != nil {
		t.Fatalf("org team list output is not JSON: %q: %v", output, err)
	}
	if len(members) != 0 {
		t.Fatalf("members = %#v, want empty array", members)
	}
}

func setupOrgRemediationContext(t *testing.T, serverURL, orgID, orgName string) {
	t.Helper()
	configDir := filepath.Join(t.TempDir(), ".satusky")
	if err := os.MkdirAll(filepath.Join(configDir, "profiles"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "profiles", "test.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "context.json"), []byte(`{"active_profile":"test"}`), 0600); err != nil {
		t.Fatal(err)
	}
	originalStore := satuskyctx.Default()
	satuskyctx.SetDefault(satuskyctx.NewTestStore(configDir))
	t.Cleanup(func() { satuskyctx.SetDefault(originalStore) })
	if err := satuskyctx.SetToken("test-token"); err != nil {
		t.Fatal(err)
	}
	if err := satuskyctx.SetCurrentOrganization(orgID, orgName, "example-org"); err != nil {
		t.Fatal(err)
	}
	if serverURL != "" {
		t.Setenv("SATUSKY_API_URL", serverURL+"/v1/cli")
	}
}

func captureOrgStdout(t *testing.T, run func() error) string {
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
