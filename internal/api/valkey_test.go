package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValkeyPasswordResponsesPreserveAcceptedRestartState(t *testing.T) {
	const storageID = "storage-123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		switch request.URL.Path {
		case "/v1/cli/databases/storage-123/valkey/users":
			_, _ = fmt.Fprint(w, `{"error":false,"data":{"user":{"username":"worker","access_preset":"read_write"},"password":"created-secret"}}`)
		case "/v1/cli/databases/storage-123/valkey/users/worker/rotate-password":
			_, _ = fmt.Fprint(w, `{"error":false,"data":{"username":"worker","password":"rotated-secret"}}`)
		case "/v1/cli/databases/storage-123/credentials/rotate":
			_, _ = fmt.Fprint(w, `{"error":false,"data":{"username":"default","password":"default-secret"}}`)
		default:
			t.Errorf("unexpected path %s", request.URL.Path)
		}
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	created, err := CreateValkeyUser(storageID, ValkeyCreateUserRequest{
		Username:     "worker",
		AccessPreset: "read_write",
	})
	if err != nil {
		t.Fatalf("CreateValkeyUser() error = %v", err)
	}
	if !created.RestartPending || created.Password != "created-secret" {
		t.Fatalf("created response = %#v, want password and pending restart", created)
	}

	rotated, err := RotateValkeyUserPassword(storageID, "worker")
	if err != nil {
		t.Fatalf("RotateValkeyUserPassword() error = %v", err)
	}
	if !rotated.RestartPending || rotated.Password != "rotated-secret" {
		t.Fatalf("rotated response = %#v, want password and pending restart", rotated)
	}

	defaultCredential, err := RotateValkeyCredentials(storageID)
	if err != nil {
		t.Fatalf("RotateValkeyCredentials() error = %v", err)
	}
	if !defaultCredential.RestartPending || defaultCredential.Password != "default-secret" {
		t.Fatalf("default response = %#v, want password and pending restart", defaultCredential)
	}
}

func TestValkeyMetricsPreserveEmptyStringValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/cli/databases/storage-123/metrics" {
			t.Errorf("path = %s, want Valkey metrics path", request.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"error":false,"data":{"connected_clients":"4","replication_link_up":""}}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	metrics, err := GetValkeyMetrics("storage-123")
	if err != nil {
		t.Fatalf("GetValkeyMetrics() error = %v", err)
	}
	if metrics["connected_clients"] != "4" {
		t.Fatalf("connected_clients = %q, want 4", metrics["connected_clients"])
	}
	if value, ok := metrics["replication_link_up"]; !ok || value != "" {
		t.Fatalf("replication_link_up = %q, %t; want present empty string", value, ok)
	}
}
