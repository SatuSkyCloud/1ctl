package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetMarketplacePublicAccessPutsEnabledToTheDeployment(t *testing.T) {
	var method, path string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"success":true,"data":{"public_access":false}}`))
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	if err := SetMarketplacePublicAccess("d-123", false); err != nil {
		t.Fatalf("SetMarketplacePublicAccess() error = %v", err)
	}
	if method != http.MethodPut || path != "/v1/marketplaces/deployments/d-123/public-access" {
		t.Fatalf("request = %s %s", method, path)
	}
	if enabled, ok := body["enabled"].(bool); !ok || enabled {
		t.Fatalf("body enabled = %v, want explicit false", body["enabled"])
	}
}

func TestMarketplaceDeployRequestOmitsPublicAccessUnlessAsked(t *testing.T) {
	private, _ := json.Marshal(MarketplaceDeployRequest{DeploymentName: "nats"})
	var privateFields map[string]any
	_ = json.Unmarshal(private, &privateFields)
	if _, present := privateFields["public_access"]; present {
		t.Fatalf("private deploy sends public_access: %s", private)
	}
	public, _ := json.Marshal(MarketplaceDeployRequest{DeploymentName: "nats", PublicAccess: true})
	var publicFields map[string]any
	_ = json.Unmarshal(public, &publicFields)
	if publicFields["public_access"] != true {
		t.Fatalf("public deploy body = %s", public)
	}
}
