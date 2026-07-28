package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func readinessStatus(application, reconciliation, workload string) DeploymentStatus {
	return DeploymentStatus{Readiness: &DeploymentReadiness{
		Reconciliation: DeploymentReconciliationReadiness{State: reconciliation},
		Workload:       DeploymentWorkloadReadiness{State: workload},
		Application:    DeploymentConditionReadiness{Basis: "readiness_probe", State: application},
	}}
}

func writeLiveReadiness(w http.ResponseWriter, status DeploymentStatus) {
	_, _ = fmt.Fprintf(w, `{"replica_status":%q,"readiness":{"reconciliation":{"state":%q,"generation":2,"observed_generation":2},"workload":{"state":%q,"desired_replicas":1,"ready_replicas":1,"available_replicas":1,"updated_replicas":1},"application":{"basis":"readiness_probe","state":%q}}}`, status.ReplicaStatus, status.Readiness.Reconciliation.State, status.Readiness.Workload.State, status.Readiness.Application.State)
}

func TestGetLiveDeploymentStatusUsesMainAPIAndParsesReadiness(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/deployments/dep-1/status/live" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		writeLiveReadiness(w, readinessStatus("verified", "current", "available"))
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	status, err := GetLiveDeploymentStatus("dep-1")
	if err != nil {
		t.Fatalf("GetLiveDeploymentStatus() error = %v", err)
	}
	if status.Readiness == nil || status.Readiness.Application.State != "verified" || status.Readiness.Workload.State != "available" {
		t.Fatalf("readiness = %+v", status.Readiness)
	}
}

func TestGetLiveDeploymentStatusParsesRouteEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/deployments/dep-1/status/live" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"readiness":{"route":{"basis":"gateway_httproute","state":"failing","code":"ROUTE_UNRESOLVED_REFS","remediation":"Verify the HTTPRoute backend service and cross-namespace references."}}}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	status, err := GetLiveDeploymentStatus("dep-1")
	if err != nil {
		t.Fatalf("GetLiveDeploymentStatus() error = %v", err)
	}
	route := status.Readiness.Route
	if route.Basis != "gateway_httproute" || route.State != "failing" || route.Code != "ROUTE_UNRESOLVED_REFS" || route.Remediation != "Verify the HTTPRoute backend service and cross-namespace references." {
		t.Fatalf("route readiness = %+v", route)
	}
}

func TestDeploymentReadinessEvaluation(t *testing.T) {
	tests := []struct {
		name     string
		status   DeploymentStatus
		mode     DeploymentWaitMode
		ready    bool
		wantCode string
		wantText string
	}{
		{name: "verified application", status: readinessStatus("verified", "current", "available"), mode: DeploymentWaitModeApplication, ready: true},
		{name: "unconfigured is typed", status: readinessStatus("unconfigured", "current", "available"), mode: DeploymentWaitModeApplication, wantCode: "READINESS_UNVERIFIED"},
		{name: "application failing is typed", status: readinessStatus("failing", "current", "available"), mode: DeploymentWaitModeApplication, wantCode: "READINESS_FAILED"},
		{name: "verified application still needs available workload", status: readinessStatus("verified", "current", "progressing"), mode: DeploymentWaitModeApplication, wantText: "waiting for current reconciliation and available workload"},
		{name: "workload mode requires current availability", status: readinessStatus("unknown", "current", "available"), mode: DeploymentWaitModeWorkload, ready: true},
		{name: "missing readiness remains unverified", status: DeploymentStatus{Status: StatusRunningK8s}, mode: DeploymentWaitModeApplication, wantText: "backend did not provide readiness conditions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.readinessEvaluation(tt.mode)
			if got.Ready != tt.ready {
				t.Fatalf("ready = %v, want %v (%+v)", got.Ready, tt.ready, got)
			}
			if tt.wantCode != "" {
				var statusErr *HTTPStatusError
				if !errors.As(got.TerminalError, &statusErr) || statusErr.Code != tt.wantCode || len(statusErr.Remediation) == 0 {
					t.Fatalf("terminal error = %#v, want code %s with remediation", got.TerminalError, tt.wantCode)
				}
			}
			if tt.wantText != "" && got.Reason != tt.wantText {
				t.Fatalf("reason = %q, want %q", got.Reason, tt.wantText)
			}
		})
	}
}

func TestWaitForDeploymentUsesLiveReadinessAndCompatibilityModes(t *testing.T) {
	tests := []struct {
		name         string
		status       *DeploymentStatus
		liveNotFound bool
		mode         DeploymentWaitMode
		verify       func() bool
		wantCode     string
		wantSuccess  bool
	}{
		{name: "default verified", status: ptrReadinessStatus("verified", "current", "available"), wantSuccess: true},
		{name: "workload compatibility", status: ptrReadinessStatus("unknown", "current", "available"), mode: DeploymentWaitModeWorkload, wantSuccess: true},
		{name: "application readiness ignores route failure", status: readinessStatusWithFailingRoute(), wantSuccess: true},
		{name: "default does not accept running unverified", status: ptrReadinessStatus("unknown", "current", "available")},
		{name: "old endpoint does not accept legacy running", liveNotFound: true},
		{name: "failing has stable code", status: ptrReadinessStatus("failing", "current", "available"), wantCode: "READINESS_FAILED"},
		{name: "health verifier only succeeds when positive", liveNotFound: true, verify: func() bool { return true }, wantSuccess: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/deployments/dep-1/status/live" {
					t.Fatalf("unexpected path %s", r.URL.Path)
				}
				if tt.liveNotFound {
					http.NotFound(w, r)
					return
				}
				writeLiveReadiness(w, *tt.status)
			}))
			defer server.Close()
			configureAdminAPITestContext(t, server.URL+"/v1/cli")

			_, err := WaitForDeploymentWithOptions("dep-1", 8*time.Millisecond, DeploymentWaitOptions{Mode: tt.mode, PollInterval: time.Millisecond, VerifyApplication: tt.verify})
			if tt.wantSuccess {
				if err != nil {
					t.Fatalf("WaitForDeploymentWithOptions() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("WaitForDeploymentWithOptions() error = nil")
			}
			if tt.wantCode != "" {
				var statusErr *HTTPStatusError
				if !errors.As(err, &statusErr) || statusErr.Code != tt.wantCode {
					t.Fatalf("error = %#v, want code %s", err, tt.wantCode)
				}
			}
		})
	}
}

func readinessStatusWithFailingRoute() *DeploymentStatus {
	status := ptrReadinessStatus("verified", "current", "available")
	status.Readiness.Route = DeploymentConditionReadiness{
		Basis:       "gateway_httproute",
		State:       "failing",
		Code:        "ROUTE_REJECTED",
		Remediation: "Review the HTTPRoute parent acceptance and listener configuration.",
	}
	return status
}

func ptrReadinessStatus(application, reconciliation, workload string) *DeploymentStatus {
	status := readinessStatus(application, reconciliation, workload)
	status.ReplicaStatus = "running_unverified"
	return &status
}

func TestValidateDeploymentWaitMode(t *testing.T) {
	if _, err := ValidateDeploymentWaitMode("invalid"); err == nil {
		t.Fatal("invalid wait mode accepted")
	}
}
