package packagecmd

import (
	stdcontext "context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"1ctl/internal/api"
	"1ctl/internal/config"
	cliContext "1ctl/internal/context"
	"1ctl/internal/packageartifact"
	"1ctl/internal/utils"
	"github.com/google/uuid"
)

func setupPackageCommandTest(t *testing.T, apiURL, organizationID string) {
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
	if err := cliContext.SetCurrentOrgID(organizationID); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SATUSKY_API_URL", apiURL+"/v1/cli")
}

func TestPublishDefaultsPrivateAndRequestsPublicOnlyWhenAsked(t *testing.T) {
	organizationID := uuid.NewString()
	releaseID := uuid.NewString()
	var uploadCount, publicCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/marketplace-publisher/organizations/" + organizationID + "/releases":
			uploadCount++
			_, _ = io.WriteString(w, `{"error":false,"data":{"marketplace_id":"market","release_id":"`+releaseID+`","archive_digest":"sha256:abc","visibility":"private"}}`)
		case "/v1/marketplace-publisher/organizations/" + organizationID + "/releases/" + releaseID + "/request-public":
			publicCount++
			_, _ = io.WriteString(w, `{"error":false,"data":{"marketplace_id":"market","release_id":"`+releaseID+`","archive_digest":"sha256:abc","visibility":"public_pending"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupPackageCommandTest(t, server.URL, organizationID)
	archive, _, err := packageartifact.Create(&config.ProjectConfig{
		App:   config.AppConfig{Name: "demo"},
		Build: config.BuildConfig{Image: "registry.example.test/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(t.TempDir(), "chosen-output-name.tar.gz")
	if err := os.WriteFile(artifact, archive, 0600); err != nil {
		t.Fatal(err)
	}
	if err := handlePublish(stdcontext.Background(), publishInput{Artifact: artifact}); err != nil {
		t.Fatal(err)
	}
	if uploadCount != 1 || publicCount != 0 {
		t.Fatalf("private publish calls upload=%d public=%d, want 1 and 0", uploadCount, publicCount)
	}
	if err := handlePublish(stdcontext.Background(), publishInput{Artifact: artifact, Public: true, Reason: "review"}); err != nil {
		t.Fatal(err)
	}
	if uploadCount != 2 || publicCount != 1 {
		t.Fatalf("public publish calls upload=%d public=%d, want 2 and 1", uploadCount, publicCount)
	}
}

func TestCurrentOrganizationIDDoesNotFallBackToNamespace(t *testing.T) {
	originalStore := cliContext.Default()
	store := cliContext.NewTestStore(t.TempDir())
	cliContext.SetDefault(store)
	t.Cleanup(func() { cliContext.SetDefault(originalStore) })
	if _, err := currentOrganizationID(); err == nil {
		t.Fatal("currentOrganizationID() succeeded without authenticated organization ID")
	}
}

func TestCreateJSONOutputIsOneDocument(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "satusky.toml")
	outputPath := filepath.Join(dir, "demo.tar.gz")
	if err := os.WriteFile(configPath, []byte(`[app]
name = "demo"

[build]
image = "registry.example.test/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
`), 0600); err != nil {
		t.Fatal(err)
	}
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })
	output := capturePackageStdout(t, func() error {
		return handleCreate(stdcontext.Background(), createInput{Config: configPath, Output: outputPath})
	})
	var created struct {
		PackageName string `json:"package_name"`
		Output      string `json:"output"`
	}
	if err := json.Unmarshal([]byte(output), &created); err != nil {
		t.Fatalf("create JSON output is invalid: %q: %v", output, err)
	}
	if created.PackageName != "demo" || created.Output != outputPath {
		t.Fatalf("create JSON = %#v", created)
	}
}

func TestPublishJSONOutputIsOneArtifactDocument(t *testing.T) {
	organizationID := uuid.NewString()
	releaseID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/marketplace-publisher/organizations/"+organizationID+"/releases" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"error":false,"data":{"marketplace_id":"market","release_id":"`+releaseID+`","archive_digest":"sha256:abc","visibility":"private"}}`)
	}))
	defer server.Close()
	setupPackageCommandTest(t, server.URL, organizationID)
	archive, _, err := packageartifact.Create(&config.ProjectConfig{
		App:   config.AppConfig{Name: "demo"},
		Build: config.BuildConfig{Image: "registry.example.test/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(t.TempDir(), "demo.tar.gz")
	if err := os.WriteFile(artifact, archive, 0600); err != nil {
		t.Fatal(err)
	}
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })
	output := capturePackageStdout(t, func() error {
		return handlePublish(stdcontext.Background(), publishInput{Artifact: artifact})
	})
	var published struct {
		MarketplaceID string `json:"marketplace_id"`
		ReleaseID     string `json:"release_id"`
		ArchiveDigest string `json:"archive_digest"`
		Visibility    string `json:"visibility"`
	}
	if err := json.Unmarshal([]byte(output), &published); err != nil {
		t.Fatalf("publish JSON output is invalid: %q: %v", output, err)
	}
	if published.MarketplaceID != "market" || published.ReleaseID != releaseID || published.ArchiveDigest != "sha256:abc" || published.Visibility != "private" {
		t.Fatalf("publish JSON = %#v", published)
	}
}

