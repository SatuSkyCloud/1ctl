package integration

import (
	"1ctl/internal/api"
	"testing"
)

func TestSecretFlow(t *testing.T) {
	setupTestAuth(t)

	// First create a deployment to attach secrets to
	deploymentID := createTestDeployment(t)
	defer cleanupDeployment(t, deploymentID.String())

	t.Run("create and manage secret", func(t *testing.T) {
		secret := api.Secret{
			DeploymentID: deploymentID,
			AppLabel:     "test-app",
			KeyValues: []api.KeyValuePair{
				{Key: "API_KEY", Value: "secret-key-123"},
				{Key: "API_SECRET", Value: "secret-value-456"},
			},
		}

		// Create secret
		secretResp, err := api.CreateSecret(secret)
		if err != nil {
			t.Fatalf("CreateSecret() error = %v", err)
		}

		// Test secret listing
		secrets, err := api.ListSecrets()
		if err != nil {
			t.Errorf("ListSecrets() error = %v", err)
		}

		found := false
		for _, s := range secrets {
			if s.SecretID == secretResp.SecretID {
				found = true
				// Verify secret has correct number of keys
				if len(s.KeyValues) != 2 {
					t.Errorf("Expected 2 secret keys, got %d", len(s.KeyValues))
				}
				break
			}
		}
		if !found {
			t.Error("Created secret not found in list")
		}

		// Test secret deletion
		err = api.DeleteSecret(secretResp.SecretID.String())
		if err != nil {
			t.Errorf("DeleteSecret() error = %v", err)
		}
	})
}
