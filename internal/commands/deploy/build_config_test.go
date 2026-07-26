package deploy

import (
	"testing"

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
