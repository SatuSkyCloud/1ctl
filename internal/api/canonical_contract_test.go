package api

import (
	cliContext "1ctl/internal/context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestMarketplaceCanonicalRequests(t *testing.T) {
	marketplaceID := uuid.NewString()
	organizationID := uuid.NewString()
	namespace := "tenant-a"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-satusky-api-key"); got != "test-token" {
			t.Errorf("auth header = %q, want test-token", got)
		}
		if got := r.Header.Get("x-satusky-organization-id"); got != organizationID {
			t.Errorf("marketplace organization header = %q, want %q", got, organizationID)
		}
		switch r.URL.Path {
		case "/v1/marketplaces/all":
			if r.Method != http.MethodGet || r.URL.RawQuery != "limit=25&offset=2&sortBy=name" {
				t.Errorf("list request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"error":false,"data":[]}`)
		case "/v1/marketplaces/id/" + marketplaceID:
			if r.Method != http.MethodGet {
				t.Errorf("detail method = %s, want GET", r.Method)
			}
			_, _ = io.WriteString(w, `{"error":false,"data":{"marketplace_id":"`+marketplaceID+`","marketplace_name":"app","deployable":true}}`)
		case "/v1/marketplaces/deploy/create/" + namespace + "/" + marketplaceID:
			if r.Method != http.MethodPost {
				t.Errorf("deploy method = %s, want POST", r.Method)
			}
			var got map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode deploy body: %v", err)
			}
			want := map[string]interface{}{
				"deployment_name": "demo",
				"hostnames":       []interface{}{"machine-a"},
				"cpu_request":     "500m",
				"memory_request":  "1Gi",
				"storage_size":    "10Gi",
			}
			if len(got) != len(want) {
				t.Errorf("deploy body = %v, want exactly %v", got, want)
			}
			for key, value := range want {
				if got[key] == nil || !reflect.DeepEqual(got[key], value) {
					t.Errorf("deploy body[%q] = %v, want %v", key, got[key], value)
				}
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+uuid.NewString()+`","status":"deploying"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")
	if err := cliContext.SetCurrentOrgID(organizationID); err != nil {
		t.Fatal(err)
	}

	if _, err := GetMarketplaceApps(25, 2, "name"); err != nil {
		t.Fatalf("GetMarketplaceApps() error = %v", err)
	}
	if _, err := GetMarketplaceApp(marketplaceID); err != nil {
		t.Fatalf("GetMarketplaceApp() error = %v", err)
	}
	if _, err := DeployMarketplaceApp(namespace, marketplaceID, MarketplaceDeployRequest{
		DeploymentName: "demo", Hostnames: []string{"machine-a"}, CPURequest: "500m",
		MemoryRequest: "1Gi", StorageSize: "10Gi",
	}); err != nil {
		t.Fatalf("DeployMarketplaceApp() error = %v", err)
	}
}

func TestAccountAndDeletionCanonicalRequests(t *testing.T) {
	userID := uuid.NewString()
	deploymentID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-satusky-api-key"); got != "test-token" {
			t.Errorf("auth header = %q, want test-token", got)
		}
		switch r.URL.Path {
		case "/v1/users/" + userID:
			if r.Method != http.MethodPut {
				t.Errorf("update method = %s, want PUT", r.Method)
			}
			var got UpdateUserRequest
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode update body: %v", err)
			}
			if got != (UpdateUserRequest{Name: "Ada", Email: "ada@example.com"}) {
				t.Errorf("update body = %+v", got)
			}
			_, _ = io.WriteString(w, `{"error":false,"data":{"user_id":"`+userID+`","email":"ada@example.com"}}`)
		case "/v1/auth/change-password":
			if r.Method != http.MethodPost {
				t.Errorf("password method = %s, want POST", r.Method)
			}
			var got ChangePasswordRequest
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode password body: %v", err)
			}
			if got != (ChangePasswordRequest{CurrentPassword: "old", NewPassword: "new-password"}) {
				t.Errorf("password body = %+v", got)
			}
		case "/v1/auth/revoke-all":
			if r.Method != http.MethodPost {
				t.Errorf("revoke method = %s, want POST", r.Method)
			}
		case "/v1/deployments/" + deploymentID:
			if r.Method != http.MethodDelete {
				t.Errorf("delete method = %s, want DELETE", r.Method)
			}
			_, _ = io.WriteString(w, `{"error":false,"data":{"deleted_deployments":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	if _, err := UpdateUser(userID, UpdateUserRequest{Name: "Ada", Email: "ada@example.com"}); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if err := ChangePassword("old", "new-password"); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if err := RevokeAllSessions(); err != nil {
		t.Fatalf("RevokeAllSessions() error = %v", err)
	}
	if got := cliContext.GetToken(); got != "" {
		t.Fatalf("token after revoke = %q, want empty", got)
	}

	// Re-seed auth state because revoke intentionally clears it.
	if err := cliContext.SetToken("test-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := DeleteDeployment(deploymentID); err != nil {
		t.Fatalf("DeleteDeployment() error = %v", err)
	}
}
