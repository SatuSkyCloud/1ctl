package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"1ctl/internal/validator"
	"github.com/BurntSushi/toml"
)

const DefaultConfigFile = "satusky.toml"

// ProjectConfig is the in-memory representation of satusky.toml.
//
// Field-by-field precedence at deploy time (highest wins):
//
//	CLI flag (explicit, c.IsSet) > satusky.toml > platform default (Value: on flag)
//
// Fields with the zero value are treated as "not set" by the merge.
type ProjectConfig struct {
	App          AppConfig          `toml:"app"`
	Build        BuildConfig        `toml:"build"`
	Checks       ChecksConfig       `toml:"checks"`
	Secrets      SecretsConfig      `toml:"secrets"`
	Deploy       DeployConfig       `toml:"deploy"`
	Volume       VolumeConfig       `toml:"volume"`
	Volumes      []VolumeConfig     `toml:"volumes"`
	HPA          HPAConfig          `toml:"hpa"`
	VPA          VPAConfig          `toml:"vpa"`
	PDB          PDBConfig          `toml:"pdb"`
	Env          EnvConfig          `toml:"env"`
	Multicluster MulticlusterConfig `toml:"multicluster"`
	Path         string             `toml:"-"`

	legacyVolumeSet bool
}

// AppConfig holds app identity and resource fields.
// For build, health-check, and deploy-strategy settings see [build], [checks], [deploy].
type AppConfig struct {
	Name         string `toml:"name"`
	Port         int    `toml:"port"`
	CPU          string `toml:"cpu"` // Deprecated: legacy burst CPU alias.
	CPURequest   string `toml:"cpu_request"`
	CPULimit     string `toml:"cpu_limit"`
	Memory       string `toml:"memory"`
	Replicas     int    `toml:"replicas"`
	Domain       string `toml:"domain"`
	Zone         string `toml:"zone"`
	Organization string `toml:"organization"`

	// Backward-compat: fields below were moved to [build], [checks], or [deploy] in the v2 schema.
	// Normalize() copies them to the preferred location when the target section is empty.
	Dockerfile            string   `toml:"dockerfile"`
	FastBuild             bool     `toml:"fast_build"`
	HealthPath            string   `toml:"health_path"`
	Strategy              string   `toml:"strategy"`
	RollingMaxSurge       string   `toml:"rolling_max_surge"`
	RollingMaxUnavailable string   `toml:"rolling_max_unavailable"`
	MachineTag            string   `toml:"machine_tag"`
	WaitFor               []string `toml:"wait_for"`
}

// BuildConfig controls how the container image is built.
type BuildConfig struct {
	Dockerfile string `toml:"dockerfile"`
	Image      string `toml:"image"`
	TargetArch string `toml:"target_arch"`
	FastBuild  bool   `toml:"fast_build"`
}

// ChecksConfig controls deployment health checks and smoke testing.
type ChecksConfig struct {
	HealthPath string       `toml:"health_path"`
	Startup    *ProbeConfig `toml:"startup"`
	Readiness  *ProbeConfig `toml:"readiness"`
	Liveness   *ProbeConfig `toml:"liveness"`
}

// ProbeConfig is the Kubernetes-compatible subset of a container probe.
// Exactly one handler must be configured when a probe is declared.
type ProbeConfig struct {
	HTTPGet   *HTTPGetProbeConfig   `toml:"http_get"`
	TCPSocket *TCPSocketProbeConfig `toml:"tcp_socket"`
	Exec      *ExecProbeConfig      `toml:"exec"`

	InitialDelaySeconds *int32 `toml:"initial_delay_seconds"`
	TimeoutSeconds      *int32 `toml:"timeout_seconds"`
	PeriodSeconds       *int32 `toml:"period_seconds"`
	SuccessThreshold    *int32 `toml:"success_threshold"`
	FailureThreshold    *int32 `toml:"failure_threshold"`
}

type HTTPGetProbeConfig struct {
	Path string `toml:"path"`
	Port int32  `toml:"port"`
}

type TCPSocketProbeConfig struct {
	Port int32 `toml:"port"`
}

type ExecProbeConfig struct {
	Command []string `toml:"command"`
}

// SecretsConfig declares required secret environment-variable keys only.
// Values remain managed outside satusky.toml.
type SecretsConfig struct {
	Required []string `toml:"required"`
}

