package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	satuskyctx "1ctl/internal/context"
	"github.com/google/uuid"
)

func TestConvexLifecycleContract(t *testing.T) {
	id := uuid.NewString()
	org := uuid.NewString()
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/cli/databases/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["engine"] != "convex" || body["namespace"] != "tenant-test" || body["organization_id"] != org {
				t.Errorf("incorrect tenancy/engine: %v", body)
			}
			convex, ok := body["convex"].(map[string]any)
			if !ok || convex["instance_name"] != "test" || convex["dashboard_enabled"] != true {
				t.Errorf("incorrect Convex settings: %v", convex)
			}
			if _, ok := convex["s3_access_key"]; ok {
				t.Error("managed creation must omit S3 credentials")
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"data":{"storage_id":%q,"engine":"convex"}}`, id)
		case "/v1/cli/databases/namespace/tenant-test":
			_, _ = fmt.Fprintf(w, `{"data":[{"storage_id":%q,"engine":"convex"},{"engine":"cnpg"}]}`, id)
		case "/v1/cli/databases/id/" + id:
			_, _ = fmt.Fprintf(w, `{"data":{"storage_id":%q,"engine":"convex"}}`, id)
		case "/v1/cli/databases/" + id + "/status":
			_, _ = fmt.Fprint(w, `{"data":{"engine":"convex","status":"ready","database_ready":true,"backend_ready":true,"public_reachability_verified":false}}`)
		case "/v1/cli/databases/" + id + "/credentials":
			_, _ = fmt.Fprint(w, `{"data":{"api_url":"https://api.example.test","instance_secret":"fixture-secret"}}`)
		case "/v1/cli/databases/" + id + "/redeploy", "/v1/cli/databases/" + id:
			_, _ = fmt.Fprint(w, `{"error":false}`)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")
	if err := satuskyctx.SetCurrentOrganization(org, "test", "tenant-test"); err != nil {
		t.Fatal(err)
	}
	created, err := CreateConvex(ConvexCreateOptions{Name: "test", DashboardEnabled: true})
	if err != nil || created.StorageID.String() != id {
		t.Fatalf("create: %v %v", created, err)
	}
	items, err := ListConvex()
	if err != nil || len(items) != 1 {
		t.Fatalf("list: %v %v", items, err)
	}
	if _, err := GetConvex(id); err != nil {
		t.Fatal(err)
	}
	status, err := GetConvexStatus(id)
	if err != nil || status.Status != "ready" || status.PublicReachabilityVerified {
		t.Fatalf("status: %v %v", status, err)
	}
	credentials, err := GetConvexCredentials(id)
	if err != nil || credentials.InstanceSecret != "fixture-secret" {
		t.Fatalf("credentials: %v", err)
	}
	if err := RedeployConvex(id); err != nil {
		t.Fatal(err)
	}
	if err := DeleteConvex(id); err != nil {
		t.Fatal(err)
	}
	want := []string{"POST /v1/cli/databases/create", "GET /v1/cli/databases/namespace/tenant-test", "GET /v1/cli/databases/id/" + id, "GET /v1/cli/databases/" + id + "/status", "GET /v1/cli/databases/" + id + "/credentials", "POST /v1/cli/databases/" + id + "/redeploy", "DELETE /v1/cli/databases/" + id}
	if fmt.Sprint(calls) != fmt.Sprint(want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestGetConvexRejectsOtherEngine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, `{"data":{"engine":"cnpg"}}`) }))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")
	if _, err := GetConvex(uuid.NewString()); err == nil {
		t.Fatal("accepted another engine")
	}
}
