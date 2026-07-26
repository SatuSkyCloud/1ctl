package nats

import (
	stdcontext "context"
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

func setupNATSHandlerTest(t *testing.T, serverURL string) {
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

func TestHandleCreateSendsExactJetStreamHAProfile(t *testing.T) {
	marketplaceID := uuid.NewString()
	deploymentID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/marketplaces/all":
			_, _ = io.WriteString(w, `{"error":false,"data":[{"marketplace_id":"`+marketplaceID+`","marketplace_name":"nats","deployable":true}]}`)
		case "/v1/marketplaces/deploy/create/tenant-a/" + marketplaceID:
			var request struct {
				DeploymentName string                 `json:"deployment_name"`
				Replicas       int                    `json:"replicas"`
				CPU            string                 `json:"cpu_request"`
				Memory         string                 `json:"memory_request"`
				Storage        string                 `json:"storage_size"`
				Values         map[string]interface{} `json:"values"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.DeploymentName != "orders" || request.Replicas != 3 ||
				request.CPU != "300m" || request.Memory != "512Mi" || request.Storage != "20Gi" {
				t.Fatalf("request = %#v", request)
			}
			config := request.Values["config"].(map[string]interface{})
			cluster := config["cluster"].(map[string]interface{})
			jetstream := config["jetstream"].(map[string]interface{})
			pvc := jetstream["fileStore"].(map[string]interface{})["pvc"].(map[string]interface{})
			if cluster["enabled"] != true || cluster["replicas"] != float64(3) ||
				jetstream["enabled"] != true || pvc["enabled"] != true ||
				pvc["size"] != "20Gi" || pvc["storageClassName"] != "ceph-block" {
				t.Fatalf("profile config = %#v", config)
			}
			merge := config["merge"].(map[string]interface{})
			token := merge["authorization"].(map[string]interface{})["token"]
			if token != "<< $NATS_TOKEN >>" {
				t.Fatalf("token template = %q", token)
			}
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"orders","status":"deploying"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupNATSHandlerTest(t, server.URL)

	err := handleCreate(stdcontext.Background(), createInput{
		Name: "orders", JetStream: true, CPU: "300m", Memory: "512Mi",
		StorageSize: "20Gi", StorageClass: "ceph-block",
	})
	if err != nil {
		t.Fatalf("handleCreate() error = %v", err)
	}
}

func TestHandleCredentialsDownloadsBothOutputsToPrivateFiles(t *testing.T) {
	deploymentID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"orders","marketplace_app_name":"nats"}}`)
		case "/v1/marketplace-outputs/deployments/" + deploymentID + "/outputs/client-url/download":
			_, _ = io.WriteString(w, "nats://secret@orders:4222")
		case "/v1/marketplace-outputs/deployments/" + deploymentID + "/outputs/client-token/download":
			_, _ = io.WriteString(w, "secret")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupNATSHandlerTest(t, server.URL)
	outputDir := filepath.Join(t.TempDir(), "credentials")

	if err := handleCredentials(stdcontext.Background(), credentialsInput{
		Deployment: deploymentID,
		OutputDir:  outputDir,
	}); err != nil {
		t.Fatalf("handleCredentials() error = %v", err)
	}
	for name, want := range map[string]string{
		"client-url.txt":   "nats://secret@orders:4222",
		"client-token.txt": "secret",
	} {
		path := filepath.Join(outputDir, name)
		value, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(value) != want {
			t.Fatalf("%s = %q", name, value)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("%s mode = %o, want 600", name, info.Mode().Perm())
		}
	}
}

func TestHandleDeleteUsesDeploymentLifecycleAndPurgeQuery(t *testing.T) {
	deploymentID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"orders","marketplace_app_name":"nats"}}`)
		case "/v1/deployments/" + deploymentID:
			if r.Method != http.MethodDelete || r.URL.Query().Get("purge_retained") != "true" {
				t.Fatalf("delete request = %s %s", r.Method, r.URL.String())
			}
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","status":"deleting","purge_retained":true}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupNATSHandlerTest(t, server.URL)

	if err := handleDelete(stdcontext.Background(), deleteInput{
		Deployment: deploymentID,
		Yes:        true,
		Purge:      true,
		NoWait:     true,
	}); err != nil {
		t.Fatalf("handleDelete() error = %v", err)
	}
}
