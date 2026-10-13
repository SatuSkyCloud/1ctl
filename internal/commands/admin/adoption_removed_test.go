package admin

import (
	"context"
	"strings"
	"testing"

	"1ctl/internal/api"
)

// TestAdoptionCommandsReportTheRemovedEndpoint pins the fix for the second half of
// satusky-core_backend#633: the adopt endpoints are deliberately unregistered
// (domains/admin/http keeps a static test asserting they stay that way), but these
// commands went on calling them, so every use failed with a bare
// "Cannot POST /admin/deployments/:id/adopt" while --help still advertised it.
//
// The assertions are that the message explains the removal and, more importantly,
// that no request is made at all.
func TestAdoptionCommandsReportTheRemovedEndpoint(t *testing.T) {
	called := false
	originalAdopt, originalRouting := adoptDeployment, adoptDeploymentRouting
	t.Cleanup(func() { adoptDeployment, adoptDeploymentRouting = originalAdopt, originalRouting })

	adoptDeployment = func(string, string, api.DeploymentAdoptionRequest) (*api.DeploymentAdoptionResult, error) {
		called = true
		return nil, nil
	}
	adoptDeploymentRouting = func(string, string, api.DeploymentAdoptionRequest) (*api.DeploymentRoutingAdoptionResult, error) {
		called = true
		return nil, nil
	}

	handlers := map[string]func(context.Context, deploymentAdoptInput) error{
		"adopt":         handleDeploymentAdopt,
		"routing-adopt": handleDeploymentRoutingAdopt,
	}
	for name, handle := range handlers {
		t.Run(name, func(t *testing.T) {
			err := handle(context.Background(), deploymentAdoptInput{
				DeploymentID: "11111111-1111-1111-1111-111111111111",
			})
			if err == nil {
				t.Fatal("adoption must fail: the endpoint it would call is not registered")
			}
			for _, want := range []string{"no longer served by the API", "satusky-core_backend#633"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q must contain %q", err.Error(), want)
				}
			}
		})
	}
	if called {
		t.Fatal("the handlers must not call endpoints that are not registered")
	}
}
