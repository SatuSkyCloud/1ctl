package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCreateDeploymentIntentContract(t *testing.T) {
	deploymentID := uuid.NewString()
	requestID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/cli/deployments/intent" {
			t.Fatalf("request = %s %s, want POST /v1/cli/deployments/intent", r.Method, r.URL.Path)
		}
		if got := r.Header.Get(requestIDHeader); got != requestID {
			t.Fatalf("%s = %q, want %q", requestIDHeader, got, requestID)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		config, ok := body["config"].(map[string]any)
		if !ok {
			t.Fatalf("config = %#v", body["config"])
		}
		secrets := config["required_secrets"].([]any)
		if len(secrets) != 1 || secrets[0].(map[string]any)["key"] != "DATABASE_URL" || strings.Contains(fmt.Sprint(body), "secret-value") {
			t.Fatalf("required_secrets = %#v; body must not contain a secret value", secrets)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprintf(w, `{"operation_id":%q,"deployment_id":%q,"namespace":"tenant-a","app_label":"api","generation":4,"config_revision":2,"config_sha256":"digest","state":"pending","status_url":"/v1/cli/deployments/id/%s","missing_required_secrets":["DATABASE_URL"]}`,
			requestID, deploymentID, deploymentID)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	accepted, err := CreateDeploymentIntent(DeploymentIntent{
		Deployment:  Deployment{Namespace: "tenant-a", AppLabel: "api", Image: "ghcr.io/acme/api:v1", Port: 8080},
		Environment: []KeyValuePair{{Key: "LOG_LEVEL", Value: "info"}},
		Config: DeploymentDesiredStateConfig{
			ReadinessProbe:  &DeploymentProbe{TCPSocket: &DeploymentTCPSocketProbe{Port: 8080}},
			RequiredSecrets: []DeploymentRequiredSecret{{Key: "DATABASE_URL"}},
		},
		Volumes:     []DeploymentIntentVolume{{VolumeName: "data", ClaimName: "data-pvc", StorageClass: "ceph-block", StorageSize: "1Gi", MountPath: "/data"}},
		Service:     &DeploymentIntentService{Name: "api-service", Port: 8080},
		PublicRoute: &DeploymentIntentPublicRoute{Kind: "default_dns"},
	}, requestID)
	if err != nil {
		t.Fatalf("CreateDeploymentIntent: %v", err)
	}
	if accepted.State != "pending" || accepted.DeploymentID != deploymentID || accepted.Generation != 4 || accepted.MissingRequiredSecrets[0] != "DATABASE_URL" {
		t.Fatalf("accepted = %+v", accepted)
	}
}

func TestCreateDeploymentIntentRetriesTransientFailuresWithSameRequestID(t *testing.T) {
	requestID := uuid.NewString()
	attempts := 0
	var ids []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		ids = append(ids, r.Header.Get(requestIDHeader))
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprint(w, `{"message":"try again"}`)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprint(w, `{"operation_id":"op","deployment_id":"dep","state":"pending"}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")
	if _, err := CreateDeploymentIntent(DeploymentIntent{Deployment: Deployment{Namespace: "ns", AppLabel: "app", Image: "repo/app:v1"}}, requestID); err != nil {
		t.Fatalf("CreateDeploymentIntent: %v", err)
	}
	if attempts != 2 || ids[0] == "" || ids[0] != ids[1] {
		t.Fatalf("attempts=%d request IDs=%v, want two equal non-empty IDs", attempts, ids)
	}
}

func TestCreateDeploymentIntentDoesNotRetryConflict(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusConflict)
		_, _ = fmt.Fprint(w, `{"message":"request ID conflict"}`)
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")
	if _, err := CreateDeploymentIntent(DeploymentIntent{Deployment: Deployment{Namespace: "ns", AppLabel: "app", Image: "repo/app:v1"}}, uuid.NewString()); err == nil {
		t.Fatal("CreateDeploymentIntent error = nil, want conflict")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}
