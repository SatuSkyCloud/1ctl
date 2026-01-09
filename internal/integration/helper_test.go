//go:build integration
// +build integration

package integration

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/deploy"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

var (
	ErrDeploymentFailed  = errors.New("deployment failed")
	ErrDeploymentTimeout = errors.New("timeout waiting for deployment to be ready")
	ErrServiceTimeout    = errors.New("timeout waiting for service to be ready")
	ErrIngressTimeout    = errors.New("timeout waiting for ingress to be ready")
)

// setupTestAuth sets up authentication for tests
func setupTestAuth(t *testing.T) {
	t.Helper()
	token := "test_token"
	if err := context.SetToken(token); err != nil {
		t.Fatalf("Failed to set token: %v", err)
	}
	// Set test user ID for deployment operations
	testUserID := uuid.New().String()
	if err := context.SetUserID(testUserID); err != nil {
		t.Fatalf("Failed to set user ID: %v", err)
	}
	// Set test organization context
	if err := context.SetCurrentOrganization("test-org-id", "Test Org", "test-org"); err != nil {
		t.Fatalf("Failed to set organization: %v", err)
	}
}

// setupTestDockerfile creates a test Dockerfile
func setupTestDockerfile(t *testing.T) string {
	t.Helper()

	// Create testdata directory if it doesn't exist
	testDataDir := filepath.Join("testdata")
	if err := os.MkdirAll(testDataDir, 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}

	dockerfilePath := filepath.Join(testDataDir, "Dockerfile")
	dockerfileContent := `FROM golang:1.21-alpine
WORKDIR /app
COPY . .
CMD ["./app"]`

	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		t.Fatalf("Failed to create test Dockerfile: %v", err)
	}

	return dockerfilePath
}

// cleanupDeployment helps clean up test deployments
func cleanupDeployment(t *testing.T, deploymentID string) {
	t.Helper()
	if err := api.DeleteDeployment(nil, deploymentID); err != nil {
		t.Logf("Warning: Failed to cleanup deployment %s: %v", deploymentID, err)
	}
}

// createTestDeployment creates a deployment for testing
func createTestDeployment(t *testing.T) uuid.UUID {
	t.Helper()
	dockerfilePath := setupTestDockerfile(t)

	opts := deploy.DeploymentOptions{
		CPU:            "100m",
		Memory:         "128Mi",
		Organization:   "test-project",
		Port:           8080,
		DockerfilePath: dockerfilePath,
	}

	resp, err := deploy.Deploy(opts)
	if err != nil {
		t.Fatalf("Failed to create test deployment: %v", err)
	}

	// Wait for deployment to be ready
	status, err := waitForDeployment(t, resp.DeploymentID.String())
	if err != nil {
		t.Fatalf("Test deployment failed to become ready: %v", err)
	}

	if status.Status != api.StatusCompleted {
		t.Fatalf("Test deployment failed to complete, status: %s", status.Status)
	}

	return resp.DeploymentID
}

// createTestService creates a service for testing
func createTestService(t *testing.T, deploymentID uuid.UUID) string {
	t.Helper()
	service := api.Service{
		DeploymentID: deploymentID,
		ServiceName:  "test-service",
		Namespace:    "test-namespace",
		Port:         8080,
	}

	var serviceID string
	err := api.UpsertService(service, &serviceID)
	if err != nil {
		t.Fatalf("Failed to create test service: %v", err)
	}

	return serviceID
}

// cleanupService helps clean up test services
func cleanupService(t *testing.T, serviceID string) {
	t.Helper()
	if err := api.DeleteService(nil, serviceID); err != nil {
		t.Logf("Warning: Failed to cleanup service %s: %v", serviceID, err)
	}
}
