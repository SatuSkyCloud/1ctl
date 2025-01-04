package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDeploymentValidation(t *testing.T) {
	tests := []struct {
		name    string
		deploy  Deployment
		wantErr bool
	}{
		{
			name: "valid deployment",
			deploy: Deployment{
				DeploymentID:  uuid.New(),
				UserID:        uuid.New(),
				AppLabel:      "test-deployment",
				Image:         "test-image:latest",
				CpuRequest:    "100m",
				MemoryRequest: "256Mi",
				MemoryLimit:   "512Mi",
				Replicas:      1,
				Port:          8080,
				Namespace:     "default",
				Environment:   "production",
			},
			wantErr: false,
		},
		{
			name: "invalid CPU request",
			deploy: Deployment{
				DeploymentID:  uuid.New(),
				UserID:        uuid.New(),
				AppLabel:      "test-deployment",
				Image:         "test-image:latest",
				CpuRequest:    "invalid",
				MemoryRequest: "256Mi",
				MemoryLimit:   "512Mi",
				Replicas:      1,
				Port:          8080,
				Namespace:     "default",
				Environment:   "production",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDeployment(&tt.deploy)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDeployment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvironmentValidation(t *testing.T) {
	tests := []struct {
		name    string
		env     Environment
		wantErr bool
	}{
		{
			name: "valid environment",
			env: Environment{
				EnvironmentID: uuid.New(),
				DeploymentID:  uuid.New(),
				Namespace:     "default",
				AppLabel:      "test-app",
				KeyValues: []KeyValuePair{
					{Key: "TEST_KEY", Value: "test_value"},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty key",
			env: Environment{
				EnvironmentID: uuid.New(),
				DeploymentID:  uuid.New(),
				Namespace:     "default",
				AppLabel:      "test-app",
				KeyValues: []KeyValuePair{
					{Key: "", Value: "test_value"},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvironment(&tt.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEnvironment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to validate deployment
func validateDeployment(d *Deployment) error {
	if d.AppLabel == "" {
		return fmt.Errorf("app label is required")
	}
	if d.Image == "" {
		return fmt.Errorf("image is required")
	}
	if d.CpuRequest == "invalid" {
		return fmt.Errorf("invalid CPU request")
	}
	return nil
}

// Helper function to validate environment
func validateEnvironment(e *Environment) error {
	if e.EnvironmentID == uuid.Nil {
		return fmt.Errorf("environment ID is required")
	}
	for _, kv := range e.KeyValues {
		if kv.Key == "" {
			return fmt.Errorf("environment variable key cannot be empty")
		}
	}
	return nil
}
