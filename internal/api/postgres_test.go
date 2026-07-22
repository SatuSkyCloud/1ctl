package api

import (
	cliContext "1ctl/internal/context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateDatabaseUserResponseDecodesReadinessFields(t *testing.T) {
	payload := []byte(`{
		"data": {
			"username": "reporter",
			"secret_name": "cnpg-user-test-reporter"
		},
		"password": "secret",
		"ready": false,
		"reconciliation_status": "pending",
		"readiness_message": "CNPG reconciles managed database roles asynchronously"
	}`)

	var resp CreateDatabaseUserResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if resp.User.Username != "reporter" {
		t.Fatalf("User.Username = %q, want reporter", resp.User.Username)
	}
	if resp.Password != "secret" {
		t.Fatalf("Password = %q, want secret", resp.Password)
	}
	if resp.Ready {
		t.Fatal("Ready = true, want false")
	}
	if resp.ReconciliationStatus != "pending" {
		t.Fatalf("ReconciliationStatus = %q, want pending", resp.ReconciliationStatus)
	}
	if resp.ReadinessMessage == "" {
		t.Fatal("ReadinessMessage is empty")
	}
}

func TestPostgresRequestsUseCanonicalDatabasePaths(t *testing.T) {
	const (
		orgID     = "62338977-2dc7-4698-86c3-3b83d1c4be80"
		namespace = "test-namespace"
		storageID = "storage-123"
		ruleID    = "rule-456"
	)

	tests := []struct {
		name     string
		method   string
		path     string
		response string
		invoke   func() error
	}{
		{
			name: "create", method: http.MethodPost, path: "/v1/cli/databases/create",
			response: `{"error":false,"data":{}}`,
			invoke: func() error {
				_, err := CreatePostgresCluster(PostgresCreateOptions{})
				return err
			},
		},
		{
			name: "list", method: http.MethodGet, path: "/v1/cli/databases/namespace/" + namespace,
			response: `{"error":false,"data":[]}`,
			invoke: func() error {
				_, err := ListPostgresClusters(namespace)
				return err
			},
		},
		{
			name: "get", method: http.MethodGet, path: "/v1/cli/databases/id/" + storageID,
			response: `{"error":false,"data":{}}`,
			invoke: func() error {
				_, err := GetPostgresCluster(storageID)
				return err
			},
		},
		{
			name: "delete", method: http.MethodDelete, path: "/v1/cli/databases/" + storageID,
			invoke: func() error { return DeletePostgresCluster(storageID) },
		},
		{
			name: "redeploy", method: http.MethodPost, path: "/v1/cli/databases/" + storageID + "/redeploy",
			invoke: func() error { return RedeployPostgresCluster(storageID) },
		},
		{
			name: "status", method: http.MethodGet, path: "/v1/cli/databases/" + storageID + "/status",
			response: `{"error":false,"data":{}}`,
			invoke: func() error {
				_, err := GetPostgresStatus(storageID)
				return err
			},
		},
		{
			name: "credentials", method: http.MethodGet, path: "/v1/cli/databases/" + storageID + "/credentials",
			response: `{"error":false,"data":{}}`,
			invoke: func() error {
				_, err := GetPostgresCredentials(storageID)
				return err
			},
		},
		{
			name: "list users", method: http.MethodGet, path: "/v1/cli/databases/" + storageID + "/database-users",
			response: `{"error":false,"data":[]}`,
			invoke: func() error {
				_, err := ListPostgresUsers(storageID)
				return err
			},
		},
		{
			name: "create user", method: http.MethodPost, path: "/v1/cli/databases/" + storageID + "/database-users",
			response: `{"error":false,"data":{},"ready":false}`,
			invoke: func() error {
				_, err := CreatePostgresUser(storageID, CreateDatabaseUserRequest{})
				return err
			},
		},
		{
			name: "delete user", method: http.MethodDelete, path: "/v1/cli/databases/" + storageID + "/database-users/test-user",
			invoke: func() error { return DeletePostgresUser(storageID, "test-user") },
		},
		{
			name: "list firewall rules", method: http.MethodGet, path: "/v1/cli/databases/" + storageID + "/firewall-rules",
			response: `{"error":false,"data":[]}`,
			invoke: func() error {
				_, err := ListPostgresFirewallRules(storageID)
				return err
			},
		},
		{
			name: "create firewall rule", method: http.MethodPost, path: "/v1/cli/databases/" + storageID + "/firewall-rules",
			response: `{"error":false,"data":{}}`,
			invoke: func() error {
				_, err := CreatePostgresFirewallRule(storageID, CreateFirewallRuleRequest{})
				return err
			},
		},
		{
			name: "update firewall rule", method: http.MethodPatch, path: "/v1/cli/databases/" + storageID + "/firewall-rules/" + ruleID,
			response: `{"error":false,"data":{}}`,
			invoke: func() error {
				_, err := UpdatePostgresFirewallRule(storageID, ruleID, UpdateFirewallRuleRequest{})
				return err
			},
		},
		{
			name: "delete firewall rule", method: http.MethodDelete, path: "/v1/cli/databases/" + storageID + "/firewall-rules/" + ruleID,
			invoke: func() error { return DeletePostgresFirewallRule(storageID, ruleID) },
		},
		{
			name: "list storage classes", method: http.MethodGet, path: "/v1/cli/databases/storage-classes",
			response: `{"error":false,"data":[]}`,
			invoke: func() error {
				_, err := ListStorageClasses()
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != tt.method {
					t.Errorf("method = %s, want %s", request.Method, tt.method)
				}
				if request.URL.Path != tt.path {
					t.Errorf("path = %s, want %s", request.URL.Path, tt.path)
				}
				if tt.response != "" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprint(w, tt.response)
				}
			}))
			defer server.Close()

			configureAdminAPITestContext(t, server.URL+"/v1/cli")
			if err := cliContext.SetCurrentOrganization(orgID, "test-org", namespace); err != nil {
				t.Fatalf("SetCurrentOrganization() error = %v", err)
			}
			if err := tt.invoke(); err != nil {
				t.Fatalf("request error = %v", err)
			}
		})
	}
}
