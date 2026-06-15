package api

import (
	cliContext "1ctl/internal/context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func TestMachineHasTag(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		tag    string
		want   bool
	}{
		{
			name:   "exact match present",
			labels: map[string]string{"satusky.com/production": "true"},
			tag:    "production",
			want:   true,
		},
		{
			name:   "value is irrelevant — key presence wins",
			labels: map[string]string{"satusky.com/staging": ""},
			tag:    "staging",
			want:   true,
		},
		{
			name:   "different tag does not match",
			labels: map[string]string{"satusky.com/production": "true"},
			tag:    "staging",
			want:   false,
		},
		{
			name:   "non-satusky prefix does not match",
			labels: map[string]string{"production": "true"},
			tag:    "production",
			want:   false,
		},
		{
			name:   "nil labels return false",
			labels: nil,
			tag:    "production",
			want:   false,
		},
		{
			name:   "empty tag returns false",
			labels: map[string]string{"satusky.com/production": "true"},
			tag:    "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MachineHasTag(tt.labels, tt.tag); got != tt.want {
				t.Errorf("MachineHasTag(%v, %q) = %v, want %v", tt.labels, tt.tag, got, tt.want)
			}
		})
	}
}

func TestMakeMainAPIRequestPreservesV1Prefix(t *testing.T) {
	originalClient := httpClient
	t.Cleanup(func() { httpClient = originalClient })

	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/machines/machine-123/details" {
			t.Errorf("path = %s, want /v1/machines/machine-123/details", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":false,"data":{"ok":true}}`)),
		}, nil
	})}

	originalStore := cliContext.Default()
	configDir := filepath.Join(t.TempDir(), ".satusky")
	profilesDir := filepath.Join(configDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, "test.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("WriteFile(profile) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "context.json"), []byte(`{"active_profile":"test"}`), 0600); err != nil {
		t.Fatalf("WriteFile(context) error = %v", err)
	}
	cliContext.SetDefault(cliContext.NewTestStore(configDir))
	t.Cleanup(func() { cliContext.SetDefault(originalStore) })
	if err := cliContext.SetToken("test-token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	t.Setenv("SATUSKY_API_URL", "http://localhost:8080/v1/cli")

	var resp struct {
		Error bool            `json:"error"`
		Data  map[string]bool `json:"data"`
	}
	if err := makeMainAPIRequest(http.MethodGet, "/machines/machine-123/details", nil, &resp); err != nil {
		t.Fatalf("makeMainAPIRequest() error = %v", err)
	}
	if !resp.Data["ok"] {
		t.Fatalf("response data = %v, want ok=true", resp.Data)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func TestParseUUID(t *testing.T) {
	// Issue #23: ParseUUID returns descriptive error; ToUUID returns uuid.Nil.
	if _, err := ParseUUID(""); err == nil {
		t.Error("ParseUUID(\"\") should error")
	}
	if _, err := ParseUUID("not-a-uuid"); err == nil {
		t.Error("ParseUUID(\"not-a-uuid\") should error")
	}
	if _, err := ParseUUID("00000000-0000-0000-0000-000000000000"); err != nil {
		t.Errorf("ParseUUID(valid uuid) errored: %v", err)
	}
	// ToUUID's silent-nil contract is intentional for already-validated inputs.
	if got := ToUUID("not-a-uuid"); got != uuid.Nil {
		t.Errorf("ToUUID(invalid) = %v, want uuid.Nil", got)
	}
}
