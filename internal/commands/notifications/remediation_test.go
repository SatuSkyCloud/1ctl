package notifications

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

func TestNotificationDeleteRejectsMissingID(t *testing.T) {
	err := notifDeleteCommand().Run(context.Background(), []string{"delete"})
	if err == nil || !strings.Contains(err.Error(), "notification ID is required") {
		t.Fatalf("error = %v, want notification ID required error", err)
	}
}

func TestHandleNotifCountJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/cli/organizations/org-123/notifications/unread-count" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"error":false,"data":{"count":7}}`)
	}))
	defer server.Close()

	setupNotificationRemediationContext(t, server.URL, "org-123")
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })

	output := captureNotificationStdout(t, func() error {
		return handleNotifCount(context.Background())
	})
	var count struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(output), &count); err != nil {
		t.Fatalf("notification count output is not JSON: %q: %v", output, err)
	}
	if count.Count != 7 {
		t.Fatalf("count = %d, want 7", count.Count)
	}
}

func setupNotificationRemediationContext(t *testing.T, serverURL, orgID string) {
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
	if err := satuskyctx.SetCurrentOrgID(orgID); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SATUSKY_API_URL", serverURL+"/v1/cli")
}

func captureNotificationStdout(t *testing.T, run func() error) string {
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
