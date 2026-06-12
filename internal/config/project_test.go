package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeToml writes a satusky.toml with the given contents to a fresh temp dir
// and returns the path to the file (and the dir, so the caller can chdir).
func writeToml(t *testing.T, contents string) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = filepath.Join(dir, DefaultConfigFile)
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatalf("write toml: %v", err)
	}
	return dir, path
}

func TestParseV2Schema_ScalarFields(t *testing.T) {
	contents := `
[app]
name = "myapp"
port = 3000
dockerfile = "Dockerfile.prod"
cpu = "1"
memory = "512Mi"
replicas = 3
domain = "myapp.example.com"
health_path = "/healthz"
zone = "my-kul-1b"
organization = "my-org-slug"
strategy = "recreate"
rolling_max_surge = "50%"
rolling_max_unavailable = "0"
machine_tag = "production"
wait_for = ["postgres:5432", "redis:6379"]
`
	_, path := writeToml(t, contents)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.App.Name != "myapp" {
		t.Errorf("Name = %q, want myapp", cfg.App.Name)
	}
	if cfg.App.Port != 3000 {
		t.Errorf("Port = %d, want 3000", cfg.App.Port)
	}
	if cfg.App.Dockerfile != "Dockerfile.prod" {
		t.Errorf("Dockerfile = %q, want Dockerfile.prod (issue #18: was silently ignored before v2)", cfg.App.Dockerfile)
	}
	if cfg.App.CPU != "1" {
		t.Errorf("CPU = %q, want 1", cfg.App.CPU)
	}
	if cfg.App.Memory != "512Mi" {
		t.Errorf("Memory = %q, want 512Mi", cfg.App.Memory)
	}
	if cfg.App.Domain != "myapp.example.com" {
		t.Errorf("Domain = %q, want myapp.example.com", cfg.App.Domain)
	}
	if cfg.App.HealthPath != "/healthz" {
		t.Errorf("HealthPath = %q, want /healthz", cfg.App.HealthPath)
	}
	if cfg.App.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3 (issue #18: was silently ignored before v2)", cfg.App.Replicas)
	}
	if cfg.App.Zone != "my-kul-1b" {
		t.Errorf("Zone = %q, want my-kul-1b", cfg.App.Zone)
	}
	if cfg.App.Organization != "my-org-slug" {
		t.Errorf("Organization = %q, want my-org-slug", cfg.App.Organization)
	}
	if cfg.App.Strategy != "recreate" {
		t.Errorf("Strategy = %q, want recreate", cfg.App.Strategy)
	}
	if cfg.App.RollingMaxSurge != "50%" {
		t.Errorf("RollingMaxSurge = %q, want 50%%", cfg.App.RollingMaxSurge)
	}
	if cfg.App.RollingMaxUnavailable != "0" {
		t.Errorf("RollingMaxUnavailable = %q, want 0", cfg.App.RollingMaxUnavailable)
	}
	if cfg.App.MachineTag != "production" {
		t.Errorf("MachineTag = %q, want production", cfg.App.MachineTag)
	}
	if len(cfg.App.WaitFor) != 2 || cfg.App.WaitFor[0] != "postgres:5432" || cfg.App.WaitFor[1] != "redis:6379" {
		t.Errorf("WaitFor = %v, want [postgres:5432 redis:6379]", cfg.App.WaitFor)
	}
}

func TestParseV2Schema_VolumeSection(t *testing.T) {
	contents := `
[app]
name = "myapp"

[volume]
size = "10Gi"
mount = "/data"
`
	_, path := writeToml(t, contents)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Volume.Size != "10Gi" {
		t.Errorf("Volume.Size = %q, want 10Gi", cfg.Volume.Size)
	}
	if cfg.Volume.Mount != "/data" {
		t.Errorf("Volume.Mount = %q, want /data", cfg.Volume.Mount)
	}
}

func TestParseV2Schema_HPASection(t *testing.T) {
	contents := `
[app]
name = "myapp"

[hpa]
enabled = true
min_replicas = 2
max_replicas = 8
cpu_target = 75
memory_target = 80
`
	_, path := writeToml(t, contents)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.HPA.Enabled {
		t.Error("HPA.Enabled = false, want true")
	}
	if cfg.HPA.MinReplicas != 2 {
		t.Errorf("HPA.MinReplicas = %d, want 2", cfg.HPA.MinReplicas)
	}
	if cfg.HPA.MaxReplicas != 8 {
		t.Errorf("HPA.MaxReplicas = %d, want 8", cfg.HPA.MaxReplicas)
	}
	if cfg.HPA.CPUTarget != 75 {
		t.Errorf("HPA.CPUTarget = %d, want 75", cfg.HPA.CPUTarget)
	}
	if cfg.HPA.MemoryTarget != 80 {
		t.Errorf("HPA.MemoryTarget = %d, want 80", cfg.HPA.MemoryTarget)
	}
}