func TestDeleteWithYesOutputsOneTombstoneJSONDocument(t *testing.T) {
	organizationID := uuid.NewString()
	releaseID := uuid.NewString()
	var deleteCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalls++
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/marketplace-publisher/organizations/"+organizationID+"/releases/"+releaseID {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"error":false,"data":{"release_id":"`+releaseID+`","deleted_at":"2026-07-28T12:34:56Z","deleted_by":"user-123"}}`)
	}))
	defer server.Close()
	setupPackageCommandTest(t, server.URL, organizationID)
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })

	output, err := capturePackageStdoutResult(t, func() error {
		return Command().Run(stdcontext.Background(), []string{"package", "delete", releaseID, "--yes"})
	})
	if err != nil {
		t.Fatalf("package delete --yes: %v", err)
	}
	var tombstone struct {
		ReleaseID string `json:"release_id"`
		DeletedAt string `json:"deleted_at"`
		DeletedBy string `json:"deleted_by"`
	}
	decoder := json.NewDecoder(strings.NewReader(output))
	if err := decoder.Decode(&tombstone); err != nil {
		t.Fatalf("delete JSON output is invalid: %q: %v", output, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("delete JSON output contains more than one document: %q", output)
	}
	if tombstone.ReleaseID != releaseID || tombstone.DeletedAt != "2026-07-28T12:34:56Z" || tombstone.DeletedBy != "user-123" {
		t.Fatalf("delete JSON = %#v", tombstone)
	}
	if deleteCalls != 1 {
		t.Fatalf("DELETE calls = %d, want 1", deleteCalls)
	}
}

func TestDeleteInteractiveConfirmationSendsDelete(t *testing.T) {
	organizationID := uuid.NewString()
	releaseID := uuid.NewString()
	var deleteCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalls++
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/marketplace-publisher/organizations/"+organizationID+"/releases/"+releaseID {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"error":false,"data":{"release_id":"`+releaseID+`","deleted_at":"2026-07-28T12:34:56Z","deleted_by":"user-123"}}`)
	}))
	defer server.Close()
	setupPackageCommandTest(t, server.URL, organizationID)
	setPackageStdin(t, "yes\n")

	output, err := capturePackageStdoutResult(t, func() error {
		return handleDelete(stdcontext.Background(), deleteInput{ReleaseID: releaseID})
	})
	if err != nil {
		t.Fatalf("interactive package delete: %v", err)
	}
	if !strings.Contains(output, "archive bytes will be retained and not purged") || !strings.Contains(output, "RELEASE ID") {
		t.Fatalf("interactive delete output = %q, want retention prompt and tombstone table", output)
	}
	if deleteCalls != 1 {
		t.Fatalf("DELETE calls = %d, want 1", deleteCalls)
	}
}

func TestDeleteCancellationDoesNotSendRequest(t *testing.T) {
	organizationID := uuid.NewString()
	var requestCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCalls++
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()
	setupPackageCommandTest(t, server.URL, organizationID)
	setPackageStdin(t, "n\n")

	output, err := capturePackageStdoutResult(t, func() error {
		return handleDelete(stdcontext.Background(), deleteInput{ReleaseID: uuid.NewString()})
	})
	if err == nil || err.Error() != "package release deletion cancelled" {
		t.Fatalf("cancellation error = %v, want deterministic cancellation error", err)
	}
	if requestCalls != 0 {
		t.Fatalf("request calls = %d, want 0 after cancellation", requestCalls)
	}
	if strings.Contains(output, "RELEASE ID") {
		t.Fatalf("cancellation output includes success table: %q", output)
	}
}

func TestDeleteInvalidReleaseIDDoesNotSendRequest(t *testing.T) {
	organizationID := uuid.NewString()
	var requestCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCalls++
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()
	setupPackageCommandTest(t, server.URL, organizationID)

	err := handleDelete(stdcontext.Background(), deleteInput{ReleaseID: "not-a-uuid", Yes: true})
	if err == nil || err.Error() != "release ID is invalid; provide a valid UUID" {
		t.Fatalf("invalid release ID error = %v", err)
	}
	if requestCalls != 0 {
		t.Fatalf("request calls = %d, want 0 for invalid release ID", requestCalls)
	}
}

func TestDeletePreservesRemoteConflictWithoutSuccessOutput(t *testing.T) {
	organizationID := uuid.NewString()
	releaseID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/marketplace-publisher/organizations/"+organizationID+"/releases/"+releaseID {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":true,"code":"CONFLICT","message":"release is in use","details":{"release_id":"`+releaseID+`"},"retryable":false,"remediation":["remove dependent deployments before deleting the release"]}`)
	}))
	defer server.Close()
	setupPackageCommandTest(t, server.URL, organizationID)
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })

	output, err := capturePackageStdoutResult(t, func() error {
		return handleDelete(stdcontext.Background(), deleteInput{ReleaseID: releaseID, Yes: true})
	})
	var statusErr *api.HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusConflict || statusErr.Code != "CONFLICT" || len(statusErr.Remediation) != 1 {
		t.Fatalf("delete error = %#v, want typed conflict with remediation", err)
	}
	if output != "" {
		t.Fatalf("remote conflict printed unexpected stdout: %q", output)
	}
}

func capturePackageStdout(t *testing.T, run func() error) string {
	t.Helper()
	output, err := capturePackageStdoutResult(t, run)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func capturePackageStdoutResult(t *testing.T, run func() error) (string, error) {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	runErr := run()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = original
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(output), runErr
}

func setPackageStdin(t *testing.T, input string) {
	t.Helper()
	original := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(writer, input); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = original
		_ = reader.Close()
	})
}
