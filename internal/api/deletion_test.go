package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func deletionOperationJSON(status, state string, terminal bool, statusURL string) string {
	return `{"error":false,"data":{"deployment_id":"dep-1","namespace":"ns","app_label":"app","operation":"delete","status":"` + status + `","terminal":` + boolString(terminal) + `,"status_url":"` + statusURL + `","poll_after_ms":1,"purge_retained":true,"cleanup_scope":["deployment"],"lifecycle":{"state":"` + state + `","terminal":` + boolString(terminal) + `,"retryable":false}}}`
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func TestDeleteDeploymentCanonicalAsyncContractAndPurgeQuery(t *testing.T) {
	var deletes atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/deployments/dep-1" || r.Method != http.MethodDelete {
			t.Fatalf("request = %s %s, want canonical DELETE", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("purge_retained") != "true" || r.URL.RawQuery != "purge_retained=true" {
			t.Fatalf("query = %q, want exactly purge_retained=true", r.URL.RawQuery)
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Fatalf("DELETE body = %q, want empty", body)
		}
		deletes.Add(1)
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, deletionOperationJSON("deleting", "deleting", false, server.URL+"/v1/deletion-status/dep-1"))
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	op, err := DeleteDeployment("dep-1", true)
	if err != nil {
		t.Fatalf("DeleteDeployment() error = %v", err)
	}
	if op.Status != "deleting" || op.Terminal || !op.PurgeRetained || op.StatusURL == "" {
		t.Fatalf("operation = %+v, want accepted deleting operation", op)
	}
	if deletes.Load() != 1 {
		t.Fatalf("delete count = %d, want 1", deletes.Load())
	}
}

func TestWaitForDeploymentDeletionTerminalStates(t *testing.T) {
	tests := []struct {
		name       string
		first      string
		second     string
		wantState  string
		wantFailed bool
	}{
		{name: "deleting then deleted", first: "deleting", second: "deleted", wantState: "deleted"},
		{name: "deletion failed", first: "deleting", second: "deletion_failed", wantState: "deletion_failed", wantFailed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var polls atomic.Int32
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/deletion-status/dep-1" {
					t.Fatalf("poll path = %s", r.URL.Path)
				}
				state := tt.first
				if polls.Add(1) > 1 {
					state = tt.second
				}
				terminal := state != "deleting"
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, deletionOperationJSON(state, state, terminal, server.URL+r.URL.Path))
			}))
			defer server.Close()
			configureAdminAPITestContext(t, server.URL+"/v1/cli")

			op := &DeploymentDeletionOperation{DeploymentID: "dep-1", Status: "deleting", StatusURL: server.URL + "/v1/deletion-status/dep-1", PollAfterMs: 1}
			final, err := WaitForDeploymentDeletion(op, time.Second)
			if err != nil {
				t.Fatalf("WaitForDeploymentDeletion() error = %v", err)
			}
			if final.Lifecycle.State != tt.wantState || final.IsSuccessful() != !tt.wantFailed {
				t.Fatalf("final = %+v", final)
			}
		})
	}
}

func TestWaitForDeploymentDeletion404AfterAcceptanceIsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"not found"}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	op, err := WaitForDeploymentDeletion(&DeploymentDeletionOperation{
		DeploymentID: "dep-1", Status: "deleting", StatusURL: server.URL + "/v1/deletion-status/dep-1", PollAfterMs: 1,
	}, time.Second)
	if err != nil || !op.IsSuccessful() || op.Lifecycle.State != "deleted" {
		t.Fatalf("result = %+v, error = %v", op, err)
	}
}

func TestWaitForDeploymentDeletionRetries503AndRejects409(t *testing.T) {
	t.Run("503 retry", func(t *testing.T) {
		var calls atomic.Int32
		var server *httptest.Server
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = io.WriteString(w, `{"message":"busy"}`)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, deletionOperationJSON("deleted", "deleted", true, server.URL+r.URL.Path))
		}))
		defer server.Close()
		configureAdminAPITestContext(t, server.URL+"/v1/cli")
		op, err := WaitForDeploymentDeletion(&DeploymentDeletionOperation{DeploymentID: "dep-1", StatusURL: server.URL + "/v1/status", PollAfterMs: 1}, time.Second)
		if err != nil || !op.IsSuccessful() || calls.Load() != 2 {
			t.Fatalf("operation = %+v, calls = %d, error = %v", op, calls.Load(), err)
		}
	})

	t.Run("409 is terminal error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = io.WriteString(w, `{"message":"deletion conflict"}`)
		}))
		defer server.Close()
		configureAdminAPITestContext(t, server.URL+"/v1/cli")
		_, err := WaitForDeploymentDeletion(&DeploymentDeletionOperation{DeploymentID: "dep-1", StatusURL: server.URL + "/v1/status", PollAfterMs: 1}, time.Second)
		statusErr, ok := err.(*HTTPStatusError)
		if !ok || statusErr.StatusCode != http.StatusConflict {
			t.Fatalf("error = %T %v, want HTTP 409", err, err)
		}
	})
}

func TestWaitForDeploymentDeletionTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, deletionOperationJSON("deleting", "deleting", false, r.URL.String()))
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	op, err := WaitForDeploymentDeletion(&DeploymentDeletionOperation{
		DeploymentID: "dep-1", Status: "deleting", StatusURL: server.URL + "/v1/deletion-status/dep-1", PollAfterMs: 1,
	}, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timeout") || op == nil {
		t.Fatalf("result = %+v, error = %v, want bounded timeout", op, err)
	}
}

func TestDeploymentPurgePolicyUsesBackendMetadata(t *testing.T) {
	if (Deployment{Source: "generic"}).IsMarketplaceManaged() {
		t.Fatal("generic deployment must not allow purge")
	}
	if !(Deployment{Source: "marketplace"}).IsMarketplaceManaged() {
		t.Fatal("marketplace source must allow purge")
	}
	if !(Deployment{DeploymentSource: "marketplace"}).IsMarketplaceManaged() {
		t.Fatal("deployment_source marketplace must allow purge")
	}
	if !(Deployment{MarketplaceReleaseID: "release-1"}).IsMarketplaceManaged() {
		t.Fatal("marketplace release metadata must allow purge")
	}
	if !(Deployment{MarketplaceAppName: "wordpress"}).IsMarketplaceManaged() {
		t.Fatal("marketplace metadata must allow purge")
	}
}

func TestDeploymentDeletionOperationJSONIsTyped(t *testing.T) {
	op := DeploymentDeletionOperation{DeploymentID: "dep-1", Status: "deleted", Terminal: true, Lifecycle: DeploymentDeletionLifecycle{State: "deleted", Terminal: true}}
	data, err := json.Marshal(op)
	if err != nil || !strings.Contains(string(data), `"lifecycle"`) || !strings.Contains(string(data), `"deployment_id"`) {
		t.Fatalf("json = %s, error = %v", data, err)
	}
}
