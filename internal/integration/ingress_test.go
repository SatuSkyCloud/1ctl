package integration

import (
	"1ctl/internal/api"
	"errors"
	"testing"
	"time"
)

func TestIngressFlow(t *testing.T) {
	setupTestAuth(t)

	// First create a deployment and service
	deploymentID := createTestDeployment(t)
	defer cleanupDeployment(t, deploymentID.String())

	// Create service for ingress
	serviceID := createTestService(t, deploymentID)
	defer cleanupService(t, serviceID)

	t.Run("create and manage ingress", func(t *testing.T) {
		ingress := api.Ingress{
			DeploymentID: deploymentID,
			ServiceID:    api.ToUUID(serviceID),
			AppLabel:     "test-app",
			Namespace:    "test-namespace",
			DnsConfig:    "default",
			DomainName:   "test.satusky.com",
			Port:         8080,
		}

		// Create ingress
		ingressResp, err := api.CreateIngress(ingress)
		if err != nil {
			t.Fatalf("CreateIngress() error = %v", err)
		}

		// Wait for ingress to be ready
		err = waitForIngress(t, ingressResp.IngressID.String())
		if err != nil {
			t.Fatalf("Ingress failed to become ready: %v", err)
		}

		// Test ingress listing
		ingresses, err := api.ListIngresses()
		if err != nil {
			t.Errorf("ListIngresses() error = %v", err)
		}

		found := false
		for _, i := range ingresses {
			if i.IngressID == ingressResp.IngressID {
				found = true
				if i.DomainName != ingress.DomainName {
					t.Errorf("Domain = %v, want %v", i.DomainName, ingress.DomainName)
				}
				break
			}
		}
		if !found {
			t.Error("Created ingress not found in list")
		}

		// Test ingress deletion
		err = api.DeleteIngress(nil, ingressResp.IngressID.String())
		if err != nil {
			t.Errorf("DeleteIngress() error = %v", err)
		}
	})
}

func waitForIngress(t *testing.T, ingressID string) error {
	t.Helper()
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return errors.New("timeout waiting for ingress to be ready")
		case <-ticker.C:
			ingresses, err := api.ListIngresses()
			if err != nil {
				return err
			}

			for _, i := range ingresses {
				if i.IngressID.String() == ingressID {
					return nil
				}
			}
		}
	}
}
