package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func buildContextArchive(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "context.tar.gz")
	if err := os.WriteFile(path, []byte("not-a-real-archive"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func buildSubmitServer(t *testing.T, status int, body string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server.URL + "/v1/cli"
}

// A cloud build is queued rather than performed, so the backend answers
// 202 Accepted with the build ID to poll. Enumerating 200 and 201 as the only
// acceptable codes turned that correct response into "deployment failed" while
// the build carried on running server-side.
func TestSubmitBuildAcceptsQueuedResponse(t *testing.T) {
	configureAdminAPITestContext(t, buildSubmitServer(t, http.StatusAccepted,
		`{"error":false,"data":{"build_id":"a6be58ac-afb4-46bd-a29f-c1b4f25a70e4","status":"queued"}}`))

	buildID, err := SubmitBuild(buildContextArchive(t), "backend", "Dockerfile", "", nil)

	if err != nil {
		t.Fatalf("SubmitBuild() error = %v, want the queued build to be accepted", err)
	}
	if buildID != "a6be58ac-afb4-46bd-a29f-c1b4f25a70e4" {
		t.Fatalf("buildID = %q, want the ID from the accepted response", buildID)
	}
}

func TestSubmitBuildAcceptsOKAndCreated(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		configureAdminAPITestContext(t, buildSubmitServer(t, status,
			`{"error":false,"data":{"build_id":"11111111-1111-1111-1111-111111111111","status":"queued"}}`))

		buildID, err := SubmitBuild(buildContextArchive(t), "backend", "Dockerfile", "", nil)
		if err != nil {
			t.Fatalf("SubmitBuild() with HTTP %d error = %v", status, err)
		}
		if buildID == "" {
			t.Fatalf("SubmitBuild() with HTTP %d returned an empty build ID", status)
		}
	}
}

// A real rejection must still be reported, otherwise widening the accepted
// range would swallow genuine failures.
func TestSubmitBuildReportsRejection(t *testing.T) {
	configureAdminAPITestContext(t, buildSubmitServer(t, http.StatusBadRequest,
		`{"error":true,"message":"dockerfile not found"}`))

	_, err := SubmitBuild(buildContextArchive(t), "backend", "Dockerfile", "", nil)

	if err == nil {
		t.Fatal("SubmitBuild() error = nil, want the rejection reported")
	}
	if !strings.Contains(err.Error(), "dockerfile not found") {
		t.Fatalf("error = %q, want the server's reason", err)
	}
}

// An envelope flagged as an error is a rejection whatever the status code says.
func TestSubmitBuildHonoursTheErrorFlag(t *testing.T) {
	configureAdminAPITestContext(t, buildSubmitServer(t, http.StatusAccepted,
		`{"error":true,"data":{"build_id":"","status":"failed"}}`))

	_, err := SubmitBuild(buildContextArchive(t), "backend", "Dockerfile", "", nil)

	if err == nil {
		t.Fatal("SubmitBuild() error = nil, want the flagged rejection reported")
	}
}

// An accepted response carrying no build ID would otherwise return an empty
// string and leave the caller polling /builds//status indefinitely.
func TestSubmitBuildRejectsAnAcceptedResponseWithNoBuildID(t *testing.T) {
	configureAdminAPITestContext(t, buildSubmitServer(t, http.StatusAccepted,
		`{"error":false,"data":{"status":"queued"}}`))

	_, err := SubmitBuild(buildContextArchive(t), "backend", "Dockerfile", "", nil)

	if err == nil {
		t.Fatal("SubmitBuild() error = nil, want a missing build ID reported")
	}
	if !strings.Contains(err.Error(), "no build ID") {
		t.Fatalf("error = %q, want it to name the missing build ID", err)
	}
}
