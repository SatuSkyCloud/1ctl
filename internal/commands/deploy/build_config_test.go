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

func TestMergeConfigUsesDeployStrategyInsteadOfCommandDefaults(t *testing.T) {
	cfg := &config.ProjectConfig{
		Deploy: config.DeployConfig{
			Strategy:              "rolling",
			RollingMaxSurge:       "1",
			RollingMaxUnavailable: "0",
		},
	}

	merged := mergeConfig(DeployInput{
		Strategy:          "rolling",
		RollingMaxSurge:   "25%",
		RollingMaxUnavail: "25%",
	}, cfg)

	if merged.Strategy != "rolling" {
		t.Errorf("Strategy = %q, want rolling", merged.Strategy)
	}
	if merged.RollingMaxSurge != "1" {
		t.Errorf("RollingMaxSurge = %q, want 1", merged.RollingMaxSurge)
	}
	if merged.RollingMaxUnavail != "0" {
		t.Errorf("RollingMaxUnavail = %q, want 0", merged.RollingMaxUnavail)
	}
}

func TestMergeConfigExplicitDefaultValuedFlagsTakePrecedence(t *testing.T) {
	cfg := &config.ProjectConfig{
		App: config.AppConfig{
			CPURequest: "500m",
			CPULimit:   "2",
			Memory:     "1Gi",
			Port:       3000,
		},
		Deploy: config.DeployConfig{
			Strategy:              "recreate",
			RollingMaxSurge:       "1",
			RollingMaxUnavailable: "0",
		},
	}
	in := DeployInput{
		CPURequest:        "250m",
		CPULimit:          "1",
		Memory:            "256Mi",
		Port:              8080,
		Strategy:          "rolling",
		RollingMaxSurge:   "25%",
		RollingMaxUnavail: "25%",
		SetFlags: map[string]bool{
			flagCPURequest: true, flagCPULimit: true, flagMemory: true,
			flagPort: true, flagStrategy: true, flagRollingMaxSurge: true,
			flagRollingMaxUnavail: true,
		},
	}

	got := mergeConfig(in, cfg)
	if got.CPURequest != "250m" || got.CPULimit != "1" || got.Memory != "256Mi" || got.Port != 8080 {
		t.Fatalf("resource flags were overwritten by config: %+v", got.DeployInput)
	}
	if got.Strategy != "rolling" || got.RollingMaxSurge != "25%" || got.RollingMaxUnavail != "25%" {
		t.Fatalf("rollout flags were overwritten by config: %+v", got.DeployInput)
	}
}

func TestMergeConfigUsesConfiguredDockerfileWhenFlagIsUnset(t *testing.T) {
	cfg := &config.ProjectConfig{Build: config.BuildConfig{Dockerfile: "Dockerfile.prod"}}

	got := mergeConfig(DeployInput{
		Dockerfile: "Dockerfile",
		SetFlags:   map[string]bool{},
	}, cfg)

	if got.Dockerfile != "Dockerfile.prod" {
		t.Fatalf("Dockerfile = %q, want Dockerfile.prod", got.Dockerfile)
	}
}

func TestMergeConfigExplicitDefaultDockerfileTakesPrecedence(t *testing.T) {
	cfg := &config.ProjectConfig{Build: config.BuildConfig{Dockerfile: "Dockerfile.prod"}}

	got := mergeConfig(DeployInput{
		Dockerfile: "Dockerfile",
		SetFlags:   map[string]bool{flagDockerfile: true},
	}, cfg)

	if got.Dockerfile != "Dockerfile" {
		t.Fatalf("Dockerfile = %q, want explicit Dockerfile", got.Dockerfile)
	}
}

