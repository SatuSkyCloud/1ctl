package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestManagedRestartRetryReusesRequestID(t *testing.T) {
	var mu sync.Mutex
	var requestIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestIDs = append(requestIDs, r.Header.Get(requestIDHeader))
		attempt := len(requestIDs)
		mu.Unlock()

		if attempt == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"message":"try again"}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	if err := RestartDeployment(uuid.NewString(), uuid.NewString()); err != nil {
		t.Fatalf("RestartDeployment() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(requestIDs) != 2 {
		t.Fatalf("attempts = %d, want 2", len(requestIDs))
	}
	if requestIDs[0] == "" || requestIDs[0] != requestIDs[1] {
		t.Fatalf("request IDs = %q, want the same non-empty ID", requestIDs)
	}
}

func TestManagedOperationsUseDifferentRequestIDs(t *testing.T) {
	var requestIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestIDs = append(requestIDs, r.Header.Get(requestIDHeader))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	if err := RestartDeployment(uuid.NewString(), uuid.NewString()); err != nil {
		t.Fatalf("first RestartDeployment() error = %v", err)
	}
	if err := RestartDeployment(uuid.NewString(), uuid.NewString()); err != nil {
		t.Fatalf("second RestartDeployment() error = %v", err)
	}
	if len(requestIDs) != 2 || requestIDs[0] == "" || requestIDs[0] == requestIDs[1] {
		t.Fatalf("request IDs = %q, want two different non-empty IDs", requestIDs)
	}
}

func TestManagedRollback409IsNotRetried(t *testing.T) {
	attempts := 0
	requestID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if got := r.Header.Get(requestIDHeader); got != requestID {
			t.Errorf("%s = %q, want %q", requestIDHeader, got, requestID)
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = fmt.Fprint(w, `{"message":"deployment version conflict"}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	err := RollbackDeployment(uuid.NewString(), 2, requestID)
	if err == nil || !strings.Contains(err.Error(), "deployment version conflict") {
		t.Fatalf("RollbackDeployment() error = %v, want conflict", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestDeploymentUpsertUsesCanonicalPutWithoutCreateRetry(t *testing.T) {
	attempts := 0
	requestID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/v1/deployments/upsert/ns/app" {
			t.Errorf("path = %s, want canonical upsert path", r.URL.Path)
		}
		if got := r.Header.Get(requestIDHeader); got != requestID {
			t.Errorf("%s = %q, want %q", requestIDHeader, got, requestID)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(w, `{"message":"try again"}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	err := UpsertDeployment(Deployment{Namespace: "ns", AppLabel: "app"}, nil, requestID)
	if err == nil {
		t.Fatal("UpsertDeployment() error = nil, want unavailable error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want one non-retried create", attempts)
	}
}
