package integration

import (
	"1ctl/internal/api"
	"testing"
)

func TestEnvironmentFlow(t *testing.T) {
	setupTestAuth(t)

	// First create a deployment to attach environment to
	deploymentID := createTestDeployment(t)
	defer cleanupDeployment(t, deploymentID.String())

	t.Run("create and manage environment", func(t *testing.T) {
		env := api.Environment{
			DeploymentID: deploymentID,
			AppLabel:     "test-env",
			KeyValues: []api.KeyValuePair{
				{Key: "DB_HOST", Value: "localhost"},
				{Key: "DB_PORT", Value: "5432"},
			},
		}

		// Create environment
		envResp, err := api.CreateEnvironment(env)
		if err != nil {
			t.Fatalf("CreateEnvironment() error = %v", err)
		}

		// Test environment listing
		environments, err := api.ListEnvironments()
		if err != nil {
			t.Errorf("ListEnvironments() error = %v", err)
		}

		found := false
		for _, e := range environments {
			if e.EnvironmentID == envResp.EnvironmentID {
				found = true
				// Verify environment variables
				if len(e.KeyValues) != 2 {
					t.Errorf("Expected 2 environment variables, got %d", len(e.KeyValues))
				}
				break
			}
		}
		if !found {
			t.Error("Created environment not found in list")
		}

		// Test environment deletion
		err = api.DeleteEnvironment(nil, envResp.EnvironmentID.String())
		if err != nil {
			t.Errorf("DeleteEnvironment() error = %v", err)
		}
	})
}
