// Package deploy defines the "1ctl deploy" command tree — flag names,
// input structs, and CLI wiring. Handler logic lives in handlers.go.
package deploy

import (
	"context"
	"strings"

	"github.com/urfave/cli/v3"
)

// --- Flag name constants ------------------------------------------------

const (
	flagName                = "name"
	flagCPU                 = "cpu"
	flagCPURequest          = "cpu-request"
	flagCPULimit            = "cpu-limit"
	flagMemory              = "memory"
	flagMachine             = "machine"
	flagMachineTag          = "machine-tag"

	flagDomain              = "domain"
	flagOrganization        = "organization"
	flagHealthPath          = "health-path"
	flagDockerfile          = "dockerfile"
	flagImage               = "image"
	flagFast                = "fast"
	flagPort                = "port"
	flagEnv                 = "env"
	flagVolumeSize          = "volume-size"
	flagVolumeMount         = "volume-mount"
	flagVolumeStorageClass  = "volume-storage-class"
	flagZone                = "zone"
	flagMulticluster        = "multi-cluster"
	flagMulticlusterMode    = "multicluster-mode"
	flagBackupEnabled       = "backup-enabled"
	flagBackupSchedule      = "backup-schedule"
	flagBackupRetention     = "backup-retention"
	flagBackupPriority      = "backup-priority-cluster"
	flagReplicas            = "replicas"
	flagPDB                 = "pdb"
	flagPDBType             = "pdb-type"
	flagPDBMinAvailable     = "pdb-min-available"
	flagPDBPercent          = "pdb-percent"
	flagHPA                 = "hpa"
	flagHPAMinReplicas      = "hpa-min-replicas"
	flagHPAMaxReplicas      = "hpa-max-replicas"
	flagHPACPUCoreTarget    = "hpa-cpu-target"
	flagHPAMemoryTarget     = "hpa-memory-target"
	flagVPA                 = "vpa"
	flagVPAMode             = "vpa-mode"
	flagVPAMinCPU           = "vpa-min-cpu"
	flagVPAMaxCPU           = "vpa-max-cpu"
	flagVPAMinMemory        = "vpa-min-memory"
	flagVPAMaxMemory        = "vpa-max-memory"
	flagWaitFor             = "wait-for"
	flagStrategy            = "strategy"
	flagRollingMaxSurge     = "rolling-max-surge"
	flagRollingMaxUnavail   = "rolling-max-unavailable"
	flagConfig              = "config"
	flagDeploymentID        = "deployment-id"
	flagApp                 = "app"
	flagYes                 = "yes"
	flagVersion             = "version"
	flagWatch               = "watch"

)

// --- Input structs ------------------------------------------------------

// DeployInput holds all flags for the main deploy action.
type DeployInput struct {
	Name                 string
	CPU                  string
	CPURequest           string
	CPULimit             string
	Memory               string
	Machine              []string
	MachineTag           string
	Domain               string
	Organization         string
	HealthPath           string
	Dockerfile           string
	Image                string
	Fast                 bool
	Port                 int
	Env                  []string
	VolumeSize           string
	VolumeMount          string
	VolumeStorageClass   string
	Zone                 string
	Multicluster         bool
	MulticlusterMode     string
	BackupEnabled        bool
	BackupSchedule       string
	BackupRetention      string
	BackupPriority       int
	Replicas             int
	PDB                  bool
	PDBType              string
	PDBMinAvailable      int
	PDBPercent           int
	HPA                  bool
	HPAMinReplicas       int
	HPAMaxReplicas       int
	HPACPUCoreTarget     int
	HPAMemoryTarget      int
	VPA                  bool
	VPAMode              string
	VPAMinCPU            string
	VPAMaxCPU            string
	VPAMinMemory         string
	VPAMaxMemory         string
	WaitFor              []string
	Strategy             string
	RollingMaxSurge      string
	RollingMaxUnavail    string
	Config               string
}

// GetDeploymentInput holds flags for the "get" subcommand.
type GetDeploymentInput struct {
	DeploymentID string
	App          string
	Config       string
}

// StatusInput holds flags for the "status" subcommand.
type StatusInput struct {
	DeploymentID string
	App          string
	Config       string
	Watch        bool
}

// DestroyInput holds flags for the "delete" subcommand.
type DestroyInput struct {
	DeploymentID  string
	App           string
	Config        string
	Yes           bool
	RetainVolumes bool // if true, skip PVC destruction
}

