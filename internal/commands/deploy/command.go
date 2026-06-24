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
flagMachineTagStrategy = "machine-tag-strategy"
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
	MachineTag           []string
	MachineTagStrategy   string
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
		Usage: "Deploy your application to SatuSky Cloud",
		Description: `Build and deploy your application to SatuSky Cloud.

Images are built in the cloud — no local Docker installation required.

   1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 512Mi --port 8080`,
		Flags: deployFlags(&in),
		Commands: []*cli.Command{
			listCommand(),
			getCommand(),
			statusCommand(),
			destroyCommand(),
			restartCommand(),
			releasesCommand(),
			rollbackCommand(),
			openCommand(),
			scaleCommand(),
		},
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
		optionalString(flagName, "Application name (default: auto-detected from satusky.toml or git remote)", &in.Name),
		optionalString(flagCPU, "Legacy alias for --cpu-limit (e.g., '1', '500m')", &in.CPU),
		optionalStringVal(flagCPURequest, "Guaranteed CPU reservation per replica (e.g., '250m')", "250m", &in.CPURequest),
		optionalStringVal(flagCPULimit, "Maximum burst CPU per replica (e.g., '1')", "1", &in.CPULimit),
		optionalStringVal(flagMemory, "Memory allocation (e.g., '512Mi', '2Gi')", "256Mi", &in.Memory),
		optionalStringSlice(flagMachine, "Explicit machine name (BYOA). Repeatable for multi-machine.", &in.Machine),
		optionalStringSlice(flagMachineTag, "Deploy to your machines labelled with this tag (BYOA). Repeatable for AND logic.", &in.MachineTag),
		optionalStringVal(flagMachineTagStrategy, "Tag matching strategy: 'and' (all tags required) or 'or' (any tag matches)", "and", &in.MachineTagStrategy),
		optionalString(flagDomain, "Custom domain (default: *.satusky.com)", &in.Domain),
		optionalString(flagOrganization, "Organization name (default: current organization)", &in.Organization),
		optionalString(flagHealthPath, "HTTP path to smoke test after deploy wait succeeds (default: tries /health then /)", &in.HealthPath),
		optionalStringVal(flagDockerfile, "Dockerfile path for cloud build (default: Dockerfile)", "Dockerfile", &in.Dockerfile),
		optionalString(flagImage, "Pre-built image reference — skips cloud build entirely", &in.Image),
		optionalBool(flagFast, "Use the accelerated cloud build backend before deploying (ignored when --image is set)", &in.Fast),
		optionalIntVal(flagPort, "Application port (default: 8080)", 8080, &in.Port),
		optionalStringSlice(flagEnv, "Environment variables (format: KEY=VALUE)", &in.Env),
		optionalString(flagVolumeSize, "Storage size (e.g., '10Gi')", &in.VolumeSize),
		optionalString(flagVolumeMount, "Storage mount path", &in.VolumeMount),
		optionalStringVal(flagVolumeStorageClass, "Storage class for volumes (e.g., 'ceph-block')", "ceph-block", &in.VolumeStorageClass),
		optionalString(flagZone, "Target deployment zone (e.g., 'my-kul-1b', 'my-bki-1a')", &in.Zone),
		&cli.BoolFlag{
			Name:    flagMulticluster,
			Aliases: []string{"multicluster"},
			Usage:   "Enable multi-cluster deployment across KL and BKI clusters",
			Destination: &in.Multicluster,
		},
		optionalStringVal(flagMulticlusterMode, "Multi-cluster mode: 'active-active' or 'active-passive'", "active-passive", &in.MulticlusterMode),
		optionalBoolVal(flagBackupEnabled, "Enable backups (auto-enabled for active-passive)", true, &in.BackupEnabled),
		optionalStringVal(flagBackupSchedule, "Backup frequency: 'hourly', 'daily', 'weekly'", "daily", &in.BackupSchedule),
		optionalStringVal(flagBackupRetention, "Backup retention: '24h', '72h', '168h', '720h'", "168h", &in.BackupRetention),
		optionalIntVal(flagBackupPriority, "Which cluster performs backups: 1 (Primary/KL) or 2 (Secondary/BKI)", 1, &in.BackupPriority),
		optionalInt(flagReplicas, "Manual replica count override (default: auto from machine count)", &in.Replicas),
		optionalBool(flagPDB, "Enable PodDisruptionBudget (auto-enabled when replicas > 1)", &in.PDB),
		optionalStringVal(flagPDBType, "PDB type: 'auto', 'fixed', 'percent'", "auto", &in.PDBType),
		optionalInt(flagPDBMinAvailable, "Minimum available pods for PDB type=fixed", &in.PDBMinAvailable),
		optionalInt(flagPDBPercent, "Minimum available percentage for PDB type=percent (1-100)", &in.PDBPercent),
		optionalBool(flagHPA, "Enable HorizontalPodAutoscaler", &in.HPA),
		optionalIntVal(flagHPAMinReplicas, "HPA minimum replicas", 1, &in.HPAMinReplicas),
		optionalIntVal(flagHPAMaxReplicas, "HPA maximum replicas", 10, &in.HPAMaxReplicas),
		optionalIntVal(flagHPACPUCoreTarget, "HPA target CPU utilization percentage", 80, &in.HPACPUCoreTarget),
		optionalIntVal(flagHPAMemoryTarget, "HPA target memory utilization percentage (0 = disabled)", 0, &in.HPAMemoryTarget),
		optionalBool(flagVPA, "Enable VerticalPodAutoscaler", &in.VPA),
		optionalStringVal(flagVPAMode, "VPA update mode: 'Off', 'Initial', 'Auto'", "Off", &in.VPAMode),
		optionalString(flagVPAMinCPU, "VPA minimum CPU (e.g., '100m')", &in.VPAMinCPU),
		optionalString(flagVPAMaxCPU, "VPA maximum CPU (e.g., '4')", &in.VPAMaxCPU),
		optionalString(flagVPAMinMemory, "VPA minimum memory (e.g., '128Mi')", &in.VPAMinMemory),
		optionalString(flagVPAMaxMemory, "VPA maximum memory (e.g., '8Gi')", &in.VPAMaxMemory),
		optionalStringSlice(flagWaitFor, "Wait for a TCP dependency (format: host:port). Repeatable.", &in.WaitFor),
		optionalStringVal(flagStrategy, "Deployment rollout strategy: rolling (default), recreate", "rolling", &in.Strategy),
		optionalStringVal(flagRollingMaxSurge, "Rolling update max surge. Pods or percentage", "25%", &in.RollingMaxSurge),
		optionalStringVal(flagRollingMaxUnavail, "Rolling update max unavailable. Pods or percentage", "25%", &in.RollingMaxUnavail),
		optionalString(flagConfig, "Config name or path (e.g. staging, satusky.staging.toml)", &in.Config),
	}
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List deployments",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleListDeployments(ctx)
		},
	}
}

