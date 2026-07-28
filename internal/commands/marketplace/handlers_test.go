package marketplace

import (
	stdcontext "context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cliContext "1ctl/internal/context"
	"1ctl/internal/utils"
	"github.com/google/uuid"
)

func setupMarketplaceHandlerTest(t *testing.T, apiURL string) {
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
	if err := cliContext.SetCurrentNamespace("tenant-a"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SATUSKY_API_URL", apiURL+"/v1/cli")
}

func TestMarketplaceDeployUsesDeployableAsAuthority(t *testing.T) {
	t.Run("coming soon but deployable is allowed", func(t *testing.T) {
		marketplaceID := uuid.NewString()
		var creates int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v1/marketplaces/all":
				_, _ = io.WriteString(w, `{"error":false,"data":[{"marketplace_id":"`+marketplaceID+`","marketplace_name":"demo","deployable":true,"coming_soon":true}]}`)
			case "/v1/marketplaces/deploy/create/tenant-a/" + marketplaceID:
				creates++
				_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+uuid.NewString()+`","app_label":"demo","domain":"demo.example.com","status":"deploying"}}`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		setupMarketplaceHandlerTest(t, server.URL)

		if err := handleMarketplaceDeploy(stdcontext.Background(), marketplaceDeployInput{AppName: "demo"}); err != nil {
			t.Fatalf("handleMarketplaceDeploy() error = %v", err)
		}
		if creates != 1 {
			t.Fatalf("create calls = %d, want 1", creates)
		}
	})

	t.Run("untrusted app is rejected before create with its deployability code", func(t *testing.T) {
		marketplaceID := uuid.NewString()
		var creates int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/marketplaces/all" {
				_, _ = io.WriteString(w, `{"error":false,"data":[{"marketplace_id":"`+marketplaceID+`","marketplace_name":"wordpress","deployable":false,"deployability_code":"PACKAGE_TRUST_INVALID","coming_soon":false}]}`)
				return
			}
			creates++
			http.NotFound(w, r)
		}))
		defer server.Close()
		setupMarketplaceHandlerTest(t, server.URL)

		err := handleMarketplaceDeploy(stdcontext.Background(), marketplaceDeployInput{AppName: "wordpress"})
		var diagnostic *utils.LocalDiagnosticError
		if !errors.As(err, &diagnostic) {
			t.Fatalf("error type = %T, want *utils.LocalDiagnosticError", err)
		}
		if diagnostic.Code != "PACKAGE_TRUST_INVALID" || !strings.Contains(diagnostic.Message, "deployment is unavailable") {
			t.Fatalf("diagnostic = %+v", diagnostic)
		}
		if creates != 0 {
			t.Fatalf("create calls = %d, want 0", creates)
		}
	})

	t.Run("non deployable legacy app uses fallback code", func(t *testing.T) {
		marketplaceID := uuid.NewString()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/marketplaces/all" {
				_, _ = io.WriteString(w, `{"error":false,"data":[{"marketplace_id":"`+marketplaceID+`","marketplace_name":"legacy","deployable":false}]}`)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()
		setupMarketplaceHandlerTest(t, server.URL)

		err := handleMarketplaceDeploy(stdcontext.Background(), marketplaceDeployInput{AppName: "legacy"})
		var diagnostic *utils.LocalDiagnosticError
		if !errors.As(err, &diagnostic) || diagnostic.Code != "FEATURE_NOT_AVAILABLE" {
			t.Fatalf("error = %v, want FEATURE_NOT_AVAILABLE local diagnostic", err)
		}
		if got := diagnostic.Remediation; len(got) != 1 || got[0] != "This marketplace app is not currently available for deployment." {
			t.Fatalf("remediation = %q", got)
		}
	})
}

func TestMarketplaceUnavailableTextOutputIncludesDeployabilityCode(t *testing.T) {
	marketplaceID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/marketplaces/all":
			_, _ = io.WriteString(w, `{"error":false,"data":[{"marketplace_id":"`+marketplaceID+`","marketplace_name":"wordpress","deployable":false,"deployability_code":"PACKAGE_TRUST_INVALID"}]}`)
		case "/v1/marketplaces/id/" + marketplaceID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"marketplace_id":"`+marketplaceID+`","marketplace_name":"wordpress","deployable":false,"deployability_code":"PACKAGE_TRUST_INVALID"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupMarketplaceHandlerTest(t, server.URL)

	for _, run := range []func() error{
		func() error { return handleMarketplaceList(stdcontext.Background(), marketplaceListInput{}) },
		func() error { return handleMarketplaceGet(stdcontext.Background(), marketplaceID) },
	} {
		output, err := captureMarketplaceStdout(t, run)
		if err != nil {
			t.Fatalf("marketplace handler error = %v", err)
		}
		for _, want := range []string{"Unavailable", "Availability code", "PACKAGE_TRUST_INVALID"} {
			if !strings.Contains(output, want) {
				t.Fatalf("output %q does not contain %q", output, want)
			}
		}
	}
}

func captureMarketplaceStdout(t *testing.T, run func() error) (string, error) {
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

func TestMarketplaceGetPrintsResolvedAppAsJSON(t *testing.T) {
	marketplaceID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/marketplaces/all" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"error":false,"data":[{"marketplace_id":"`+marketplaceID+`","marketplace_name":"demo","description":"A demo app","deployable":true}]}`)
	}))
	defer server.Close()
	setupMarketplaceHandlerTest(t, server.URL)

	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })
	output, err := captureMarketplaceStdout(t, func() error {
		return handleMarketplaceGet(stdcontext.Background(), "demo")
	})
	if err != nil {
		t.Fatalf("marketplace handler error = %v", err)
	}
	if !strings.Contains(output, `"marketplace_name": "demo"`) {
		t.Fatalf("JSON output = %q", output)
	}
	if strings.Contains(output, "Marketplace App:") {
		t.Fatalf("table output leaked into JSON mode: %q", output)
	}
}
