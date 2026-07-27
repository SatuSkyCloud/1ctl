package deploy

import (
	"testing"

	"1ctl/internal/api"
	"1ctl/internal/config"
)

func TestMergeConfigExplicitImageTakesPrecedence(t *testing.T) {
	cfg := &config.ProjectConfig{
		Build: config.BuildConfig{
			Image:      "registry.example/config:v1",
			TargetArch: "arm64",
		},
	}
	got := mergeConfig(DeployInput{
		Image:        "registry.example/cli:v2",
		Dockerfile:   "Dockerfile",
		Organization: "test-org",
	}, cfg)

	if got.Image != "registry.example/cli:v2" {
		t.Errorf("Image = %q, want explicit CLI image", got.Image)
	}
	if got.TargetArch != "arm64" {
		t.Errorf("TargetArch = %q, want config architecture to apply to explicit image", got.TargetArch)
	}
}

func TestPrepareDeploymentOptionsMapsCanonicalDeclarations(t *testing.T) {
	one := int32(1)
	cfg := &config.ProjectConfig{
		Checks: config.ChecksConfig{Readiness: &config.ProbeConfig{
			TCPSocket:        &config.TCPSocketProbeConfig{Port: 8080},
			PeriodSeconds:    &one,
			FailureThreshold: &one,
		}},
		Secrets: config.SecretsConfig{Required: []string{"DATABASE_URL"}},
		Volumes: []config.VolumeConfig{{Name: "data", Claim: "api-data-pvc", Size: "1Gi", Mount: "/data", StorageClass: "ceph-block"}},
	}
	merged := mergeConfig(DeployInput{Image: "registry.example/api:v1", Organization: "test-org", Strategy: "rolling"}, cfg)
	opts, err := prepareDeploymentOptions(merged, cfg)
	if err != nil {
		t.Fatalf("prepareDeploymentOptions: %v", err)
	}
	if opts.DesiredStateConfig.ReadinessProbe == nil || opts.DesiredStateConfig.ReadinessProbe.TCPSocket == nil || opts.DesiredStateConfig.ReadinessProbe.TCPSocket.Port != 8080 {
		t.Fatalf("readiness probe = %+v", opts.DesiredStateConfig.ReadinessProbe)
	}
	if got := opts.DesiredStateConfig.RequiredSecrets; len(got) != 1 || got[0] != (api.DeploymentRequiredSecret{Key: "DATABASE_URL"}) {
		t.Fatalf("required secrets = %+v", got)
	}
	if got := opts.IntentVolumes; len(got) != 1 || got[0].ClaimName != "api-data-pvc" || got[0].MountPath != "/data" {
		t.Fatalf("intent volumes = %+v", got)
	}
	if !opts.AtomicOnlyConfig {
		t.Fatal("AtomicOnlyConfig = false, want true for canonical declarations")
	}
}

func TestPrepareDeploymentOptionsUsesConfigPrebuiltImage(t *testing.T) {
	cfg := &config.ProjectConfig{
		Build: config.BuildConfig{
			Image:      "registry.example/config:v1",
			TargetArch: "amd64",
		},
	}
	merged := mergeConfig(DeployInput{
		Dockerfile:   "Dockerfile",
		Organization: "test-org",
		Strategy:     "rolling",
	}, cfg)

	opts, err := prepareDeploymentOptions(merged, cfg)
	if err != nil {
		t.Fatalf("prepareDeploymentOptions: %v", err)
	}
	if opts.PrebuiltImage != "registry.example/config:v1" {
		t.Errorf("PrebuiltImage = %q, want config image", opts.PrebuiltImage)
	}
	if opts.DockerfilePath != "" {
		t.Errorf("DockerfilePath = %q, want empty for pre-built image", opts.DockerfilePath)
	}
	if opts.TargetArch != "amd64" {
		t.Errorf("TargetArch = %q, want amd64", opts.TargetArch)
	}
}
