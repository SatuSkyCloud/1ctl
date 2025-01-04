package integration

import (
	"1ctl/internal/api"
	"testing"
	"time"
)

func TestServiceFlow(t *testing.T) {
	setupTestAuth(t)

	// First create a deployment to attach services to
	deploymentID := createTestDeployment(t)
	defer cleanupDeployment(t, deploymentID.String())

	t.Run("create and manage service", func(t *testing.T) {
		service := api.Service{
			DeploymentID: deploymentID,
			ServiceName:  "test-service",
			Port:         8080,
		}

		var serviceID string
		err := api.CreateService(service, &serviceID)
		if err != nil {
			t.Fatalf("CreateService() error = %v", err)
		}

		// Wait for service to be ready
		err = waitForService(t, serviceID)
		if err != nil {
			t.Fatalf("Service failed to become ready: %v", err)
		}

		// Test service listing
		services, err := api.ListServices()
		if err != nil {
			t.Errorf("ListServices() error = %v", err)
		}

		found := false
		for _, s := range services {
			if s.ServiceID.String() == serviceID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Created service not found in list")
		}

		// Test service deletion
		err = api.DeleteService(nil, serviceID)
		if err != nil {
			t.Errorf("DeleteService() error = %v", err)
		}
	})
}

func waitForService(t *testing.T, serviceID string) error {
	t.Helper()
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return ErrServiceTimeout
		case <-ticker.C:
			services, err := api.ListServices()
			if err != nil {
				return err
			}

			for _, s := range services {
				if s.ServiceID.String() == serviceID {
					return nil
				}
			}
		}
	}
}
