package mocks

import (
	"1ctl/internal/api"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MockAPI provides mock implementations for API calls
type MockAPI struct {
	LoginCLIFunc          func(token string) (*api.TokenValidate, error)
	ValidateTokenFunc     func(token string) (*api.TokenValidate, error)
	UpsertDeploymentFunc  func(deployment api.Deployment, response *string) error
	UpsertServiceFunc     func(service api.Service, response *string) error
	UpsertIngressFunc     func(ingress api.Ingress) (*api.Ingress, error)
	CreateVolumeFunc      func(volume api.Volume) error
	CreateEnvironmentFunc func(env api.Environment) (*api.Environment, error)
}

// DefaultMockAPI returns a MockAPI with default successful responses
func DefaultMockAPI() *MockAPI {
	return &MockAPI{
		LoginCLIFunc: func(token string) (*api.TokenValidate, error) {
			return &api.TokenValidate{
				Valid:     true,
				UserEmail: "test@example.com",
			}, nil
		},
		UpsertDeploymentFunc: func(deployment api.Deployment, response *string) error {
			*response = uuid.New().String()
			return nil
		},
		UpsertServiceFunc: func(service api.Service, response *string) error {
			*response = uuid.New().String()
			return nil
		},
		UpsertIngressFunc: func(ingress api.Ingress) (*api.Ingress, error) {
			ingress.IngressID = uuid.New()
			ingress.CreatedAt = time.Now()
			return &ingress, nil
		},
		CreateVolumeFunc: func(volume api.Volume) error {
			return nil
		},
		CreateEnvironmentFunc: func(env api.Environment) (*api.Environment, error) {
			env.EnvironmentID = uuid.New()
			env.CreatedAt = time.Now()
			return &env, nil
		},
	}
}

// ErrorMockAPI returns a MockAPI that returns errors
func ErrorMockAPI() *MockAPI {
	return &MockAPI{
		UpsertDeploymentFunc: func(deployment api.Deployment, response *string) error {
			return fmt.Errorf("mock deployment error")
		},
		UpsertServiceFunc: func(service api.Service, response *string) error {
			return fmt.Errorf("mock service error")
		},
		UpsertIngressFunc: func(ingress api.Ingress) (*api.Ingress, error) {
			return nil, fmt.Errorf("mock ingress error")
		},
		CreateVolumeFunc: func(volume api.Volume) error {
			return fmt.Errorf("mock volume error")
		},
		CreateEnvironmentFunc: func(env api.Environment) (*api.Environment, error) {
			return nil, fmt.Errorf("mock environment error")
		},
	}
}

// Add implementation for ValidateToken
func (m *MockAPI) ValidateToken(token string) (*api.TokenValidate, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(token)
	}
	return nil, fmt.Errorf("ValidateTokenFunc not implemented")
}
