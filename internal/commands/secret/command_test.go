package secret

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func TestFlagsHaveDestination(t *testing.T) {
	walkCommands(Command(), func(cmd *cli.Command) {
		for _, f := range cmd.Flags {
			if !isRequired(f) {
				continue
			}
			if hasNilDestination(f) {
				t.Errorf("command %q: required flag %q has no Destination — value will be lost", cmd.Name, flagNameFrom(f))
			}
		}
	})
}

func TestSecretMetadataKeyCount(t *testing.T) {
	if got := secretMetadataKeyCount(api.Secret{KeyCount: 2, Keys: []string{"A", "B"}}); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
	if got := secretMetadataKeyCount(api.Secret{Keys: []string{"A"}}); got != 1 {
		t.Fatalf("expected fallback count 1, got %d", got)
	}
}

func TestHandleCreateSecretLeavesOrderedRolloutToBackend(t *testing.T) {
	deploymentID := uuid.NewString()
	var upsertCalls, restartCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/secrets/upsert":
			upsertCalls++
			_, _ = io.WriteString(w, `{"error":false,"data":{"app_label":"demo"}}`)
		case "/v1/deployments/" + deploymentID + "/restart":
			restartCalls++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupSecretCommandTest(t, server.URL)

	err := handleCreateSecret(context.Background(), secretCreateInput{
		DeploymentID: deploymentID,
		Name:         "demo",
		KV:           []string{"TOKEN=value"},
	})
	if err != nil {
		t.Fatalf("handleCreateSecret() error = %v", err)
	}
	if upsertCalls != 1 || restartCalls != 0 {
		t.Fatalf("calls: upsert=%d restart=%d, want upsert=1 restart=0", upsertCalls, restartCalls)
	}
}

func setupSecretCommandTest(t *testing.T, serverURL string) {
	t.Helper()
	configDir := filepath.Join(t.TempDir(), ".satusky")
	if err := os.MkdirAll(filepath.Join(configDir, "profiles"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "profiles", "test.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "context.json"), []byte(`{"active_profile":"test"}`), 0600); err != nil {
		t.Fatal(err)
	}
	originalStore := satuskyctx.Default()
	satuskyctx.SetDefault(satuskyctx.NewTestStore(configDir))
	t.Cleanup(func() { satuskyctx.SetDefault(originalStore) })
	if err := satuskyctx.SetToken("test-token"); err != nil {
		t.Fatal(err)
	}
	if err := satuskyctx.SetCurrentNamespace("test"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SATUSKY_API_URL", serverURL+"/v1/cli")
}

func walkCommands(cmd *cli.Command, fn func(*cli.Command)) {
	fn(cmd)
	for _, sub := range cmd.Commands {
		walkCommands(sub, fn)
	}
}

func isRequired(f cli.Flag) bool {
	return reflect.ValueOf(f).Elem().FieldByName("Required").Bool()
}

func hasNilDestination(f cli.Flag) bool {
	dest := reflect.ValueOf(f).Elem().FieldByName("Destination")
	if !dest.IsValid() {
		return true
	}
	return dest.IsNil()
}

func flagNameFrom(f cli.Flag) string {
	return reflect.ValueOf(f).Elem().FieldByName("Name").String()
}
