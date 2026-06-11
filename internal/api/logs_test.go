package api

import (
	"1ctl/internal/context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGetStoredLogsReturnsSourceMetadata(t *testing.T) {
	originalStore := context.Default()
	t.Cleanup(func() { context.SetDefault(originalStore) })

	tempDir := t.TempDir()
	store := context.NewTestStore(tempDir)
	store.SetProfileOverride("test")
	context.SetDefault(store)

	if err := context.SetToken("test-token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-satusky-api-key"); got != "test-token" {
			t.Fatalf("x-satusky-api-key = %q, want test-token", got)
		}
		if got := r.URL.Path; got != "/v1/cli/loki/logs/deployment-123" {
			t.Fatalf("path = %q, want /v1/cli/loki/logs/deployment-123", got)
		}
		if got := r.URL.Query().Get("tail"); got != "2" {
			t.Fatalf("tail = %q, want 2", got)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		fmt.Fprintf(w, `{
			"error": false,
			"message": "Logs retrieved successfully via stored deployment logs",
			"data": [
				{
					"log_id": %q,
					"deployment_id": %q,
					"timestamp": %q,
					"message": "fallback line",
					"pod_name": "",
					"container": "",
					"level": ""
				}
			],
			"source": "stored",
			"degraded": true,
			"fallback_reason": "loki query unavailable; served stored deployment logs",
			"fallback_source": "loki",
			"count": 1
		}`, uuid.New(), uuid.New(), now)
	}))
	defer server.Close()

	t.Setenv("SATUSKY_API_URL", server.URL+"/v1/cli")

	logs, meta, err := GetStoredLogs("deployment-123", 2)
	if err != nil {
		t.Fatalf("GetStoredLogs() error = %v", err)
	}

	if meta == nil {
		t.Fatal("GetStoredLogs() meta = nil")
	}

	if len(logs) != 1 {
		t.Fatalf("len(logs) = %d, want 1", len(logs))
	}

	if meta.Source != "stored" {
		t.Fatalf("meta.Source = %q, want stored", meta.Source)
	}
	if !meta.Degraded {
		t.Fatal("meta.Degraded = false, want true")
	}
	if meta.FallbackSource != "loki" {
		t.Fatalf("meta.FallbackSource = %q, want loki", meta.FallbackSource)
	}
	if !strings.Contains(meta.Message, "stored deployment logs") {
		t.Fatalf("meta.Message = %q, want stored deployment logs note", meta.Message)
	}
}
