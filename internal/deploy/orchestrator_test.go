package deploy

import (
	"testing"

	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/testutils"
)

func TestBuildStrategyConfig(t *testing.T) {
	tests := []struct {
		name            string
		opts            DeploymentOptions
		wantNil         bool
		wantType        api.DeploymentStrategyType
		wantSurge       string
		wantUnavailable string
	}{
		{
			name:    "untouched defaults omit config",
			opts:    DeploymentOptions{Strategy: "rolling", RollingMaxSurge: "25%", RollingMaxUnavailable: "25%"},
			wantNil: true,
		},
		{
			name:            "explicit defaults are preserved (#27 sub-5)",
			opts:            DeploymentOptions{Strategy: "rolling", RollingMaxSurge: "25%", RollingMaxUnavailable: "25%", RollingFlagsExplicit: true},
			wantNil:         false,
			wantType:        api.StrategyRolling,
			wantSurge:       "25%",
			wantUnavailable: "25%",
		},
		{
			name:            "non-default values send config",
			opts:            DeploymentOptions{Strategy: "rolling", RollingMaxSurge: "50%", RollingMaxUnavailable: "0"},
			wantNil:         false,
			wantType:        api.StrategyRolling,
			wantSurge:       "50%",
			wantUnavailable: "0",
		},
		{
			name:     "recreate strategy always sends config",
			opts:     DeploymentOptions{Strategy: "recreate"},
			wantNil:  false,
			wantType: api.StrategyRecreate,
		},
		{
			name:    "empty strategy is treated as rolling+default",
			opts:    DeploymentOptions{RollingMaxSurge: "25%", RollingMaxUnavailable: "25%"},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildStrategyConfig(tt.opts)
			if tt.wantNil {
				if got != nil {
					t.Errorf("buildStrategyConfig = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("buildStrategyConfig = nil, want non-nil")
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if tt.wantType == api.StrategyRolling {
				if got.Rolling == nil {
					t.Fatalf("Rolling is nil for rolling strategy")
				}
				if got.Rolling.MaxSurge != tt.wantSurge {
					t.Errorf("MaxSurge = %q, want %q", got.Rolling.MaxSurge, tt.wantSurge)
				}
				if got.Rolling.MaxUnavailable != tt.wantUnavailable {
					t.Errorf("MaxUnavailable = %q, want %q", got.Rolling.MaxUnavailable, tt.wantUnavailable)
				}
			}
		})
	}
}

func TestNormalizeTargetArch(t *testing.T) {
	tests := []struct {
		name      string
		imageArch string
		want      string
	}{
		{name: "empty", imageArch: "", want: ""},
		{name: "single amd64", imageArch: "amd64", want: "amd64"},
		{name: "single arm64", imageArch: "arm64", want: "arm64"},
		{name: "linux prefix amd64", imageArch: "linux/amd64", want: "amd64"},
		{name: "linux prefix arm64", imageArch: "linux/arm64", want: "arm64"},
		{name: "multi-arch list", imageArch: "linux/amd64,linux/arm64", want: ""},
		{name: "unknown value", imageArch: "linux/ppc64le", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTargetArch(tt.imageArch); got != tt.want {
				t.Errorf("normalizeTargetArch(%q) = %q, want %q", tt.imageArch, got, tt.want)
			}
		})
	}
}

func TestDeploy(t *testing.T) {
	// Skip this test in CI - it requires Docker daemon and actual API
	// This is an integration test that should run with proper setup
	t.Skip("Skipping integration test - requires Docker daemon and API")

	// Setup test context
	if err := context.SetToken("test-token"); err != nil {
		t.Fatalf("failed to set token: %v", err)
	}
	if err := context.SetUserID("test-user"); err != nil {
		t.Fatalf("failed to set user ID: %v", err)
	}
	if err := context.SetCurrentNamespace("test-org"); err != nil {
		t.Fatalf("failed to set namespace: %v", err)
	}

	tests := []struct {
		name    string
		opts    DeploymentOptions
		mockAPI *testutils.MockAPI
		wantErr bool
	}{
		{
			name: "successful deployment",
			opts: DeploymentOptions{
				CPU:            "1",
				Memory:         "512Mi",
				Organization:   "test-project",
				Port:           8080,
				DockerfilePath: "Dockerfile",
			},
			mockAPI: testutils.DefaultMockAPI(),
			wantErr: false,
		},
		{
			name: "deployment with environment",
			opts: DeploymentOptions{
				CPU:            "1",
				Memory:         "512Mi",
				Organization:   "test-project",
				Port:           8080,
				DockerfilePath: "Dockerfile",
				EnvEnabled:     true,
				Environment: &api.Environment{
					KeyValues: []api.KeyValuePair{
						{Key: "TEST_KEY", Value: "test_value"},
					},
				},
			},
			mockAPI: testutils.DefaultMockAPI(),
			wantErr: false,
		},
		{
			name: "deployment with volume",
			opts: DeploymentOptions{
				CPU:            "1",
				Memory:         "512Mi",
				Organization:   "test-project",
				Port:           8080,
				DockerfilePath: "Dockerfile",
				VolumeEnabled:  true,
				Volume: &api.Volume{
					StorageSize: "10Gi",
					MountPath:   "/data",
				},
			},
			mockAPI: testutils.DefaultMockAPI(),
			wantErr: false,
		},
		{
			name: "deployment error",
			opts: DeploymentOptions{
				CPU:            "1",
				Memory:         "512Mi",
				Organization:   "test-project",
				Port:           8080,
				DockerfilePath: "Dockerfile",
			},
			mockAPI: testutils.ErrorMockAPI(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: Replace actual API calls with mock
			// This requires refactoring the deploy package to accept an API interface

			resp, err := Deploy(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Deploy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && resp == nil {
				t.Error("Deploy() returned nil response for successful deployment")
			}
		})
	}
}

func TestSubmitRemoteBuild(t *testing.T) {
	tests := []struct {
		name           string
		dockerfilePath string
		projectName    string
		wantErr        bool
	}{
		{
			name:           "missing dockerfile is rejected before upload",
			dockerfilePath: "testdata/nonexistent",
			projectName:    "test-project",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := submitRemoteBuild(tt.dockerfilePath, tt.projectName)
			if (err != nil) != tt.wantErr {
				t.Errorf("submitRemoteBuild() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
