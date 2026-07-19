package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	cliContext "1ctl/internal/context"

	"github.com/google/uuid"
)

func TestAdoptDeploymentSendsMainAPIPathBodyAndRequestID(t *testing.T) {
	deploymentID := uuid.NewString()
	requestID := uuid.NewString()
	want := DeploymentAdoptionRequest{
		Reason:                  "legacy canary",
		ExpectedUID:             uuid.NewString(),
		ExpectedResourceVersion: "412",
		ExpectedGeneration:      7,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/v1/admin/deployments/"+deploymentID+"/adopt" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if got := request.Header.Get(requestIDHeader); got != requestID {
			t.Errorf("%s = %q, want %q", requestIDHeader, got, requestID)
		}
		var got DeploymentAdoptionRequest
		if err := json.NewDecoder(request.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if got != want {
			t.Errorf("body = %+v, want %+v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"error": false,
			"data": map[string]string{
				"deployment_id": deploymentID,
				"app_label":     "canary",
				"request_id":    requestID,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Encode() error = %v", err)
		}
	}))
	defer server.Close()

	configureAdminAPITestContext(t, server.URL+"/v1/cli")
	result, err := AdoptDeployment(deploymentID, requestID, want)
	if err != nil {
		t.Fatalf("AdoptDeployment() error = %v", err)
	}
	if result.DeploymentID != deploymentID || result.RequestID != requestID || result.AppLabel != "canary" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAdoptDeploymentRoutingSendsMainAPIPathBodyAndRequestID(t *testing.T) {
	deploymentID := uuid.NewString()
	requestID := uuid.NewString()
	want := DeploymentAdoptionRequest{
		Reason:                  "routing cutover",
		ExpectedUID:             uuid.NewString(),
		ExpectedResourceVersion: "719",
		ExpectedGeneration:      9,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/v1/admin/deployments/"+deploymentID+"/routing/adopt" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if got := request.Header.Get(requestIDHeader); got != requestID {
			t.Errorf("%s = %q, want %q", requestIDHeader, got, requestID)
		}
		var got DeploymentAdoptionRequest
		if err := json.NewDecoder(request.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if got != want {
			t.Errorf("body = %+v, want %+v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error": false,
			"data": map[string]interface{}{
				"deployment_id":   deploymentID,
				"name":            "canary-route",
				"force":           true,
				"already_managed": false,
				"request_id":      requestID,
			},
		}); err != nil {
			t.Errorf("Encode() error = %v", err)
		}
	}))
	defer server.Close()

	configureAdminAPITestContext(t, server.URL+"/v1/cli")
	result, err := AdoptDeploymentRouting(deploymentID, requestID, want)
	if err != nil {
		t.Fatalf("AdoptDeploymentRouting() error = %v", err)
	}
	if result.DeploymentID != deploymentID || result.RequestID != requestID || result.Name != "canary-route" ||
		!result.Force || result.AlreadyManaged {
		t.Fatalf("result = %+v", result)
	}
}

func configureAdminAPITestContext(t *testing.T, apiURL string) {
	t.Helper()
	originalStore := cliContext.Default()
	configDir := filepath.Join(t.TempDir(), ".satusky")
	profilesDir := filepath.Join(configDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, "test.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("WriteFile(profile) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "context.json"), []byte(`{"active_profile":"test"}`), 0600); err != nil {
		t.Fatalf("WriteFile(context) error = %v", err)
	}
	cliContext.SetDefault(cliContext.NewTestStore(configDir))
	t.Cleanup(func() { cliContext.SetDefault(originalStore) })
	if err := cliContext.SetToken("test-token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}
	t.Setenv("SATUSKY_API_URL", apiURL)
}