// DeployConfig controls deployment strategy and runtime placement.
type DeployConfig struct {
	Strategy              string   `toml:"strategy"`
	RollingMaxSurge       string   `toml:"rolling_max_surge"`
	RollingMaxUnavailable string   `toml:"rolling_max_unavailable"`
	MachineTag            string   `toml:"machine_tag"`
	WaitFor               []string `toml:"wait_for"`
}

type VolumeConfig struct {
	Name         string `toml:"name"`
	Claim        string `toml:"claim"`
	Size         string `toml:"size"`
	Mount        string `toml:"mount"`
	StorageClass string `toml:"storage_class"`
}

var environmentVariableNamePattern = regexp.MustCompile(`^[-._A-Za-z][-._A-Za-z0-9]*$`)

type HPAConfig struct {
	Enabled      bool  `toml:"enabled"`
	MinReplicas  int32 `toml:"min_replicas"`
	MaxReplicas  int32 `toml:"max_replicas"`
	CPUTarget    int32 `toml:"cpu_target"`
	MemoryTarget int32 `toml:"memory_target"`
}

type VPAConfig struct {
	Enabled   bool   `toml:"enabled"`
	Mode      string `toml:"mode"`
	MinCPU    string `toml:"min_cpu"`
	MaxCPU    string `toml:"max_cpu"`
	MinMemory string `toml:"min_memory"`
	MaxMemory string `toml:"max_memory"`
}

// PDBConfig fields match the API surface (api.PDBConfig): only MinAvailable
// and Percent are accepted today. MaxUnavailable is intentionally not on the
// struct — the platform doesn't yet support it and silently dropping the
// field would surprise users who set it expecting it to work.
type PDBConfig struct {
	Enabled      bool   `toml:"enabled"`
	Type         string `toml:"type"`
	MinAvailable int32  `toml:"min_available"`
	Percent      int32  `toml:"percent"`
}

// EnvConfig holds environment variables defined in satusky.toml [env] section.
// These are non-sensitive runtime configuration values, following Fly.io's
// convention where env vars live in the config file while secrets are managed
// through the CLI (1ctl secret create).
type EnvConfig map[string]string

type MulticlusterConfig struct {
	Enabled               bool   `toml:"enabled"`
	Mode                  string `toml:"mode"`
	BackupEnabled         bool   `toml:"backup_enabled"`
	BackupSchedule        string `toml:"backup_schedule"`
	BackupRetention       string `toml:"backup_retention"`
	BackupPriorityCluster int    `toml:"backup_priority_cluster"`
}

// LoadConfig resolves and loads the config file. Returns an error if not found.
func LoadConfig(configArg string) (*ProjectConfig, error) {
	path, err := resolveConfigPath(configArg)
	if err != nil {
		return nil, err
	}
	return decodeProjectConfig(path)
}

// FindConfig looks for a config file without requiring one to exist. Returns nil, nil if not found.
func FindConfig(configArg string) (*ProjectConfig, error) {
	path, err := resolveConfigPath(configArg)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return decodeProjectConfig(path)
}

func decodeProjectConfig(path string) (*ProjectConfig, error) {
	var cfg ProjectConfig
	metadata, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", path, err)
	}
	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, key := range undecoded {
			keys = append(keys, key.String())
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("invalid %s: unknown configuration keys: %s", path, strings.Join(keys, ", "))
	}
	cfg.Path = path
	cfg.legacyVolumeSet = metadata.IsDefined("volume")
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", path, err)
	}
	return &cfg, nil
}

