package marketplace

import (
	stdcontext "context"
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

	t.Run("non deployable is rejected before create", func(t *testing.T) {
		marketplaceID := uuid.NewString()
		var creates int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/marketplaces/all" {
				_, _ = io.WriteString(w, `{"error":false,"data":[{"marketplace_id":"`+marketplaceID+`","marketplace_name":"demo","deployable":false,"coming_soon":false}]}`)
				return
			}
			creates++
			http.NotFound(w, r)
		}))
		defer server.Close()
		setupMarketplaceHandlerTest(t, server.URL)

		err := handleMarketplaceDeploy(stdcontext.Background(), marketplaceDeployInput{AppName: "demo"})
		if err == nil || !strings.Contains(err.Error(), "not deployable") {
			t.Fatalf("error = %v, want not deployable", err)
		}
		if creates != 0 {
			t.Fatalf("create calls = %d, want 0", creates)
		}
	})
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
	output := captureMarketplaceStdout(t, func() error {
		return handleMarketplaceGet(stdcontext.Background(), "demo")
	})
	if !strings.Contains(output, `"marketplace_name": "demo"`) {
		t.Fatalf("JSON output = %q", output)
	}
	if strings.Contains(output, "Marketplace App:") {
		t.Fatalf("table output leaked into JSON mode: %q", output)
	}
}

func captureMarketplaceStdout(t *testing.T, fn func() error) string {
	t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = write
	t.Cleanup(func() { os.Stdout = original })
	if err := fn(); err != nil {
		t.Fatal(err)
	}
	if err := write.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = original
	output, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if err := read.Close(); err != nil {
		t.Fatal(err)
	}
	return string(output)
}
