package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestGetDeploymentByAppLabelRetriesTransientServerErrors(t *testing.T) {
	deploymentID := uuid.New()
	var mu sync.Mutex
	attempts := 0
	unexpectedPath := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		pathMismatch := r.URL.Path != "/v1/cli/deployments/namespace/test/app/demo"
		if pathMismatch {
			unexpectedPath = r.URL.Path
		}
		attempt := attempts
		mu.Unlock()
		if pathMismatch {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if attempt < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"message":"temporarily unavailable"}`)
			return
		}
		_, _ = fmt.Fprintf(w, `{"error":false,"data":{"deployment_id":%q,"app_label":"demo"}}`, deploymentID)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	deployment, err := GetDeploymentByAppLabel("test", "demo")
	mu.Lock()
	gotAttempts := attempts
	gotUnexpectedPath := unexpectedPath
	mu.Unlock()
	if gotUnexpectedPath != "" {
		t.Fatalf("path = %s, want /v1/cli/deployments/namespace/test/app/demo", gotUnexpectedPath)
	}
	if err != nil {
		t.Fatalf("GetDeploymentByAppLabel() error = %v", err)
	}
	if gotAttempts != 3 {
		t.Fatalf("attempts = %d, want 3", gotAttempts)
	}
	if deployment.DeploymentID != deploymentID {
		t.Fatalf("deployment ID = %s, want %s", deployment.DeploymentID, deploymentID)
	}
}

func TestGetDeploymentByAppLabelRetriesTransportErrors(t *testing.T) {
	originalClient := httpClient
	t.Cleanup(func() { httpClient = originalClient })

	deploymentID := uuid.New()
	attempts := 0
	httpClient = &http.Client{Transport: deploymentLookupRoundTripper(func(r *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("temporary transport failure")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				fmt.Sprintf(`{"error":false,"data":{"deployment_id":%q,"app_label":"demo"}}`, deploymentID),
			)),
		}, nil
	})}
	configureAdminAPITestContext(t, "http://localhost:8080/v1/cli")

	deployment, err := GetDeploymentByAppLabel("test", "demo")
	if err != nil {
		t.Fatalf("GetDeploymentByAppLabel() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if deployment.DeploymentID != deploymentID {
		t.Fatalf("deployment ID = %s, want %s", deployment.DeploymentID, deploymentID)
	}
}

func TestGetDeploymentByAppLabelDoesNotRetryNotFound(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"deployment not found"}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	_, err := GetDeploymentByAppLabel("test", "missing")
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusNotFound {
		t.Fatalf("GetDeploymentByAppLabel() error = %T %v, want typed 404", err, err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

type deploymentLookupRoundTripper func(*http.Request) (*http.Response, error)

func (fn deploymentLookupRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
