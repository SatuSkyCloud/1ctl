package deploy

import (
	stdcontext "context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"1ctl/internal/api"
	cliContext "1ctl/internal/context"
	"1ctl/internal/utils"

	"github.com/google/uuid"
)

func TestDeploymentStatusUsesCanonicalRecordForMarketplaceDeployments(t *testing.T) {
	deploymentID := uuid.NewString()
	var lookupCalls, canonicalCalls, legacyStatusCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/namespace/tenant-a/app/pocketbase":
			lookupCalls++
			_, _ = io.WriteString(w, marketplaceDeploymentJSON(deploymentID))
		case "/v1/cli/deployments/id/" + deploymentID:
			canonicalCalls++
			_, _ = io.WriteString(w, marketplaceDeploymentJSON(deploymentID))
		case "/v1/cli/deployments/status/" + deploymentID:
			legacyStatusCalls++
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)

	if err := handleDeploymentStatus(stdcontext.Background(), StatusInput{App: "pocketbase"}); err != nil {
		t.Fatalf("marketplace status by name: %v", err)
	}
	if err := handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID}); err != nil {
		t.Fatalf("marketplace status by deployment ID: %v", err)
	}
	if lookupCalls != 1 || canonicalCalls != 2 || legacyStatusCalls != 0 {
		t.Fatalf("marketplace status calls lookup=%d canonical=%d legacy=%d; want 1, 2, 0", lookupCalls, canonicalCalls, legacyStatusCalls)
	}
}