// DeployRefInput holds common deployment-reference flags.
type DeployRefInput struct {
	DeploymentID string
	App          string
	Config       string
}

// RollbackInput holds flags for the "rollback" subcommand.
type RollbackInput struct {
	DeploymentID string
	App          string
	Config       string
	Version      int
	Yes          bool
}

// ScaleInput holds flags for the "scale" subcommand.
type ScaleInput struct {
	DeploymentID string
	App          string
	Config       string
	Replicas     int
}

// --- Helper flag constructors ------------------------------------------

func optionalString(name, usage string, dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
	}
}

func optionalStringVal(name, usage, defaultValue string, dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        name,
		Usage:       usage,
		Value:       defaultValue,
		Destination: dest,
	}
}

func optionalBool(name, usage string, dest *bool) *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
	}
}

func optionalBoolVal(name, usage string, defaultValue bool, dest *bool) *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:        name,
		Usage:       usage,
		Value:       defaultValue,
		Destination: dest,
	}
}

func optionalInt(name, usage string, dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
	}
}

func optionalIntVal(name, usage string, defaultValue int, dest *int) *cli.IntFlag {
	return &cli.IntFlag{
		Name:        name,
		Usage:       usage,
		Value:       defaultValue,
		Destination: dest,
	}
}

func optionalStringSlice(name, usage string, dest *[]string) *cli.StringSliceFlag {
	return &cli.StringSliceFlag{
		Name:        name,
		Usage:       usage,
		Destination: dest,
	}
}

// --- Command tree -------------------------------------------------------

// Command returns the root deploy command tree.
func Command() *cli.Command {
	var in DeployInput
	return &cli.Command{
		Name:  "deploy",
		Usage: "Build and deploy an application",
		Description: `Build and deploy the current project to SatuSky Cloud.

Images are built in the cloud — no local Docker installation required.

Examples:
   1ctl deploy --port 8080
   1ctl deploy --name api --port 8080 --memory 512Mi
   1ctl deploy --image ghcr.io/acme/api:v1 --port 8080
   1ctl deploy --machine-tag production --port 8080

To manage a deployed application, use "1ctl app".`,
		Flags: deployFlags(&in),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				return cli.ShowSubcommandHelp(cmd)
			}
			return handleDeploy(ctx, in)
		},
	}
}