func TestParseV2Schema_VPASection(t *testing.T) {
	contents := `
[app]
name = "myapp"

[vpa]
enabled = true
mode = "Initial"
min_cpu = "100m"
max_cpu = "2"
min_memory = "128Mi"
max_memory = "4Gi"
`
	_, path := writeToml(t, contents)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.VPA.Enabled {
		t.Error("VPA.Enabled = false, want true")
	}
	if cfg.VPA.Mode != "Initial" {
		t.Errorf("VPA.Mode = %q, want Initial", cfg.VPA.Mode)
	}
	if cfg.VPA.MinCPU != "100m" || cfg.VPA.MaxCPU != "2" {
		t.Errorf("VPA CPU bounds = %q..%q, want 100m..2", cfg.VPA.MinCPU, cfg.VPA.MaxCPU)
	}
	if cfg.VPA.MinMemory != "128Mi" || cfg.VPA.MaxMemory != "4Gi" {
		t.Errorf("VPA mem bounds = %q..%q, want 128Mi..4Gi", cfg.VPA.MinMemory, cfg.VPA.MaxMemory)
	}
}

func TestParseV2Schema_PDBSection(t *testing.T) {
	contents := `
[app]
name = "myapp"

[pdb]
enabled = true
type = "percent"
percent = 60
`
	_, path := writeToml(t, contents)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.PDB.Enabled {
		t.Error("PDB.Enabled = false, want true")
	}
	if cfg.PDB.Type != "percent" {
		t.Errorf("PDB.Type = %q, want percent", cfg.PDB.Type)
	}
	if cfg.PDB.Percent != 60 {
		t.Errorf("PDB.Percent = %d, want 60", cfg.PDB.Percent)
	}
}

func TestParseV2Schema_MulticlusterSection(t *testing.T) {
	contents := `
[app]
name = "myapp"

[multicluster]
enabled = true
mode = "active-active"
backup_enabled = true
backup_schedule = "hourly"
backup_retention = "72h"
backup_priority_cluster = 2
`
	_, path := writeToml(t, contents)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.Multicluster.Enabled {
		t.Error("Multicluster.Enabled = false, want true")
	}
	if cfg.Multicluster.Mode != "active-active" {
		t.Errorf("Multicluster.Mode = %q, want active-active", cfg.Multicluster.Mode)
	}
	if !cfg.Multicluster.BackupEnabled {
		t.Error("Multicluster.BackupEnabled = false, want true")
	}
	if cfg.Multicluster.BackupSchedule != "hourly" {
		t.Errorf("Multicluster.BackupSchedule = %q, want hourly", cfg.Multicluster.BackupSchedule)
	}
	if cfg.Multicluster.BackupRetention != "72h" {
		t.Errorf("Multicluster.BackupRetention = %q, want 72h", cfg.Multicluster.BackupRetention)
	}
	if cfg.Multicluster.BackupPriorityCluster != 2 {
		t.Errorf("Multicluster.BackupPriorityCluster = %d, want 2", cfg.Multicluster.BackupPriorityCluster)
	}
}

func TestParseV2Schema_AllSectionsTogether(t *testing.T) {
	// End-to-end sanity: a fully-populated v2 config parses without error and
	// every section is reachable from a single cfg object.
	contents := `
[app]
name = "fullapp"
port = 8080
cpu = "1"
memory = "1Gi"
replicas = 3
zone = "my-bki-1a"
strategy = "rolling"

[volume]
size = "20Gi"
mount = "/var/data"

[hpa]
enabled = true
min_replicas = 1
max_replicas = 5
cpu_target = 70

[pdb]
enabled = true
type = "fixed"
min_available = 1

[multicluster]
enabled = false
`
	_, path := writeToml(t, contents)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.App.Name != "fullapp" || cfg.Volume.Size != "20Gi" || !cfg.HPA.Enabled || cfg.PDB.Type != "fixed" || cfg.Multicluster.Enabled {
		t.Errorf("Combined parse mismatch: %+v", cfg)
	}
}

func TestParseV2Schema_BackwardsCompatibleV1(t *testing.T) {
	// A pre-v2 toml (only [app] with the original 7 fields) must still parse
	// without errors and leave the new sections zero-valued.
	contents := `
[app]
name = "v1app"
port = 8080
cpu = "0.5"
memory = "256Mi"
domain = "v1.example.com"
`
	_, path := writeToml(t, contents)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.App.Name != "v1app" || cfg.App.Domain != "v1.example.com" {
		t.Errorf("v1 fields lost: %+v", cfg.App)
	}
	if cfg.HPA.Enabled || cfg.Volume.Size != "" || cfg.Multicluster.Enabled {
		t.Errorf("v2 sections should be zero-valued on v1 input: %+v", cfg)
	}
}
