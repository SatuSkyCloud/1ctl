package secret

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
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

func TestReadSecretPairs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(path, []byte("# comment\nTOKEN=value\nEMPTY=\n\n"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := readSecretPairs(path)
	if err != nil {
		t.Fatalf("readSecretPairs() error = %v", err)
	}
	want := []string{"TOKEN=value", "EMPTY="}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readSecretPairs() = %#v, want %#v", got, want)
	}
}

func TestReadSecretPairsRejectsUnsafeFiles(t *testing.T) {
	dir := t.TempDir()
	insecure := filepath.Join(dir, "insecure.env")
	if err := os.WriteFile(insecure, []byte("TOKEN=value\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if _, err := readSecretPairs(insecure); err == nil || !strings.Contains(err.Error(), "owner-only") {
			t.Fatalf("readSecretPairs(insecure) error = %v, want owner-only error", err)
		}
	}

	target := filepath.Join(dir, "target.env")
	if err := os.WriteFile(target, []byte("TOKEN=value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.env")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readSecretPairs(link); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("readSecretPairs(symlink) error = %v, want symbolic-link error", err)
	}
}

func TestReadSecretPairsRedactsMalformedLineContents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.env")
	secretValue := "super-secret-value"
	if err := os.WriteFile(path, []byte("# comment\nDATABASE_PASSWORD "+secretValue+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := readSecretPairs(path)
	if err == nil {
		t.Fatal("readSecretPairs() error = nil, want malformed-line error")
	}
	if !strings.Contains(err.Error(), "line 2") || !strings.Contains(err.Error(), "expected KEY=VALUE") {
		t.Fatalf("readSecretPairs() error = %q, want line number and reason", err.Error())
	}
	if strings.Contains(err.Error(), secretValue) || strings.Contains(err.Error(), "DATABASE_PASSWORD") {
		t.Fatalf("readSecretPairs() error exposed secret file contents: %q", err.Error())
	}
}

func TestReadSecretPairsReportsEmptyKeyWithoutLineContents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.env")
	secretValue := "super-secret-value"
	if err := os.WriteFile(path, []byte("="+secretValue+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := readSecretPairs(path)
	if err == nil || !strings.Contains(err.Error(), "line 1") || !strings.Contains(err.Error(), "key must not be empty") {
		t.Fatalf("readSecretPairs() error = %v, want empty-key line error", err)
	}
	if strings.Contains(err.Error(), secretValue) {
		t.Fatalf("readSecretPairs() error exposed secret value: %q", err.Error())
	}
}

func TestHandleCreateSecretDefersRestartToBackend(t *testing.T) {
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
