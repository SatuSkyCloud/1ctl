package mocks

import (
	"1ctl/internal/api"
)

// APIClient is the interface that all API clients must implement
type APIClient interface {
	UpsertDeployment(deployment api.Deployment, response *string) error
	UpsertService(service api.Service, response *string) error
	UpsertIngress(ingress api.Ingress) (*api.Ingress, error)
	CreateVolume(volume api.Volume) error
	CreateEnvironment(env api.Environment) (*api.Environment, error)
}

// Ensure MockAPI implements APIClient
var _ APIClient = (*MockAPI)(nil)

// Implement the interface methods for MockAPI
func (m *MockAPI) UpsertDeployment(deployment api.Deployment, response *string) error {
	return m.UpsertDeploymentFunc(deployment, response)
}

func (m *MockAPI) UpsertService(service api.Service, response *string) error {
	return m.UpsertServiceFunc(service, response)
}

func (m *MockAPI) UpsertIngress(ingress api.Ingress) (*api.Ingress, error) {
	return m.UpsertIngressFunc(ingress)
}

func (m *MockAPI) CreateVolume(volume api.Volume) error {
	return m.CreateVolumeFunc(volume)
}

func (m *MockAPI) CreateEnvironment(env api.Environment) (*api.Environment, error) {
	return m.CreateEnvironmentFunc(env)
}