func getCommand() *cli.Command {
	var in GetDeploymentInput
	return &cli.Command{
		Name:  "get",
		Usage: "Get deployment details",
		Flags: []cli.Flag{
			optionalString(flagDeploymentID, "Deployment ID", &in.DeploymentID),
			optionalString(flagApp, "App name to resolve (alternative to --deployment-id)", &in.App),
			optionalString(flagConfig, "Config name or path", &in.Config),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleGetDeployment(ctx, in)
		},
	}
}

func statusCommand() *cli.Command {
	var in StatusInput
	return &cli.Command{
		Name:  "status",
		Usage: "Check deployment status",
		Flags: []cli.Flag{
			optionalString(flagDeploymentID, "Deployment ID to check", &in.DeploymentID),
			optionalString(flagApp, "App name to resolve (alternative to --deployment-id)", &in.App),
			optionalString(flagConfig, "Config name or path", &in.Config),
			optionalBool(flagWatch, "Watch deployment status in real-time", &in.Watch),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleDeploymentStatus(ctx, in)
		},
	}
}

func destroyCommand() *cli.Command {
	var in DestroyInput
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"destroy", "rm"},
		Usage:     "Delete a deployment and all associated resources",
		ArgsUsage: "<deployment-id-or-name>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagDeploymentID,
				Usage:       "Deployment ID (alternative to positional arg)",
				Destination: &in.DeploymentID,
			},
			&cli.StringFlag{
				Name:        flagApp,
				Usage:       "App name (alternative to positional arg)",
				Destination: &in.App,
			},
			optionalBool(flagYes, "Skip confirmation prompt", &in.Yes),
			&cli.BoolFlag{
				Name:        "retain-volumes",
				Usage:       "Retain persistent volumes instead of deleting them",
				Destination: &in.RetainVolumes,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() >= 1 {
				arg := cmd.Args().First()
				if looksLikeUUID(arg) {
					in.DeploymentID = arg
				} else {
					in.App = arg
				}
			}
			return handleDestroyDeployment(ctx, in)
		},
	}
}

