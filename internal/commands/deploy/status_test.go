package deploy

import (
	stdcontext "context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	cliContext "1ctl/internal/context"

	"github.com/google/uuid"
)

func TestDeploymentStatusUsesCanonicalRecordForMarketplaceDeployments(t *testing.T) {
	deploymentID := uuid.NewString()
	var lookupCalls, canonicalCalls, legacyStatusCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/namespace/tenant-a/app/pocketbase":
			lookupCalls++
			_, _ = io.WriteString(w, marketplaceDeploymentJSON(deploymentID))
		case "/v1/cli/deployments/id/" + deploymentID:
			canonicalCalls++
			_, _ = io.WriteString(w, marketplaceDeploymentJSON(deploymentID))
		case "/v1/cli/deployments/status/" + deploymentID:
			legacyStatusCalls++
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)

	if err := handleDeploymentStatus(stdcontext.Background(), StatusInput{App: "pocketbase"}); err != nil {
		t.Fatalf("marketplace status by name: %v", err)
	}
	if err := handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID}); err != nil {
		t.Fatalf("marketplace status by deployment ID: %v", err)
	}
	if lookupCalls != 1 || canonicalCalls != 2 || legacyStatusCalls != 0 {
		t.Fatalf("marketplace status calls lookup=%d canonical=%d legacy=%d; want 1, 2, 0", lookupCalls, canonicalCalls, legacyStatusCalls)
	}
}

func TestDeploymentStatusKeepsLegacyStatusEndpointForDirectDeployments(t *testing.T) {
	deploymentID := uuid.NewString()
	var canonicalCalls, legacyStatusCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			canonicalCalls++
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"direct","namespace":"tenant-a","status":"ready","source":"generic"}}`)
		case "/v1/cli/deployments/status/" + deploymentID:
			legacyStatusCalls++
			_, _ = io.WriteString(w, `{"error":false,"data":{"status":"Running","progress":100}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)

	if err := handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID}); err != nil {
		t.Fatalf("direct deployment status: %v", err)
	}
	if canonicalCalls != 1 || legacyStatusCalls != 1 {
		t.Fatalf("direct status calls canonical=%d legacy=%d; want 1, 1", canonicalCalls, legacyStatusCalls)
	}
}

func TestDeploymentStatusDoesNotHideCanonicalNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)

	if err := handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: uuid.NewString()}); err == nil {
		t.Fatal("marketplace-capable status lookup unexpectedly hid canonical not found")
	}
}

func setupDeploymentStatusTest(t *testing.T, apiURL string) {
	t.Helper()
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
	if err := cliContext.SetCurrentNamespace("tenant-a"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SATUSKY_API_URL", apiURL+"/v1/cli")
}

func marketplaceDeploymentJSON(deploymentID string) string {
	return `{"error":false,"data":{"deployment_id":"` + deploymentID + `","app_label":"pocketbase","namespace":"tenant-a","status":"ready","source":"marketplace","marketplace_app_name":"pocketbase"}}`
}
