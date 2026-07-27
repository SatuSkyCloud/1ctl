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

	"1ctl/internal/api"
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

func TestHandleListMatchesBackendDeploymentByPackageRelease(t *testing.T) {
	natsReleaseID := uuid.NewString()
	otherReleaseID := uuid.NewString()
	deploymentID := uuid.NewString()
	catalogRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/namespace/tenant-a":
			_, _ = io.WriteString(w, `{"error":false,"data":[`+
				`{"deployment_id":"`+deploymentID+`","app_label":"orders","marketplace_release_id":"`+natsReleaseID+`"},`+
				`{"deployment_id":"`+uuid.NewString()+`","app_label":"other","marketplace_release_id":"`+otherReleaseID+`"}`+
				`]}`)
		case "/v1/marketplaces/all":
			catalogRequests++
			_, _ = io.WriteString(w, natsCatalogResponse(natsReleaseID))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupNATSHandlerTest(t, server.URL)

	if err := handleList(stdcontext.Background()); err != nil {
		t.Fatalf("handleList() error = %v", err)
	}
	if catalogRequests != 1 {
		t.Fatalf("catalog requests = %d, want 1", catalogRequests)
	}
}

func TestIsNATSDeploymentKeepsNameCompatibility(t *testing.T) {
	if !isNATSDeployment(api.Deployment{MarketplaceAppName: " NATS "}, nil) {
		t.Fatal("legacy marketplace_app_name did not match NATS")
	}
	releaseID := uuid.New()
	deployment := api.Deployment{MarketplaceReleaseID: releaseID.String()}
	app := &api.MarketplaceApp{
		PackageRelease: &api.MarketplacePackageRelease{ReleaseID: releaseID},
	}
	if !isNATSDeployment(deployment, app) {
		t.Fatal("matching marketplace_release_id did not match NATS")
	}
	if isNATSDeployment(deployment, nil) {
		t.Fatal("release-only deployment matched without the NATS catalog app")
	}
	app.PackageRelease.ReleaseID = uuid.New()
	if isNATSDeployment(deployment, app) {
		t.Fatal("different marketplace_release_id matched NATS")
	}
}

func TestHandleGetMatchesBackendDeploymentByPackageRelease(t *testing.T) {
	natsReleaseID := uuid.NewString()
	deploymentID := uuid.NewString()
	catalogRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			_, _ = io.WriteString(w, backendNATSDeploymentResponse(deploymentID, natsReleaseID))
		case "/v1/marketplaces/all":
			catalogRequests++
			_, _ = io.WriteString(w, natsCatalogResponse(natsReleaseID))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupNATSHandlerTest(t, server.URL)

	if err := handleGet(stdcontext.Background(), deploymentInput{Deployment: deploymentID}); err != nil {
		t.Fatalf("handleGet() error = %v", err)
	}
	if catalogRequests != 1 {
		t.Fatalf("catalog requests = %d, want 1", catalogRequests)
	}
}

func TestHandleStatusMatchesBackendDeploymentByPackageRelease(t *testing.T) {
	natsReleaseID := uuid.NewString()
	deploymentID := uuid.NewString()
	catalogRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			_, _ = io.WriteString(w, backendNATSDeploymentResponse(deploymentID, natsReleaseID))
		case "/v1/marketplaces/all":
			catalogRequests++
			_, _ = io.WriteString(w, natsCatalogResponse(natsReleaseID))
		case "/v1/cli/deployments/status/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"status":"running","progress":100}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupNATSHandlerTest(t, server.URL)

	if err := handleStatus(stdcontext.Background(), deploymentInput{Deployment: deploymentID}); err != nil {
		t.Fatalf("handleStatus() error = %v", err)
	}
	if catalogRequests != 1 {
		t.Fatalf("catalog requests = %d, want 1", catalogRequests)
	}
}

func TestHandleCredentialsDownloadsBothOutputsToPrivateFiles(t *testing.T) {
	natsReleaseID := uuid.NewString()
	deploymentID := uuid.NewString()
	catalogRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			_, _ = io.WriteString(w, backendNATSDeploymentResponse(deploymentID, natsReleaseID))
		case "/v1/marketplaces/all":
			catalogRequests++
			_, _ = io.WriteString(w, natsCatalogResponse(natsReleaseID))
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
	if catalogRequests != 1 {
		t.Fatalf("catalog requests = %d, want 1", catalogRequests)
	}
}

func TestWriteCredentialFilesRefusesExistingFileWithoutChangingContent(t *testing.T) {
	outputDir := t.TempDir()
	existingPath := filepath.Join(outputDir, "client-token.txt")
	if err := os.WriteFile(existingPath, []byte("keep-me"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(existingPath, 0644); err != nil {
		t.Fatal(err)
	}

	err := writeCredentialFiles(outputDir, map[string][]byte{
		"client-url":   []byte("nats://new-secret"),
		"client-token": []byte("new-secret"),
	})
	if err == nil {
		t.Fatal("writeCredentialFiles() error = nil, want existing-file refusal")
	}
	value, readErr := os.ReadFile(existingPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(value) != "keep-me" {
		t.Fatalf("existing content = %q, want unchanged", value)
	}
	info, statErr := os.Stat(existingPath)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if info.Mode().Perm() != 0644 {
		t.Fatalf("existing mode = %o, want unchanged 644", info.Mode().Perm())
	}
	if _, statErr := os.Stat(filepath.Join(outputDir, "client-url.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("client-url.txt stat error = %v, want not exist", statErr)
	}
}

func TestHandleDeleteUsesDeploymentLifecycleAndPurgeQuery(t *testing.T) {
	natsReleaseID := uuid.NewString()
	deploymentID := uuid.NewString()
	catalogRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			_, _ = io.WriteString(w, backendNATSDeploymentResponse(deploymentID, natsReleaseID))
		case "/v1/marketplaces/all":
			catalogRequests++
			_, _ = io.WriteString(w, natsCatalogResponse(natsReleaseID))
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
	if catalogRequests != 1 {
		t.Fatalf("catalog requests = %d, want 1", catalogRequests)
	}
}

func natsCatalogResponse(releaseID string) string {
	return `{"error":false,"data":[{"marketplace_id":"` + uuid.NewString() +
		`","marketplace_name":"nats","package_release":{"release_id":"` + releaseID + `"}}]}`
}

func backendNATSDeploymentResponse(deploymentID, releaseID string) string {
	return `{"error":false,"data":{"deployment_id":"` + deploymentID +
		`","app_label":"orders","marketplace_release_id":"` + releaseID + `"}}`
}
