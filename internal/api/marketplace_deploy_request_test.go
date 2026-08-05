package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDeployMarketplaceAppUsesSingleBoundedRequest(t *testing.T) {
	if got := marketplaceDeployHTTPClient.Timeout; got != 120*time.Second {
		t.Fatalf("marketplace deploy timeout = %s, want 2m", got)
	}

	originalTransport := marketplaceDeployHTTPClient.Transport
	t.Cleanup(func() { marketplaceDeployHTTPClient.Transport = originalTransport })
	attempts := 0
	var deadlineRemaining time.Duration
	marketplaceDeployHTTPClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		attempts++
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.GetBody != nil {
			t.Error("marketplace create request is replayable; GetBody must be nil")
		}
		deadline, ok := request.Context().Deadline()
		if !ok {
			t.Error("marketplace create request has no deadline")
		} else {
			deadlineRemaining = time.Until(deadline)
		}
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"message":"try again"}`)),
		}, nil
	})

	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	_, err := DeployMarketplaceApp("tenant-a", uuid.NewString(), MarketplaceDeployRequest{DeploymentName: "wordpress"})
	if err == nil {
		t.Fatal("DeployMarketplaceApp() error = nil, want unavailable error")
	}
	if attempts != 1 {
		t.Fatalf("marketplace create attempts = %d, want exactly 1", attempts)
	}
	if deadlineRemaining < 119*time.Second || deadlineRemaining > marketplaceDeployTimeout {
		t.Fatalf("request deadline remaining = %s, want approximately 2m", deadlineRemaining)
	}
}

func TestDeployMarketplaceAppDoesNotFollowRedirect(t *testing.T) {
	createRequests := 0
	redirectRequests := 0
	marketplaceID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/marketplaces/deploy/create/tenant-a/" + marketplaceID:
			createRequests++
			writer.Header().Set("Location", "/redirected")
			writer.WriteHeader(http.StatusTemporaryRedirect)
		case "/redirected":
			redirectRequests++
			writer.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(writer, `{"error":false,"data":{"deployment_id":"`+uuid.NewString()+`","status":"deploying"}}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	configureAdminAPITestContext(t, server.URL+"/v1/cli")

	_, err := DeployMarketplaceApp("tenant-a", marketplaceID, MarketplaceDeployRequest{DeploymentName: "wordpress"})
	if err == nil || !strings.Contains(err.Error(), "307") {
		t.Fatalf("DeployMarketplaceApp() error = %v, want redirect status error", err)
	}
	if createRequests != 1 || redirectRequests != 0 {
		t.Fatalf("create requests = %d, redirected requests = %d; want 1 and 0", createRequests, redirectRequests)
	}
}
