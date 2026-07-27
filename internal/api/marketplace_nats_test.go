package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	cliContext "1ctl/internal/context"
	"github.com/google/uuid"
)

func setupNATSAPITest(t *testing.T, serverURL string) {
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
	if err := cliContext.SetCurrentOrganization("org-id", "Test", "tenant-a"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SATUSKY_API_URL", serverURL+"/v1/cli")
}

func TestDeployMarketplaceAppSendsNATSValuesContract(t *testing.T) {
	marketplaceID := uuid.NewString()
	deploymentID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost ||
			r.URL.Path != "/v1/marketplaces/deploy/create/tenant-a/"+marketplaceID {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("x-satusky-organization-id"); got != "org-id" {
			t.Fatalf("organization header = %q", got)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["deployment_name"] != "orders" || body["replicas"] != float64(3) {
			t.Fatalf("body = %#v", body)
		}
		values, ok := body["values"].(map[string]interface{})
		if !ok {
			t.Fatalf("values = %#v", body["values"])
		}
		config, ok := values["config"].(map[string]interface{})
		if !ok {
			t.Fatalf("config = %#v", values["config"])
		}
		cluster := config["cluster"].(map[string]interface{})
		if cluster["enabled"] != true || cluster["replicas"] != float64(3) {
			t.Fatalf("cluster = %#v", cluster)
		}
		_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"orders","status":"deploying"}}`)
	}))
	defer server.Close()
	setupNATSAPITest(t, server.URL)

	_, err := DeployMarketplaceApp("tenant-a", marketplaceID, MarketplaceDeployRequest{
		DeploymentName: "orders",
		Replicas:       3,
		Values: map[string]interface{}{
			"config": map[string]interface{}{
				"cluster": map[string]interface{}{"enabled": true, "replicas": 3},
			},
		},
	})
	if err != nil {
		t.Fatalf("DeployMarketplaceApp() error = %v", err)
	}
}

func TestDownloadMarketplaceDeploymentOutputPathHeadersAndRawResponse(t *testing.T) {
	deploymentID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/v1/marketplace-outputs/deployments/" + deploymentID + "/outputs/client-token/download"
		if r.Method != http.MethodGet || r.URL.Path != wantPath {
			t.Fatalf("request = %s %s, want GET %s", r.Method, r.URL.Path, wantPath)
		}
		if got := r.Header.Get("x-satusky-api-key"); got != "test-token" {
			t.Fatalf("token header = %q", got)
		}
		if got := r.Header.Get("x-satusky-organization-id"); got != "org-id" {
			t.Fatalf("organization header = %q", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "super-secret-token")
	}))
	defer server.Close()
	setupNATSAPITest(t, server.URL)

	value, err := DownloadMarketplaceDeploymentOutput(deploymentID, "client-token")
	if err != nil {
		t.Fatalf("DownloadMarketplaceDeploymentOutput() error = %v", err)
	}
	if string(value) != "super-secret-token" {
		t.Fatalf("value = %q", value)
	}
}