func TestDeploymentStatusRendersLiveApplicationReadinessInTableAndJSON(t *testing.T) {
	deploymentID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"direct","namespace":"tenant-a","status":"ready","source":"generic"}}`)
		case "/v1/cli/deployments/status/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"status":"Running","progress":100}}`)
		case "/v1/deployments/" + deploymentID + "/status/live":
			_, _ = io.WriteString(w, `{"replica_status":"complete","readiness":{"reconciliation":{"state":"current"},"workload":{"state":"available"},"application":{"basis":"readiness_probe","state":"verified"},"route":{"basis":"gateway_httproute","state":"verified","code":"ROUTE_ACCEPTED"}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)

	table := captureDeploymentStatusOutput(t, "table", func() error {
		return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
	})
	if !strings.Contains(table, "Application readiness") || !strings.Contains(table, "verified (readiness_probe)") {
		t.Fatalf("table output missing application readiness: %s", table)
	}
	if !strings.Contains(table, "Route readiness") || !strings.Contains(table, "verified (gateway_httproute); ROUTE_ACCEPTED") {
		t.Fatalf("table output missing accepted route readiness: %s", table)
	}

	jsonOutput := captureDeploymentStatusOutput(t, "json", func() error {
		return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
	})
	var output map[string]any
	if err := json.Unmarshal([]byte(jsonOutput), &output); err != nil {
		t.Fatalf("status JSON invalid: %v\n%s", err, jsonOutput)
	}
	status := output["status"].(map[string]any)
	readiness, ok := status["readiness"].(map[string]any)
	if !ok || readiness["application"].(map[string]any)["state"] != "verified" {
		t.Fatalf("status readiness = %#v", status["readiness"])
	}
	route := readiness["route"].(map[string]any)
	if route["state"] != "verified" || route["code"] != "ROUTE_ACCEPTED" {
		t.Fatalf("route readiness JSON = %#v", route)
	}
}

func TestDeploymentStatusRendersRouteFailureAndUnknownEvidence(t *testing.T) {
	tests := []struct {
		name        string
		state       string
		code        string
		remediation string
	}{
		{name: "rejected", state: "failing", code: "ROUTE_REJECTED", remediation: "Review the HTTPRoute parent acceptance and listener configuration."},
		{name: "unresolved references", state: "failing", code: "ROUTE_UNRESOLVED_REFS", remediation: "Verify the HTTPRoute backend service and cross-namespace references."},
		{name: "unknown", state: "unknown", code: "ROUTE_STALE", remediation: "Wait for the Gateway controller to publish route conditions."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := uuid.NewString()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/cli/deployments/id/" + deploymentID:
					_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"direct","namespace":"tenant-a","status":"ready","source":"generic"}}`)
				case "/v1/cli/deployments/status/" + deploymentID:
					_, _ = io.WriteString(w, `{"error":false,"data":{"status":"Running","progress":100}}`)
				case "/v1/deployments/" + deploymentID + "/status/live":
					_, _ = io.WriteString(w, `{"readiness":{"route":{"basis":"gateway_httproute","state":"`+tt.state+`","code":"`+tt.code+`","remediation":"`+tt.remediation+`"}}}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			setupDeploymentStatusTest(t, server.URL)

			table := captureDeploymentStatusOutput(t, "table", func() error {
				return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
			})
			for _, want := range []string{"Route readiness", tt.state + " (gateway_httproute)", tt.code, "remediation: " + tt.remediation} {
				if !strings.Contains(table, want) {
					t.Fatalf("table output missing %q: %s", want, table)
				}
			}

			jsonOutput := captureDeploymentStatusOutput(t, "json", func() error {
				return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
			})
			var output map[string]any
			if err := json.Unmarshal([]byte(jsonOutput), &output); err != nil {
				t.Fatalf("status JSON invalid: %v\n%s", err, jsonOutput)
			}
			route := output["status"].(map[string]any)["readiness"].(map[string]any)["route"].(map[string]any)
			if route["code"] != tt.code || route["remediation"] != tt.remediation {
				t.Fatalf("route readiness JSON = %#v", route)
			}
		})
	}
}

func TestDeploymentStatusRendersDNSConditionInTableAndJSON(t *testing.T) {
	deploymentID := uuid.NewString()
	ingressID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"direct","namespace":"tenant-a","status":"ready","source":"generic"}}`)
		case "/v1/cli/deployments/status/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"status":"Running","progress":100}}`)
		case "/v1/cli/ingresses/deploymentId/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"ingress_id":"`+ingressID+`","deployment_id":"`+deploymentID+`","domain_name":"app.example.com"}}`)
		case "/v1/cli/ingresses/" + ingressID + "/domain-status":
			_, _ = io.WriteString(w, `{"error":false,"data":{"dns":{"status":"resolved","condition":{"status":"verified","code":"DNS_VERIFIED","checked_at":"2026-07-28T10:00:00Z","observed_at":"2026-07-28T09:59:00Z"}}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)

	table := captureDeploymentStatusOutput(t, "table", func() error {
		return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
	})
	for _, want := range []string{"DNS", "DNS condition", "verified", "DNS_VERIFIED", "checked 2026-07-28T10:00:00Z", "observed 2026-07-28T09:59:00Z"} {
		if !strings.Contains(table, want) {
			t.Fatalf("table output missing %q: %s", want, table)
		}
	}

	jsonOutput := captureDeploymentStatusOutput(t, "json", func() error {
		return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
	})
	var output map[string]any
	if err := json.Unmarshal([]byte(jsonOutput), &output); err != nil {
		t.Fatalf("status JSON invalid: %v\n%s", err, jsonOutput)
	}
	domainStatus := output["domain_status"].(map[string]any)
	condition := domainStatus["dns"].(map[string]any)["condition"].(map[string]any)
	if condition["status"] != "verified" || condition["code"] != "DNS_VERIFIED" || condition["checked_at"] != "2026-07-28T10:00:00Z" {
		t.Fatalf("DNS condition JSON = %#v", condition)
	}
}

func TestDomainDNSConditionText(t *testing.T) {
	checkedAt := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	observedAt := checkedAt.Add(-time.Minute)
	text := domainDNSConditionText(api.DNSCondition{
		Status:     api.DNSConditionStatusVerified,
		Code:       "DNS_VERIFIED",
		CheckedAt:  &checkedAt,
		ObservedAt: &observedAt,
	})
	for _, want := range []string{"verified", "DNS_VERIFIED", "checked 2026-07-28T10:00:00Z", "observed 2026-07-28T09:59:00Z"} {
		if !strings.Contains(text, want) {
			t.Fatalf("condition text missing %q: %s", want, text)
		}
	}
}

func TestDeploymentStatusRendersMissingLiveReadinessConservatively(t *testing.T) {
	deploymentID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"direct","namespace":"tenant-a","status":"ready","source":"generic"}}`)
		case "/v1/cli/deployments/status/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"status":"Running","progress":100}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)

	table := captureDeploymentStatusOutput(t, "table", func() error {
		return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
	})
	if !strings.Contains(table, "unknown (backend did not provide readiness conditions)") {
		t.Fatalf("table output did not expose missing readiness: %s", table)
	}
}

