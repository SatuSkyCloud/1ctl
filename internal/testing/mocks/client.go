package mocks

import (
	"1ctl/internal/api"
)

// APIClient is the interface that all API clients must implement
type APIClient interface {
	CreateDeployment(deployment api.Deployment, response *string) error
	CreateService(service api.Service, response *string) error
	CreateIngress(ingress api.Ingress) (*api.Ingress, error)
	CreateVolume(volume api.Volume) error
	CreateEnvironment(env api.Environment) (*api.Environment, error)
}

// Ensure MockAPI implements APIClient
var _ APIClient = (*MockAPI)(nil)

// Implement the interface methods for MockAPI
func (m *MockAPI) CreateDeployment(deployment api.Deployment, response *string) error {
	return m.CreateDeploymentFunc(deployment, response)
}

func (m *MockAPI) CreateService(service api.Service, response *string) error {
	return m.CreateServiceFunc(service, response)
}

func (m *MockAPI) CreateIngress(ingress api.Ingress) (*api.Ingress, error) {
	return m.CreateIngressFunc(ingress)
}

func (m *MockAPI) CreateVolume(volume api.Volume) error {
	return m.CreateVolumeFunc(volume)
}

func (m *MockAPI) CreateEnvironment(env api.Environment) (*api.Environment, error) {
	return m.CreateEnvironmentFunc(env)
}