func TestMergeConfigUsesMulticlusterBackupConfiguration(t *testing.T) {
	cfg := &config.ProjectConfig{Multicluster: config.MulticlusterConfig{
		Enabled:               true,
		Mode:                  "active-active",
		BackupEnabled:         false,
		BackupSchedule:        "hourly",
		BackupRetention:       "72h",
		BackupPriorityCluster: 2,
	}}

	got := mergeConfig(DeployInput{
		MulticlusterMode: "active-passive",
		BackupEnabled:    true,
		BackupSchedule:   "daily",
		BackupRetention:  "168h",
		BackupPriority:   1,
		SetFlags:         map[string]bool{},
	}, cfg)

	if !got.Multicluster || got.MulticlusterMode != "active-active" {
		t.Fatalf("multicluster = %v mode = %q", got.Multicluster, got.MulticlusterMode)
	}
	if got.BackupEnabled || got.BackupSchedule != "hourly" || got.BackupRetention != "72h" || got.BackupPriority != 2 {
		t.Fatalf("backup config not applied: enabled=%v schedule=%q retention=%q priority=%d",
			got.BackupEnabled, got.BackupSchedule, got.BackupRetention, got.BackupPriority)
	}
}

func TestMergeConfigExplicitBackupDefaultsTakePrecedence(t *testing.T) {
	cfg := &config.ProjectConfig{Multicluster: config.MulticlusterConfig{
		Enabled: true, BackupEnabled: false, BackupSchedule: "hourly",
		BackupRetention: "72h", BackupPriorityCluster: 2,
	}}

	got := mergeConfig(DeployInput{
		BackupEnabled: true, BackupSchedule: "daily", BackupRetention: "168h", BackupPriority: 1,
		SetFlags: map[string]bool{
			flagBackupEnabled: true, flagBackupSchedule: true,
			flagBackupRetention: true, flagBackupPriority: true,
		},
	}, cfg)

	if !got.BackupEnabled || got.BackupSchedule != "daily" || got.BackupRetention != "168h" || got.BackupPriority != 1 {
		t.Fatalf("explicit backup flags were overwritten: enabled=%v schedule=%q retention=%q priority=%d",
			got.BackupEnabled, got.BackupSchedule, got.BackupRetention, got.BackupPriority)
	}
}

func TestMergeConfigSuppliesAutoscalingAndDisruptionValues(t *testing.T) {
	cfg := &config.ProjectConfig{
		HPA: config.HPAConfig{Enabled: true, MinReplicas: 3, MaxReplicas: 20, CPUTarget: 60, MemoryTarget: 70},
		VPA: config.VPAConfig{Enabled: true, Mode: "Initial", MinCPU: "100m", MaxCPU: "2", MinMemory: "128Mi", MaxMemory: "2Gi"},
		PDB: config.PDBConfig{Enabled: true, Type: "fixed", MinAvailable: 2},
	}
	merged := mergeConfig(DeployInput{
		Image: "registry.example/api:v1", HPAMinReplicas: 1, HPAMaxReplicas: 10,
		HPACPUCoreTarget: 80, VPAMode: "Off", PDBType: "auto", Strategy: "rolling", SetFlags: map[string]bool{},
	}, cfg)

	opts, err := prepareDeploymentOptions(merged, cfg)
	if err != nil {
		t.Fatalf("prepareDeploymentOptions: %v", err)
	}
	if opts.HPAConfig == nil || opts.HPAConfig.MinReplicas != 3 || opts.HPAConfig.MaxReplicas != 20 ||
		opts.HPAConfig.CPUTarget == nil || *opts.HPAConfig.CPUTarget != 60 ||
		opts.HPAConfig.MemoryTarget == nil || *opts.HPAConfig.MemoryTarget != 70 {
		t.Fatalf("HPA config = %+v", opts.HPAConfig)
	}
	if opts.VPAConfig == nil || opts.VPAConfig.UpdateMode != "Initial" || opts.VPAConfig.MinCPU != "100m" || opts.VPAConfig.MaxMemory != "2Gi" {
		t.Fatalf("VPA config = %+v", opts.VPAConfig)
	}
	if opts.PDBConfig == nil || opts.PDBConfig.Type != "fixed" || opts.PDBConfig.MinAvailable == nil || *opts.PDBConfig.MinAvailable != 2 {
		t.Fatalf("PDB config = %+v", opts.PDBConfig)
	}
}