func TestDeploymentStatusDoesNotInferRouteReadinessWhenLiveEndpointLacksRouteEvidence(t *testing.T) {
	deploymentID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"direct","namespace":"tenant-a","status":"ready","source":"generic"}}`)
		case "/v1/cli/deployments/status/" + deploymentID:
			_, _ = io.WriteString(w, `{"error":false,"data":{"status":"Running","progress":100}}`)
		case "/v1/deployments/" + deploymentID + "/status/live":
			_, _ = io.WriteString(w, `{"readiness":{"application":{"basis":"readiness_probe","state":"verified"}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)

	table := captureDeploymentStatusOutput(t, "table", func() error {
		return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
	})
	if strings.Contains(table, "Route readiness") {
		t.Fatalf("table output inferred route readiness from missing evidence: %s", table)
	}

	jsonOutput := captureDeploymentStatusOutput(t, "json", func() error {
		return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
	})
	var output map[string]any
	if err := json.Unmarshal([]byte(jsonOutput), &output); err != nil {
		t.Fatalf("status JSON invalid: %v\n%s", err, jsonOutput)
	}
	route := output["status"].(map[string]any)["readiness"].(map[string]any)["route"].(map[string]any)
	if _, ok := route["code"]; ok {
		t.Fatalf("legacy route unexpectedly contains code: %#v", route)
	}
	if _, ok := route["remediation"]; ok {
		t.Fatalf("legacy route unexpectedly contains remediation: %#v", route)
	}
}

func TestDeploymentStatusRendersUnconfiguredAndFailingReadiness(t *testing.T) {
	for _, applicationState := range []string{"unconfigured", "failing"} {
		t.Run(applicationState, func(t *testing.T) {
			deploymentID := uuid.NewString()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/cli/deployments/id/" + deploymentID:
					_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"direct","namespace":"tenant-a","status":"ready","source":"generic"}}`)
				case "/v1/cli/deployments/status/" + deploymentID:
					_, _ = io.WriteString(w, `{"error":false,"data":{"status":"Running","progress":100}}`)
				case "/v1/deployments/" + deploymentID + "/status/live":
					_, _ = io.WriteString(w, `{"replica_status":"running_unverified","readiness":{"reconciliation":{"state":"current"},"workload":{"state":"available"},"application":{"basis":"no_readiness_probe","state":"`+applicationState+`"}}}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			setupDeploymentStatusTest(t, server.URL)

			table := captureDeploymentStatusOutput(t, "table", func() error {
				return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
			})
			if !strings.Contains(table, applicationState+" (no_readiness_probe)") {
				t.Fatalf("table output missing %s readiness: %s", applicationState, table)
			}

			output := captureDeploymentStatusOutput(t, "json", func() error {
				return handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID})
			})
			if !strings.Contains(output, `"state": "`+applicationState+`"`) {
				t.Fatalf("JSON output missing %s readiness: %s", applicationState, output)
			}
		})
	}
}

func captureDeploymentStatusOutput(t *testing.T, format string, fn func() error) string {
	t.Helper()
	originalFormat := "table"
	if utils.IsJSONOutput() {
		originalFormat = "json"
	}
	utils.SetOutputFormat(format)
	t.Cleanup(func() { utils.SetOutputFormat(originalFormat) })

	originalStdout := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	err = fn()
	_ = write.Close()
	os.Stdout = originalStdout
	if err != nil {
		t.Fatal(err)
	}
	output, readErr := io.ReadAll(read)
	_ = read.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(output)
}

func TestDeploymentStatusKeepsLegacyStatusEndpointForDirectDeployments(t *testing.T) {
	deploymentID := uuid.NewString()
	var canonicalCalls, legacyStatusCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/cli/deployments/id/" + deploymentID:
			canonicalCalls++
			_, _ = io.WriteString(w, `{"error":false,"data":{"deployment_id":"`+deploymentID+`","app_label":"direct","namespace":"tenant-a","status":"ready","source":"generic"}}`)
		case "/v1/cli/deployments/status/" + deploymentID:
			legacyStatusCalls++
			_, _ = io.WriteString(w, `{"error":false,"data":{"status":"Running","progress":100}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)

	if err := handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: deploymentID}); err != nil {
		t.Fatalf("direct deployment status: %v", err)
	}
	if canonicalCalls != 1 || legacyStatusCalls != 1 {
		t.Fatalf("direct status calls canonical=%d legacy=%d; want 1, 1", canonicalCalls, legacyStatusCalls)
	}
}

func TestDeploymentStatusDoesNotHideCanonicalNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	setupDeploymentStatusTest(t, server.URL)

	if err := handleDeploymentStatus(stdcontext.Background(), StatusInput{DeploymentID: uuid.NewString()}); err == nil {
		t.Fatal("marketplace-capable status lookup unexpectedly hid canonical not found")
	}
}

func setupDeploymentStatusTest(t *testing.T, apiURL string) {
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

func marketplaceDeploymentJSON(deploymentID string) string {
	return `{"error":false,"data":{"deployment_id":"` + deploymentID + `","app_label":"pocketbase","namespace":"tenant-a","status":"ready","source":"marketplace","marketplace_app_name":"pocketbase"}}`
}
