package api

import (
	cliContext "1ctl/internal/context"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func setupPublisherAPITest(t *testing.T, apiURL string) {
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
	t.Setenv("SATUSKY_API_URL", apiURL+"/v1/cli")
}

func TestMarketplacePublisherClientUsesPublisherEnvelopeAndPaths(t *testing.T) {
	organizationID := uuid.NewString()
	releaseID := uuid.NewString()
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Header.Get("x-satusky-api-key") != "test-token" {
			t.Errorf("missing auth token")
		}
		switch r.URL.Path {
		case "/v1/marketplace-publisher/organizations/" + organizationID + "/releases":
			if r.Method == http.MethodPost {
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Fatal(err)
				}
				if got := r.FormValue("package_name"); got != "demo" {
					t.Errorf("package_name = %q", got)
				}
				file, _, err := r.FormFile("archive")
				if err != nil {
					t.Fatal(err)
				}
				contents, _ := io.ReadAll(file)
				if !bytes.Equal(contents, []byte("archive")) {
					t.Errorf("archive = %q", contents)
				}
				_, _ = io.WriteString(w, artifactEnvelope(releaseID, "private"))
				return
			}
			_, _ = io.WriteString(w, `{"error":false,"data":[{"marketplace_id":"market","release_id":"`+releaseID+`","archive_digest":"sha256:list","visibility":"private"}]}`)
		case "/v1/marketplace-publisher/organizations/" + organizationID + "/releases/" + releaseID:
			if r.Method == http.MethodDelete {
				_, _ = io.WriteString(w, `{"error":false,"data":{"release_id":"`+releaseID+`","deleted_at":"2026-07-28T12:34:56Z","deleted_by":"user-123"}}`)
				return
			}
			_, _ = io.WriteString(w, artifactEnvelope(releaseID, "private"))
		case "/v1/marketplace-publisher/organizations/" + organizationID + "/releases/" + releaseID + "/request-public":
			if !strings.Contains(readBody(t, r), `"reason":"review"`) {
				t.Errorf("request body does not include reason")
			}
			_, _ = io.WriteString(w, artifactEnvelope(releaseID, "public_pending"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupPublisherAPITest(t, server.URL)

	uploaded, err := UploadMarketplacePackageArtifact(organizationID, "demo", []byte("archive"))
	if err != nil || uploaded.ReleaseID != releaseID || uploaded.ArchiveDigest != "sha256:abc" {
		t.Fatalf("UploadMarketplacePackageArtifact() = %#v, %v", uploaded, err)
	}
	if _, err = ListMarketplacePackageArtifacts(organizationID); err != nil {
		t.Fatal(err)
	}
	if _, err = GetMarketplacePackageArtifact(organizationID, releaseID); err != nil {
		t.Fatal(err)
	}
	updated, err := RequestMarketplacePackageArtifactPublic(organizationID, releaseID, "review")
	if err != nil || updated.Visibility != "public_pending" {
		t.Fatalf("RequestMarketplacePackageArtifactPublic() = %#v, %v", updated, err)
	}
	deleted, err := DeleteMarketplacePackageArtifact(organizationID, releaseID)
	if err != nil || deleted.ReleaseID != releaseID || deleted.DeletedBy != "user-123" || deleted.DeletedAt.IsZero() {
		t.Fatalf("DeleteMarketplacePackageArtifact() = %#v, %v", deleted, err)
	}
	if len(requests) != 5 || requests[4] != http.MethodDelete+" /v1/marketplace-publisher/organizations/"+organizationID+"/releases/"+releaseID {
		t.Fatalf("requests = %v, want five publisher calls ending with the release DELETE", requests)
	}
}

func artifactEnvelope(releaseID, visibility string) string {
	return `{"error":false,"data":{"marketplace_id":"market","release_id":"` + releaseID + `","archive_digest":"sha256:abc","visibility":"` + visibility + `"}}`
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
