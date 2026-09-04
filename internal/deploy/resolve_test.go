package deploy

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
)

func TestResolveDeploymentIDMapsTypedNotFound(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"deployment missing"}`)
	}))
	defer server.Close()
	configureResolveTestContext(t, server.URL+"/v1/cli")

	_, err := ResolveDeploymentID("", "missing", "")
	if err == nil || !strings.Contains(err.Error(), `app "missing" not found in organization test`) ||
		!strings.Contains(err.Error(), "1ctl app list") {
		t.Fatalf("ResolveDeploymentID() error = %v, want actionable app-not-found", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestResolveDeploymentIDPreservesNonNotFoundCause(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"message":"lookup unavailable"}`)
	}))
	defer server.Close()
	configureResolveTestContext(t, server.URL+"/v1/cli")

	_, err := ResolveDeploymentID("", "demo", "")
	if err == nil || !strings.Contains(err.Error(), `resolve app "demo" in organization test`) {
		t.Fatalf("ResolveDeploymentID() error = %v, want lookup context", err)
	}
	if strings.Contains(err.Error(), "not found") {
		t.Fatalf("ResolveDeploymentID() error = %v, must not map non-404 to app-not-found", err)
	}
	var statusErr *api.HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("ResolveDeploymentID() error = %T %v, want wrapped typed 503", err, err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want bounded 3 attempts", attempts)
	}
}

func configureResolveTestContext(t *testing.T, apiURL string) {
	t.Helper()
	store := satuskyctx.NewTestStore(t.TempDir())
	store.SetProfileOverride("test")
	originalStore := satuskyctx.Default()
	satuskyctx.SetDefault(store)
	t.Cleanup(func() { satuskyctx.SetDefault(originalStore) })
	if err := store.SetToken("test-token"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetCurrentNamespace("test"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SATUSKY_API_URL", apiURL)
}