func restartCommand() *cli.Command {
	var in DeployRefInput
	return &cli.Command{
		Name:  "restart",
		Usage: "Trigger a rolling restart without redeploying",
		Flags: []cli.Flag{
			optionalString(flagDeploymentID, "Deployment ID to restart", &in.DeploymentID),
			optionalString(flagApp, "App name to resolve (alternative to --deployment-id)", &in.App),
			optionalString(flagConfig, "Config name or path", &in.Config),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleRestartDeployment(ctx, in)
		},
	}
}

func releasesCommand() *cli.Command {
	var in DeployRefInput
	return &cli.Command{
		Name:  "releases",
		Usage: "List release history for a deployment",
		Flags: []cli.Flag{
			optionalString(flagDeploymentID, "Deployment ID", &in.DeploymentID),
			optionalString(flagApp, "App name to resolve (alternative to --deployment-id)", &in.App),
			optionalString(flagConfig, "Config name or path", &in.Config),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleListReleases(ctx, in)
		},
	}
}

func rollbackCommand() *cli.Command {
	var in RollbackInput
	return &cli.Command{
		Name:  "rollback",
		Usage: "Roll back to a previous release",
		Flags: []cli.Flag{
			optionalString(flagDeploymentID, "Deployment ID", &in.DeploymentID),
			optionalString(flagApp, "App name to resolve (alternative to --deployment-id)", &in.App),
			optionalString(flagConfig, "Config name or path", &in.Config),
			optionalInt(flagVersion, "Version number to roll back to (default: previous version)", &in.Version),
			optionalBool(flagYes, "Skip confirmation prompt", &in.Yes),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleRollback(ctx, in)
		},
	}
}

func openCommand() *cli.Command {
	var in DeployRefInput
	return &cli.Command{
		Name:  "open",
		Usage: "Open a deployment's URL in the default browser",
		Flags: []cli.Flag{
			optionalString(flagDeploymentID, "Deployment ID", &in.DeploymentID),
			optionalString(flagApp, "App name to resolve (alternative to --deployment-id)", &in.App),
			optionalString(flagConfig, "Config name or path", &in.Config),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleOpenDeployment(ctx, in)
		},
	}
}

func scaleCommand() *cli.Command {
	var in ScaleInput
	return &cli.Command{
		Name:  "scale",
		Usage: "Set the replica count without redeploying",
		Flags: []cli.Flag{
			optionalString(flagDeploymentID, "Deployment ID", &in.DeploymentID),
			optionalString(flagApp, "App name to resolve (alternative to --deployment-id)", &in.App),
			optionalString(flagConfig, "Config name or path", &in.Config),
			&cli.IntFlag{
				Name:        flagReplicas,
				Usage:       "Target replica count",
				Required:    true,
				Destination: &in.Replicas,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return handleScaleDeployment(ctx, in)
		},
	}
}

// looksLikeUUID reports whether s looks like a standard UUID (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
// This is used to distinguish positional args that are deployment IDs from those
// that are app names.
func looksLikeUUID(s string) bool {
	return len(s) == 36 && strings.Count(s, "-") == 4
}
