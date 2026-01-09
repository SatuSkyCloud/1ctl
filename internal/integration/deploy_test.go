//go:build integration
// +build integration

package integration

import (
	"1ctl/internal/api"
	"1ctl/internal/deploy"
	"testing"
	"time"
)

func TestDeploymentFlow(t *testing.T) {
	setupTestAuth(t)
	dockerfilePath := setupTestDockerfile(t)

	t.Run("create and manage deployment", func(t *testing.T) {
		opts := deploy.DeploymentOptions{
			CPU:            "100m",
			Memory:         "128Mi",
			Organization:   "test-project",
			Port:           8080,
			DockerfilePath: dockerfilePath,
		}

		// Create deployment
		resp, err := deploy.Deploy(opts)
		if err != nil {
			t.Fatalf("Deploy() error = %v", err)
		}

		// Clean up deployment after test
		defer cleanupDeployment(t, resp.DeploymentID.String())

		t.Logf("Created deployment: %s", resp.DeploymentID)

		// Wait for deployment to be ready
		status, err := waitForDeployment(t, resp.DeploymentID.String())
		if err != nil {
			t.Fatalf("Waiting for deployment failed: %v", err)
		}

		if status.Status != api.StatusCompleted {
			t.Errorf("Deployment status = %s, want %s", status.Status, api.StatusCompleted)
		}

		// Test deployment listing
		deployments, err := api.ListDeployments()
		if err != nil {
			t.Errorf("ListDeployments() error = %v", err)
		}

		found := false
		for _, d := range deployments {
			if d.DeploymentID == resp.DeploymentID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Created deployment not found in list")
		}
	})
}

func waitForDeployment(t *testing.T, deploymentID string) (*api.DeploymentStatus, error) {
	t.Helper()
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, ErrDeploymentTimeout
		case <-ticker.C:
			status, err := api.GetDeploymentStatus(deploymentID)
			if err != nil {
				return nil, err
			}

			switch status.Status {
			case api.StatusCompleted:
				return status, nil
			case api.StatusFailed:
				return status, ErrDeploymentFailed
			}
		}
	}
}
