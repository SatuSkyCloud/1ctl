package deploy

import (
	"testing"

	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/testutils"

	"github.com/google/uuid"
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

func TestSourceBuildTargetArchIsAuthoritative(t *testing.T) {
	tests := []struct {
		name      string
		imageArch string
		want      string
	}{
		{name: "detected architecture replaces config", imageArch: "linux/amd64", want: "amd64"},
		{name: "multi-arch detection clears config selector", imageArch: "linux/amd64,linux/arm64", want: ""},
		{name: "unknown detection clears config selector", imageArch: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DeploymentOptions{TargetArch: "arm64"}
			setSourceBuildTargetArch(&opts, tt.imageArch)
			if opts.TargetArch != tt.want {
				t.Errorf("TargetArch = %q, want %q", opts.TargetArch, tt.want)
			}
		})
	}
}

func TestBuildAtomicDeploymentIntentMapsRuntimeOptions(t *testing.T) {
	intended, err := buildAtomicDeploymentIntent(DeploymentOptions{
		CPURequest: "500m", CPULimit: "1", Memory: "512Mi", Organization: "tenant-a", Port: 8080,
		Replicas: 2, Zone: "my-kul-1b", TargetArch: "arm64", EnvEnabled: true,
		Environment:   &api.Environment{KeyValues: []api.KeyValuePair{{Key: "LOG_LEVEL", Value: "info"}}},
		IntentVolumes: []api.DeploymentIntentVolume{{VolumeName: "data", ClaimName: "data-pvc", StorageClass: "ceph-block", StorageSize: "1Gi", MountPath: "/data"}},
		DesiredStateConfig: api.DeploymentDesiredStateConfig{
			ReadinessProbe:  &api.DeploymentProbe{TCPSocket: &api.DeploymentTCPSocketProbe{Port: 8080}},
			RequiredSecrets: []api.DeploymentRequiredSecret{{Key: "DATABASE_URL"}},
		},
		HPAConfig: &api.HPAConfig{Enabled: true, MinReplicas: 2, MaxReplicas: 5},
		VPAConfig: &api.VPAConfig{Enabled: true, UpdateMode: "Initial"},
		PDBConfig: &PDBConfig{Enabled: true, Type: PDBTypeFixed},
		WaitFor:   []api.WaitFor{{Host: "postgres", Port: 5432}}, Strategy: "recreate",
	}, "registry.example/api:v1", "api", uuid.NewString())
	if err != nil {
		t.Fatalf("buildAtomicDeploymentIntent: %v", err)
	}
	if intended.Deployment.Image != "registry.example/api:v1" || intended.Deployment.Namespace != "tenant-a" || intended.Deployment.Replicas != 2 || intended.Deployment.TargetArch != "arm64" {
		t.Fatalf("deployment = %+v", intended.Deployment)
	}
	if intended.Service == nil || intended.Service.Name != "api" || intended.PublicRoute == nil || intended.PublicRoute.Kind != "default_dns" {
		t.Fatalf("intent route/service = %+v/%+v", intended.Service, intended.PublicRoute)
	}
	if len(intended.Environment) != 1 || len(intended.Volumes) != 1 || intended.Config.ReadinessProbe == nil || len(intended.Config.RequiredSecrets) != 1 {
		t.Fatalf("intent did not preserve declarations: %+v", intended)
	}
	if intended.Deployment.HPAConfig == nil || intended.Deployment.VPAConfig == nil || intended.Deployment.PDBConfig == nil || intended.Deployment.StrategyConfig == nil || len(intended.Deployment.WaitFor) != 1 {
		t.Fatalf("runtime options not mapped: %+v", intended.Deployment)
	}
}

func TestAtomicIntentFallbackIsExplicit(t *testing.T) {
	tests := []struct {
		opts DeploymentOptions
		want string
	}{
		{opts: DeploymentOptions{Domain: "api.example.com"}, want: "custom domain routing"},
		{opts: DeploymentOptions{Hostnames: []string{"machine-a"}}, want: "explicit machine placement"},
		{opts: DeploymentOptions{MulticlusterEnabled: true}, want: "multi-cluster deployment"},
		{opts: DeploymentOptions{Dependencies: []api.Dependency{{Name: "redis"}}}, want: "dependent workload creation"},
		{opts: DeploymentOptions{}, want: ""},
	}
	for _, tt := range tests {
		if got := atomicIntentFallbackReason(tt.opts); got != tt.want {
			t.Errorf("atomicIntentFallbackReason(%+v) = %q, want %q", tt.opts, got, tt.want)
		}
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

			resp, err := Deploy(tt.opts, uuid.NewString())
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
			_, _, err := submitRemoteBuild(tt.dockerfilePath, tt.projectName, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("submitRemoteBuild() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
