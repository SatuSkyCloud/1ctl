package cleanup

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cliContext "1ctl/internal/context"
)

func TestCleanupManager_AddAndCleanupRegistry(t *testing.T) {
	cm := NewCleanupManager()
	cm.AddResource(ResourceDeployment, "dep-1", "myapp")
	cm.AddResource(ResourceService, "svc-1", "myapp")
	cm.AddResource(ResourceVolume, "myapp-volume", "myapp")

	// We don't exercise the actual API delete calls here (they require a
	// live backend). The test asserts that resources are tracked in
	// reverse-order registration so cleanup runs in dependency order.
	if len(cm.resources) != 3 {
		t.Fatalf("AddResource: have %d resources, want 3", len(cm.resources))
	}
	if cm.resources[0].Type != ResourceDeployment ||
		cm.resources[1].Type != ResourceService ||
		cm.resources[2].Type != ResourceVolume {
		t.Errorf("registration order wrong: %+v", cm.resources)
	}
}

func TestCleanupManagerReportsAcceptedDeletionAsInProgress(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/deployments/dep-1" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"dep-1","operation":"delete","status":"deleting","status_url":"`+server.URL+`/v1/deployments/id/dep-1"}}`)
	}))
	defer server.Close()
	originalStore := cliContext.Default()
	configDir := filepath.Join(t.TempDir(), ".satusky")
	if err := os.MkdirAll(filepath.Join(configDir, "profiles"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "profiles", "test.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "context.json"), []byte(`{"active_profile":"test"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cliContext.SetDefault(cliContext.NewTestStore(configDir))
	t.Cleanup(func() { cliContext.SetDefault(originalStore) })
	if err := cliContext.SetToken("test-token"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SATUSKY_API_URL", server.URL+"/v1/cli")

	cm := NewCleanupManager()
	cm.AddResource(ResourceDeployment, "dep-1", "app")
	errs := cm.Cleanup()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "accepted asynchronously") || !strings.Contains(errs[0].Error(), "in progress") {
		t.Fatalf("cleanup errors = %v", errs)
	}
}

func TestFormatCleanupErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   []error
		expected string
	}{
		{
			name:     "no errors",
			errors:   nil,
			expected: "",
		},
		{
			name: "single error",
			errors: []error{
				fmt.Errorf("test error"),
			},
			expected: "Cleanup errors:\ntest error",
		},
		{
			name: "multiple errors",
			errors: []error{
				fmt.Errorf("error 1"),
				fmt.Errorf("error 2"),
			},
			expected: "Cleanup errors:\nerror 1\nerror 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCleanupErrors(tt.errors)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
