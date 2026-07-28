package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestHTTPStatusErrorPreservesTypedBackendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":true,"code":"RATE_LIMITED","message":"rate limit reached","details":{"limit":10,"scope":"organization"},"retryable":true,"remediation":["wait before retrying","reduce request rate"],"request_id":"req-123"}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	err := makeRequest(http.MethodGet, "/errors", nil, nil)
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error = %T %v, want HTTPStatusError", err, err)
	}
	if statusErr.StatusCode != http.StatusTooManyRequests || statusErr.Message != "rate limit reached" || statusErr.Code != "RATE_LIMITED" {
		t.Fatalf("status error = %+v", statusErr)
	}
	if statusErr.RetryAfter != 3*time.Second || statusErr.Retryable == nil || !*statusErr.Retryable || statusErr.RequestID != "req-123" {
		t.Fatalf("typed diagnostics = %+v", statusErr)
	}
	if !reflect.DeepEqual(statusErr.Details, map[string]interface{}{"limit": float64(10), "scope": "organization"}) {
		t.Fatalf("details = %#v", statusErr.Details)
	}
	if !reflect.DeepEqual(statusErr.Remediation, []string{"wait before retrying", "reduce request rate"}) {
		t.Fatalf("remediation = %#v", statusErr.Remediation)
	}
}

func TestHTTPStatusErrorAcceptsLegacyBackendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":true,"message":"legacy failure","details":"legacy detail"}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	err := makeRequest(http.MethodGet, "/errors", nil, nil)
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error = %T %v, want HTTPStatusError", err, err)
	}
	if statusErr.StatusCode != http.StatusInternalServerError || statusErr.Message != "legacy failure" || statusErr.Details != "legacy detail" {
		t.Fatalf("legacy status error = %+v", statusErr)
	}
	if statusErr.Code != "" || statusErr.Retryable != nil || len(statusErr.Remediation) != 0 || statusErr.RequestID != "" {
		t.Fatalf("legacy typed fields = %+v, want empty defaults", statusErr)
	}
}

func TestHTTPStatusErrorDoesNotExposeUnparsedResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "upstream failure: token=secret-value")
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	err := makeRequest(http.MethodGet, "/errors", nil, nil)
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error = %T %v, want HTTPStatusError", err, err)
	}
	if statusErr.Message != "request failed with status 502" || statusErr.Details != nil {
		t.Fatalf("unparsed status error = %+v", statusErr)
	}
}

func TestCreateDeploymentIntentWrapPreservesTypedBackendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/cli/deployments/intent" {
			t.Errorf("request = %s %s, want POST /v1/cli/deployments/intent", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"error":true,"code":"VALIDATION_ERROR","message":"invalid deployment","details":{"field":"app_label"},"retryable":false,"remediation":["set app_label"],"request_id":"req-456"}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	_, err := CreateDeploymentIntent(DeploymentIntent{}, "caller-request-id")
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error = %T %v, want wrapped HTTPStatusError", err, err)
	}
	if statusErr.Code != "VALIDATION_ERROR" || statusErr.Retryable == nil || *statusErr.Retryable || statusErr.RequestID != "req-456" {
		t.Fatalf("wrapped typed diagnostics = %+v", statusErr)
	}
}