func deployFlags(in *DeployInput) []cli.Flag {
	return []cli.Flag{
		// ── Build ──
		optionalStringVal(flagDockerfile, "Dockerfile path for cloud build (default: Dockerfile)", "Dockerfile", &in.Dockerfile),
		optionalString(flagImage, "Pre-built image reference — skips cloud build entirely", &in.Image),
		optionalBool(flagFast, "Use the accelerated cloud build backend (ignored when --image is set)", &in.Fast),
		// ── App ──
		optionalString(flagName, "Application name (auto-detected from satusky.toml or git remote)", &in.Name),
		optionalIntVal(flagPort, "Application port", 8080, &in.Port),
		optionalString(flagDomain, "Custom domain (default: *.satusky.com)", &in.Domain),
		optionalStringSlice(flagEnv, "Environment variables (format: KEY=VALUE)", &in.Env),
		optionalString(flagConfig, "Config name or path (e.g. staging, satusky.staging.toml)", &in.Config),
		// ── Resources ──
		optionalStringVal(flagCPURequest, "Guaranteed CPU reservation per replica (e.g. '250m')", "250m", &in.CPURequest),
		optionalStringVal(flagCPULimit, "Maximum burst CPU per replica (e.g. '1')", "1", &in.CPULimit),
		optionalStringVal(flagMemory, "Memory allocation (e.g. '512Mi', '2Gi')", "256Mi", &in.Memory),
		optionalInt(flagReplicas, "Replica count override (default: auto from machine count)", &in.Replicas),
		optionalStringSlice(flagMachine, "Explicit machine name (BYOA). Repeatable.", &in.Machine),
		optionalString(flagMachineTag, "Tag expression using & (AND), | (OR), = (equals), (). Bare key checks value=\"true\".", &in.MachineTag),
		// ── Storage ──
		optionalString(flagVolumeSize, "Storage size (e.g. '10Gi')", &in.VolumeSize),
		optionalString(flagVolumeMount, "Storage mount path", &in.VolumeMount),
		optionalStringVal(flagVolumeStorageClass, "Storage class for volumes", "ceph-block", &in.VolumeStorageClass),
		// ── Placement ──
		optionalString(flagZone, "Target deployment zone (e.g. 'my-kul-1b')", &in.Zone),
		&cli.BoolFlag{
			Name:    flagMulticluster,
			Aliases: []string{"multicluster"},
			Usage:   "Enable multi-cluster deployment across KL and BKI clusters",
			Destination: &in.Multicluster,
		},
		optionalStringVal(flagMulticlusterMode, "Multi-cluster mode: 'active-active' or 'active-passive'", "active-passive", &in.MulticlusterMode),
		// ── Reliability ──
		optionalString(flagHealthPath, "HTTP path for post-deploy smoke test (default: tries /health then /)", &in.HealthPath),
		optionalStringSlice(flagWaitFor, "TCP dependency to wait for (format: host:port). Repeatable.", &in.WaitFor),
		optionalStringVal(flagStrategy, "Rollout strategy: rolling, recreate", "rolling", &in.Strategy),
		optionalStringVal(flagRollingMaxSurge, "Rolling update max surge (pods or %)", "25%", &in.RollingMaxSurge),
		optionalStringVal(flagRollingMaxUnavail, "Rolling update max unavailable (pods or %)", "25%", &in.RollingMaxUnavail),
		// ── Autoscaling ──
		optionalBool(flagHPA, "Enable HorizontalPodAutoscaler", &in.HPA),
		optionalIntVal(flagHPAMinReplicas, "HPA minimum replicas", 1, &in.HPAMinReplicas),
		optionalIntVal(flagHPAMaxReplicas, "HPA maximum replicas", 10, &in.HPAMaxReplicas),
		optionalIntVal(flagHPACPUCoreTarget, "HPA target CPU utilization %", 80, &in.HPACPUCoreTarget),
		optionalIntVal(flagHPAMemoryTarget, "HPA target memory utilization % (0 = disabled)", 0, &in.HPAMemoryTarget),
		optionalBool(flagVPA, "Enable VerticalPodAutoscaler", &in.VPA),
		optionalStringVal(flagVPAMode, "VPA update mode: 'Off', 'Initial', 'Auto'", "Off", &in.VPAMode),
		optionalString(flagVPAMinCPU, "VPA minimum CPU (e.g. '100m')", &in.VPAMinCPU),
		optionalString(flagVPAMaxCPU, "VPA maximum CPU (e.g. '4')", &in.VPAMaxCPU),
		optionalString(flagVPAMinMemory, "VPA minimum memory (e.g. '128Mi')", &in.VPAMinMemory),
		optionalString(flagVPAMaxMemory, "VPA maximum memory (e.g. '8Gi')", &in.VPAMaxMemory),
		// ── Backups ──
		optionalBoolVal(flagBackupEnabled, "Enable backups (auto-enabled for active-passive)", true, &in.BackupEnabled),
		optionalStringVal(flagBackupSchedule, "Backup frequency: 'hourly', 'daily', 'weekly'", "daily", &in.BackupSchedule),
		optionalStringVal(flagBackupRetention, "Backup retention: '24h', '72h', '168h', '720h'", "168h", &in.BackupRetention),
		optionalIntVal(flagBackupPriority, "Which cluster performs backups: 1 (KL) or 2 (BKI)", 1, &in.BackupPriority),
		// ── Legacy ──
		optionalString(flagCPU, "Legacy alias for --cpu-limit (deprecated)", &in.CPU),
		optionalString(flagOrganization, "Organization name (rarely needed)", &in.Organization),
		optionalBool(flagPDB, "Enable PodDisruptionBudget (auto-enabled when replicas > 1)", &in.PDB),
		optionalStringVal(flagPDBType, "PDB type: 'auto', 'fixed', 'percent'", "auto", &in.PDBType),
		optionalInt(flagPDBMinAvailable, "PDB minimum available pods (for type=fixed)", &in.PDBMinAvailable),
		optionalInt(flagPDBPercent, "PDB minimum available %% (for type=percent)", &in.PDBPercent),
	}
}



// looksLikeUUID reports whether s looks like a standard UUID (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
// This is used to distinguish positional args that are deployment IDs from those
// that are app names.
func looksLikeUUID(s string) bool {
	return len(s) == 36 && strings.Count(s, "-") == 4
}
