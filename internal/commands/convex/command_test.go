package convex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"github.com/google/uuid"
)

func TestCommandTree(t *testing.T) {
	want := []string{"create", "list", "get", "status", "connection", "credentials", "redeploy", "delete"}
	cmd := Command()
	if len(cmd.Commands) != len(want) {
		t.Fatalf("unexpected commands: %d", len(cmd.Commands))
	}
	for i, name := range want {
		if cmd.Commands[i].Name != name {
			t.Errorf("missing %s", name)
		}
	}
}

func TestCreateValidation(t *testing.T) {
	valid := api.ConvexCreateOptions{Name: "my-convex", StorageSize: "10Gi", CPURequest: "500m", CPULimit: "2", MemoryRequest: "1Gi", MemoryLimit: "2Gi"}
	if err := validateCreate(valid); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "UPPER", "-bad", "a/b", strings.Repeat("a", 64)} {
		in := valid
		in.Name = name
		if validateCreate(in) == nil {
			t.Errorf("accepted name %q", name)
		}
	}
	for _, size := range []string{"", "0Gi", "-1Gi", "nonsense"} {
		in := valid
		in.StorageSize = size
		if validateCreate(in) == nil {
			t.Errorf("accepted size %q", size)
		}
	}
}

func TestGeneralOutputOmitsLegacySecrets(t *testing.T) {
	var item api.StorageConfig
	if err := json.Unmarshal([]byte(`{"engine":"convex","annotations":{"convex.satusky.com/instance-secret":"secret-value"},"convex":{"instance_name":"safe","s3_secret_key":"secret-value"}}`), &item); err != nil {
		t.Fatal(err)
	}
	sanitize(&item)
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret-value") {
		t.Fatal("general output leaked credentials")
	}
}

func TestJSONDeleteRequiresConfirmationBeforeNetwork(t *testing.T) {
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })
	err := Command().Run(context.Background(), []string{"convex", "delete", uuid.NewString()})
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("got %v", err)
	}
}

func configureTestContext(t *testing.T, endpoint string) {
	t.Helper()
	old := satuskyctx.Default()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "profiles"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profiles", "test.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "context.json"), []byte(`{"active_profile":"test"}`), 0600); err != nil {
		t.Fatal(err)
	}
	satuskyctx.SetDefault(satuskyctx.NewTestStore(dir))
	t.Cleanup(func() { satuskyctx.SetDefault(old) })
	if err := satuskyctx.SetToken("fixture-token"); err != nil {
		t.Fatal(err)
	}
	if err := satuskyctx.SetCurrentOrganization(uuid.NewString(), "test", "test-ns"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SATUSKY_API_URL", endpoint+"/v1/cli")
}

func TestMutationsRejectWrongEngine(t *testing.T) {
	for _, action := range []string{"delete", "redeploy", "credentials"} {
		t.Run(action, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || !strings.Contains(r.URL.Path, "/databases/id/") {
					t.Errorf("unexpected mutation: %s %s", r.Method, r.URL.Path)
				}
				_, _ = fmt.Fprint(w, `{"data":{"engine":"cnpg"}}`)
			}))
			defer server.Close()
			configureTestContext(t, server.URL)
			args := []string{"convex", action}
			if action == "delete" {
				args = append(args, "--yes")
			}
			args = append(args, uuid.NewString())
			if err := Command().Run(context.Background(), args); err == nil {
				t.Fatal("accepted wrong engine")
			}
			if calls != 1 {
				t.Fatalf("requests = %d", calls)
			}
		})
	}
}

func TestAmbiguousNameFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"data":[{"storage_id":%q,"engine":"convex","convex":{"instance_name":"same"}},{"storage_id":%q,"engine":"convex","convex":{"instance_name":"same"}}]}`, uuid.NewString(), uuid.NewString())
	}))
	defer server.Close()
	configureTestContext(t, server.URL)
	if _, err := resolve("same"); err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("got %v", err)
	}
}

func TestConnectionAndCredentialsJSON(t *testing.T) {
	id := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/credentials") {
			_, _ = fmt.Fprint(w, `{"data":{"api_url":"https://api.example.test","dashboard_url":"https://dashboard.example.test","instance_secret":"fixture-only-secret"}}`)
		} else {
			_, _ = fmt.Fprintf(w, `{"data":{"storage_id":%q,"engine":"convex","convex":{"dashboard_enabled":false}}}`, id)
		}
	}))
	defer server.Close()
	configureTestContext(t, server.URL)
	utils.SetOutputFormat("json")
	t.Cleanup(func() { utils.SetOutputFormat("table") })
	for _, action := range []string{"connection", "credentials"} {
		t.Run(action, func(t *testing.T) {
			encoded := captureOutput(t, func() error { return Command().Run(context.Background(), []string{"convex", action, id}) })
			var result map[string]any
			if err := json.Unmarshal([]byte(encoded), &result); err != nil {
				t.Fatalf("not one JSON document: %v", err)
			}
			_, hasSecret := result["instance_secret"]
			if hasSecret != (action == "credentials") {
				t.Fatalf("unexpected secret visibility for %s", action)
			}
			if _, exists := result["dashboard_url"]; exists {
				t.Fatal("disabled dashboard was advertised")
			}
		})
	}
}

func captureOutput(t *testing.T, action func() error) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = original; _ = reader.Close(); _ = writer.Close() }()
	if err := action(); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