// Normalize applies backward-compatible field migrations.
// When the preferred section ([build], [checks], [deploy]) is empty/zero,
// values are copied from the legacy [app] location so old satusky.toml files
// continue to work without changes. The legacy [app] fields are then cleared
// so downstream consumers always read from the canonical v2 sections.
func (cfg *ProjectConfig) Normalize() {
	// Build: prefer [build] over [app].
	if cfg.Build.Dockerfile == "" {
		cfg.Build.Dockerfile = cfg.App.Dockerfile
	}
	cfg.Build.Dockerfile = strings.TrimSpace(cfg.Build.Dockerfile)
	cfg.Build.Image = strings.TrimSpace(cfg.Build.Image)
	cfg.Build.TargetArch = strings.TrimSpace(cfg.Build.TargetArch)
	if !cfg.Build.FastBuild {
		cfg.Build.FastBuild = cfg.App.FastBuild
	}
	// Checks: prefer [checks] over [app].
	if cfg.Checks.HealthPath == "" {
		cfg.Checks.HealthPath = cfg.App.HealthPath
	}
	// Deploy: prefer [deploy] over [app].
	if cfg.Deploy.Strategy == "" {
		cfg.Deploy.Strategy = cfg.App.Strategy
	}
	if cfg.Deploy.RollingMaxSurge == "" {
		cfg.Deploy.RollingMaxSurge = cfg.App.RollingMaxSurge
	}
	if cfg.Deploy.RollingMaxUnavailable == "" {
		cfg.Deploy.RollingMaxUnavailable = cfg.App.RollingMaxUnavailable
	}
	if cfg.Deploy.MachineTag == "" {
		cfg.Deploy.MachineTag = cfg.App.MachineTag
	}
	if len(cfg.Deploy.WaitFor) == 0 {
		cfg.Deploy.WaitFor = cfg.App.WaitFor
	}
	if cfg.legacyVolumeSet {
		cfg.Volumes = append(cfg.Volumes, VolumeConfig{
			Name:         legacyVolumeName(cfg.App.Name),
			Claim:        legacyVolumeClaim(cfg.App.Name),
			Size:         cfg.Volume.Size,
			Mount:        cfg.Volume.Mount,
			StorageClass: "ceph-block",
		})
	}

	// Clear legacy [app] fields so downstream consumers always read from
	// the canonical v2 sections.
	cfg.App.Dockerfile = ""
	cfg.App.FastBuild = false
	cfg.App.HealthPath = ""
	cfg.App.Strategy = ""
	cfg.App.RollingMaxSurge = ""
	cfg.App.RollingMaxUnavailable = ""
	cfg.App.MachineTag = ""
	cfg.App.WaitFor = nil
}

// Validate rejects build settings that cannot be applied unambiguously.
func (cfg *ProjectConfig) Validate() error {
	if cfg.Build.Image != "" && cfg.Build.Dockerfile != "" {
		return fmt.Errorf("[build].image and [build].dockerfile are mutually exclusive; remove dockerfile to deploy a pre-built image")
	}
	switch cfg.Build.TargetArch {
	case "", "amd64", "arm64":
	default:
		return fmt.Errorf("[build].target_arch must be empty, \"amd64\", or \"arm64\" (got %q)", cfg.Build.TargetArch)
	}
	if cfg.legacyVolumeSet && len(cfg.Volumes) > 1 {
		return fmt.Errorf("[volume] and [[volumes]] cannot be used together")
	}
	if err := validateSecrets(cfg.Secrets); err != nil {
		return err
	}
	if err := validateProbe("[checks.startup]", cfg.Checks.Startup, true); err != nil {
		return err
	}
	if err := validateProbe("[checks.readiness]", cfg.Checks.Readiness, false); err != nil {
		return err
	}
	if err := validateProbe("[checks.liveness]", cfg.Checks.Liveness, true); err != nil {
		return err
	}
	if err := validateVolumes(cfg.Volumes); err != nil {
		return err
	}
	return nil
}

// UsesCanonicalVolumes reports whether the source config used [[volumes]].
// Callers use this to avoid silently dropping repeated volume declarations
// when they must fall back to an older deployment workflow.
func (cfg *ProjectConfig) UsesCanonicalVolumes() bool {
	return !cfg.legacyVolumeSet && len(cfg.Volumes) > 0
}

func legacyVolumeName(appName string) string {
	if appName == "" {
		return "volume"
	}
	return appName + "-volume"
}

func legacyVolumeClaim(appName string) string {
	if appName == "" {
		return "claim"
	}
	return appName + "-claim"
}

func validateSecrets(secrets SecretsConfig) error {
	seen := make(map[string]struct{}, len(secrets.Required))
	for _, key := range secrets.Required {
		if !environmentVariableNamePattern.MatchString(key) || key == "." || key == ".." || strings.HasPrefix(key, "..") {
			return fmt.Errorf("[secrets].required key %q must be a Kubernetes environment-variable name", key)
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("[secrets].required key %q is declared more than once", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateProbe(name string, probe *ProbeConfig, successThresholdMustBeOne bool) error {
	if probe == nil {
		return nil
	}

	handlers := 0
	if probe.HTTPGet != nil {
		handlers++
		if !strings.HasPrefix(probe.HTTPGet.Path, "/") {
			return fmt.Errorf("%s.http_get.path must start with /", name)
		}
		if err := validateProbePort(name+".http_get.port", probe.HTTPGet.Port); err != nil {
			return err
		}
	}
	if probe.TCPSocket != nil {
		handlers++
		if err := validateProbePort(name+".tcp_socket.port", probe.TCPSocket.Port); err != nil {
			return err
		}
	}
	if probe.Exec != nil {
		handlers++
		if len(probe.Exec.Command) == 0 {
			return fmt.Errorf("%s.exec.command must not be empty", name)
		}
		for _, command := range probe.Exec.Command {
			if strings.TrimSpace(command) == "" {
				return fmt.Errorf("%s.exec.command must not contain an empty argument", name)
			}
		}
	}
	if handlers != 1 {
		return fmt.Errorf("%s must declare exactly one handler", name)
	}

	if probe.InitialDelaySeconds != nil && *probe.InitialDelaySeconds < 0 {
		return fmt.Errorf("%s.initial_delay_seconds must be nonnegative", name)
	}
	for field, value := range map[string]*int32{
		"timeout_seconds":   probe.TimeoutSeconds,
		"period_seconds":    probe.PeriodSeconds,
		"success_threshold": probe.SuccessThreshold,
		"failure_threshold": probe.FailureThreshold,
	} {
		if value != nil && *value < 1 {
			return fmt.Errorf("%s.%s must be at least 1", name, field)
		}
	}
	if successThresholdMustBeOne && probe.SuccessThreshold != nil && *probe.SuccessThreshold != 1 {
		return fmt.Errorf("%s.success_threshold must be 1", name)
	}
	return nil
}

func validateProbePort(name string, port int32) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535", name)
	}
	return nil
}

func validateVolumes(volumes []VolumeConfig) error {
	names := make(map[string]struct{}, len(volumes))
	claims := make(map[string]struct{}, len(volumes))
	mounts := make(map[string]struct{}, len(volumes))
	for index, volume := range volumes {
		prefix := fmt.Sprintf("[[volumes]][%d]", index)
		for field, value := range map[string]string{
			"name":          volume.Name,
			"claim":         volume.Claim,
			"size":          volume.Size,
			"mount":         volume.Mount,
			"storage_class": volume.StorageClass,
		} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s.%s is required", prefix, field)
			}
		}
		if err := validator.ValidateMemory(volume.Size); err != nil {
			return fmt.Errorf("%s.size must be a positive Kubernetes quantity: %w", prefix, err)
		}
		if !strings.HasPrefix(volume.Mount, "/") {
			return fmt.Errorf("%s.mount must start with /", prefix)
		}
		if _, duplicate := names[volume.Name]; duplicate {
			return fmt.Errorf("duplicate volume name %q", volume.Name)
		}
		if _, duplicate := claims[volume.Claim]; duplicate {
			return fmt.Errorf("duplicate volume claim %q", volume.Claim)
		}
		if _, duplicate := mounts[volume.Mount]; duplicate {
			return fmt.Errorf("duplicate volume mount %q", volume.Mount)
		}
		names[volume.Name] = struct{}{}
		claims[volume.Claim] = struct{}{}
		mounts[volume.Mount] = struct{}{}
	}
	return nil
}

// Save writes the config back to its original path.
func (cfg *ProjectConfig) Save() error {
	f, err := os.Create(cfg.Path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }() //nolint:errcheck
	toWrite := *cfg
	if cfg.legacyVolumeSet {
		// Normalize exposes the legacy volume through Volumes for consumers of
		// the canonical schema, but writing both declarations would be invalid.
		toWrite.Volumes = nil
	}
	return toml.NewEncoder(f).Encode(&toWrite)
}

func resolveConfigPath(configArg string) (string, error) {
	if configArg != "" {
		if strings.HasSuffix(configArg, ".toml") {
			if _, err := os.Stat(configArg); err != nil {
				return "", err
			}
			return configArg, nil
		}
		base, err := findDefaultConfigDir()
		if err != nil {
			return "", fmt.Errorf("satusky.toml not found; cannot resolve --config %s", configArg)
		}
		path := filepath.Join(base, fmt.Sprintf("satusky.%s.toml", configArg))
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("%s not found", path)
		}
		return path, nil
	}
	dir, err := findDefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultConfigFile), nil
}

func findDefaultConfigDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, DefaultConfigFile)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}
