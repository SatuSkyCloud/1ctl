package commands

import (
	"context"
	"1ctl/internal/api"
	"1ctl/internal/config"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/deploy"
	"1ctl/internal/utils"
	"1ctl/internal/validator"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func DeployCommand() *cli.Command {
	deployFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "Application name (default: auto-detected from satusky.toml or git remote)",
		},
		&cli.StringFlag{
			Name:  "cpu",
			Usage: "Legacy alias for --cpu-limit (e.g., '1', '500m')",
		},
		&cli.StringFlag{
			Name:  "cpu-request",
			Usage: "Guaranteed CPU reservation per replica (e.g., '250m')",
			Value: "250m",
		},
		&cli.StringFlag{
			Name:  "cpu-limit",
			Usage: "Maximum burst CPU per replica (e.g., '1')",
			Value: "1",
		},
		&cli.StringFlag{
			Name:  "memory",
			Usage: "Memory allocation (e.g., '512Mi', '2Gi')",
			Value: "256Mi",
		},
		&cli.StringSliceFlag{
			Name:  "machine",
			Usage: "Explicit machine name (BYOA). Repeatable for multi-machine.",
		},
		&cli.StringFlag{
			Name:  "machine-tag",
			Usage: "Deploy to your machines labelled with this tag (BYOA). Set the satusky.com/<tag> label via the labels API.",
		},
		&cli.StringFlag{
			Name:  "domain",
			Usage: "Custom domain (default: *.satusky.com)",
		},
		&cli.StringFlag{
			Name:  "organization",
			Usage: "Organization name to organize your resources (default: current organization)",
		},
		&cli.StringFlag{
			Name:  "health-path",
			Usage: "HTTP path to smoke test after deploy wait succeeds (default: tries /health then /)",
		},
		&cli.StringFlag{
			Name:  "dockerfile",
			Usage: "Dockerfile path for cloud build (default: Dockerfile)",
			Value: "Dockerfile",
		},
		&cli.StringFlag{
			Name:  "image",
			Usage: "Pre-built image reference — skips cloud build entirely (e.g. registry.satusky.com/satusky-container-registry/myapp:abc1234)",
		},
		&cli.BoolFlag{
			Name:  "fast",
			Usage: "Use the accelerated cloud build backend before deploying (ignored when --image is set)",
		},
		&cli.IntFlag{
			Name:  "port",
			Usage: "Application port (default: 8080)",
			Value: 8080,
		},
		&cli.StringSliceFlag{
			Name:  "env",
			Usage: "Environment variables (format: KEY=VALUE)",
		},
		&cli.StringFlag{
			Name:  "volume-size",
			Usage: "Storage size (e.g., '10Gi')",
		},
		&cli.StringFlag{
			Name:  "volume-mount",
			Usage: "Storage mount path",
		},
		&cli.StringFlag{
			Name:  "volume-storage-class",
			Usage: "Storage class for volumes (e.g., 'ceph-block')",
			Value: "ceph-block",
		},
		// Zone targeting flag
		&cli.StringFlag{
			Name:  "zone",
			Usage: "Target deployment zone (e.g., 'my-kul-1b', 'my-bki-1a'). Run '1ctl cluster zones' to list available zones.",
		},
		// Multi-cluster deployment flags
		&cli.BoolFlag{
			Name:  "multicluster",
			Usage: "Enable multi-cluster deployment across KL and BKI clusters",
		},
		&cli.StringFlag{
			Name:  "multicluster-mode",
			Usage: "Multi-cluster mode: 'active-active' or 'active-passive' (default: active-passive)",
			Value: "active-passive",
		},
		&cli.BoolFlag{
			Name:  "backup-enabled",
			Usage: "Enable backups (auto-enabled for active-passive, optional for active-active)",
			Value: true,
		},
		&cli.StringFlag{
			Name:  "backup-schedule",
			Usage: "Backup frequency: 'hourly', 'daily', 'weekly' (default: daily)",
			Value: "daily",
		},
		&cli.StringFlag{
			Name:  "backup-retention",
			Usage: "Backup retention: '24h', '72h', '168h', '720h' (default: 168h)",
			Value: "168h",
		},
		&cli.IntFlag{
			Name:  "backup-priority-cluster",
			Usage: "Which cluster performs backups: 1 (Primary/KL) or 2 (Secondary/BKI) (default: 1)",
			Value: 1,
		},
		// HA settings flags
		&cli.IntFlag{
			Name:  "replicas",
			Usage: "Manual replica count override (default: auto from machine count)",
		},
		// PDB flags
		&cli.BoolFlag{
			Name:  "pdb",
			Usage: "Enable PodDisruptionBudget (auto-enabled when replicas > 1)",
		},
		&cli.StringFlag{
			Name:  "pdb-type",
			Usage: "PDB type: 'auto', 'fixed', 'percent' (default: auto)",
			Value: "auto",
		},
		&cli.IntFlag{
			Name:  "pdb-min-available",
			Usage: "Minimum available pods for PDB type=fixed",
		},
		&cli.IntFlag{
			Name:  "pdb-percent",
			Usage: "Minimum available percentage for PDB type=percent (1-100)",
		},
		// HPA flags
		&cli.BoolFlag{
			Name:  "hpa",
			Usage: "Enable HorizontalPodAutoscaler",
		},
		&cli.IntFlag{
			Name:  "hpa-min-replicas",
			Usage: "HPA minimum replicas (default: 1)",
			Value: 1,
		},
		&cli.IntFlag{
			Name:  "hpa-max-replicas",
			Usage: "HPA maximum replicas (default: 10)",
			Value: 10,
		},
		&cli.IntFlag{
			Name:  "hpa-cpu-target",
			Usage: "HPA target CPU utilization percentage (default: 80)",
			Value: 80,
		},
		&cli.IntFlag{
			Name:  "hpa-memory-target",
			Usage: "HPA target memory utilization percentage (0 = disabled)",
			Value: 0,
		},
		// VPA flags
		&cli.BoolFlag{
			Name:  "vpa",
			Usage: "Enable VerticalPodAutoscaler",
		},
		&cli.StringFlag{
			Name:  "vpa-mode",
			Usage: "VPA update mode: 'Off', 'Initial', 'Auto' (default: Off)",
			Value: "Off",
		},
		&cli.StringFlag{
			Name:  "vpa-min-cpu",
			Usage: "VPA minimum CPU (e.g., '100m')",
		},
		&cli.StringFlag{
			Name:  "vpa-max-cpu",
			Usage: "VPA maximum CPU (e.g., '4')",
		},
		&cli.StringFlag{
			Name:  "vpa-min-memory",
			Usage: "VPA minimum memory (e.g., '128Mi')",
		},
		&cli.StringFlag{
			Name:  "vpa-max-memory",
			Usage: "VPA maximum memory (e.g., '8Gi')",
		},
		// Dependency readiness flags
		&cli.StringSliceFlag{
			Name:  "wait-for",
			Usage: "Wait for a TCP dependency before starting (format: host:port, e.g. postgres:5432). Can be repeated for multiple dependencies.",
		},
		// Deployment strategy flags
		&cli.StringFlag{
			Name:  "strategy",
			Usage: "Deployment rollout strategy: rolling (default), recreate",
			Value: "rolling",
		},
		&cli.StringFlag{
			Name:  "rolling-max-surge",
			Usage: "Rolling update max surge. Pods or percentage, e.g. '1' or '25%'",
			Value: "25%",
		},
		&cli.StringFlag{
			Name:  "rolling-max-unavailable",
			Usage: "Rolling update max unavailable. Pods or percentage, e.g. '0' or '25%'",
			Value: "25%",
		},
		&cli.StringFlag{
			Name:  "config",
			Usage: "Config name or path (e.g. staging, satusky.staging.toml). Default: satusky.toml",
		},
		&cli.BoolFlag{
			Name:    "wait",
			Aliases: []string{"w"},
			Usage:   "Wait until pods are Running before returning (default timeout: 5m)",
		},
	}

	return &cli.Command{
		Name:  "deploy",
		Usage: "Deploy your application to SatuSky Cloud",
		Description: `Build and deploy your application to SatuSky Cloud.

Images are built in the cloud — no local Docker installation required.
Your source directory is packaged and sent to the build service, which builds
and pushes the image directly to the registry.

   1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 512Mi --port 8080
   1ctl deploy --cpu-request 500m --cpu-limit 1 --memory 1Gi --port 3000 --env KEY=VALUE
   1ctl deploy --image registry.satusky.com/.../myapp:tag   # skip cloud build

Config file (satusky.toml) can persist deploy settings across runs.
Run '1ctl init' to create one.

Subcommands manage existing deployments:
   1ctl deploy list
   1ctl deploy status --deployment-id <id>   [or --app <name>]
   1ctl deploy restart --deployment-id <id>  [or --app <name>]
   1ctl deploy releases --deployment-id <id> [or --app <name>]
   1ctl deploy rollback --deployment-id <id> [or --app <name>]
   1ctl deploy destroy --deployment-id <id>  [or --app <name>]`,
		Flags: deployFlags,
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List deployments",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "namespace",
						Usage: "Filter by namespace",
					},
				},
				Action: handleListDeployments,
			},
			{
				Name:  "get",
				Usage: "Get deployment details",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "deployment-id",
						Usage: "Deployment ID to get details for",
					},
					&cli.StringFlag{
						Name:  "app",
						Usage: "App name to resolve (alternative to --deployment-id)",
					},
					&cli.StringFlag{
						Name:  "config",
						Usage: "Config name or path (e.g. staging, satusky.staging.toml). Default: satusky.toml",
					},
				},
				Action: handleGetDeployment,
			},
			{
				Name:  "status",
				Usage: "Check deployment status",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "deployment-id",
						Usage: "Deployment ID to check",
					},
					&cli.StringFlag{
						Name:  "app",
						Usage: "App name to resolve (alternative to --deployment-id)",
					},
					&cli.StringFlag{
						Name:  "config",
						Usage: "Config name or path (e.g. staging, satusky.staging.toml). Default: satusky.toml",
					},
					&cli.BoolFlag{
						Name:  "watch",
						Usage: "Watch deployment status in real-time",
					},
				},
				Action: handleDeploymentStatus,
			},
			{
				Name:  "destroy",
				Usage: "Delete a deployment and all associated resources",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "deployment-id",
						Usage: "Deployment ID to destroy",
					},
					&cli.StringFlag{
						Name:  "app",
						Usage: "App name to resolve (alternative to --deployment-id)",
					},
					&cli.StringFlag{
						Name:  "config",
						Usage: "Config name or path (e.g. staging, satusky.staging.toml). Default: satusky.toml",
					},
					&cli.BoolFlag{
						Name:    "yes",
						Aliases: []string{"y"},
						Usage:   "Skip confirmation prompt",
					},
				},
				Action: handleDestroyDeployment,
			},
			{
				Name:  "restart",
				Usage: "Trigger a rolling restart without redeploying",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "deployment-id",
						Usage: "Deployment ID to restart",
					},
					&cli.StringFlag{
						Name:  "app",
						Usage: "App name to resolve (alternative to --deployment-id)",
					},
					&cli.StringFlag{
						Name:  "config",
						Usage: "Config name or path. Default: satusky.toml",
					},
				},
				Action: handleRestartDeployment,
			},
			{
				Name:  "releases",
				Usage: "List release history for a deployment",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "deployment-id", Usage: "Deployment ID"},
					&cli.StringFlag{Name: "app", Usage: "App name to resolve (alternative to --deployment-id)"},
					&cli.StringFlag{Name: "config", Usage: "Config name or path. Default: satusky.toml"},
				},
				Action: handleListReleases,
			},
			{
				Name:  "rollback",
				Usage: "Roll back to a previous release",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "deployment-id", Usage: "Deployment ID"},
					&cli.StringFlag{Name: "app", Usage: "App name to resolve (alternative to --deployment-id)"},
					&cli.StringFlag{Name: "config", Usage: "Config name or path. Default: satusky.toml"},
					&cli.IntFlag{Name: "version", Usage: "Version number to roll back to (default: previous version)"},
					&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"},
				},
				Action: handleRollback,
			},
			{
				Name:  "open",
				Usage: "Open a deployment's URL in the default browser",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "deployment-id", Usage: "Deployment ID"},
					&cli.StringFlag{Name: "app", Usage: "App name to resolve (alternative to --deployment-id)"},
					&cli.StringFlag{Name: "config", Usage: "Config name or path. Default: satusky.toml"},
				},
				Action: handleOpenDeployment,
			},
			{
				Name:  "scale",
				Usage: "Set the replica count for a deployment without redeploying",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "deployment-id", Usage: "Deployment ID"},
					&cli.StringFlag{Name: "app", Usage: "App name to resolve (alternative to --deployment-id)"},
					&cli.StringFlag{Name: "config", Usage: "Config name or path. Default: satusky.toml"},
					&cli.IntFlag{Name: "replicas", Usage: "Target replica count", Required: true},
				},
				Action: handleScaleDeployment,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// If subcommand is provided, let it handle
			if cmd.NArg() > 0 {
				return cli.ShowSubcommandHelp(cmd)
			}
			return handleDeploy(ctx, cmd)
		},
	}
}

func handleDeploy(ctx context.Context, cmd *cli.Command) error {
	// Check token expiry before any work begins to fail fast with a clear message
	if err := satuskyctx.CheckTokenExpiry(); err != nil {
		return err
	}

	// Load satusky.toml once and use it for both the help-guard and the merge.
	// Previously this file was parsed three separate times per deploy.
	cfg, err := config.FindConfig(cmd.String("config"))
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to load config: %s", err.Error()), nil)
	}

	// Show help only when no deployable resource defaults are available.
	// `cpu` and `memory` have flag defaults, so a basic Dockerfile deploy
	// should not require users to repeat those values in satusky.toml.
	if shouldShowDeployHelp(cmd, cfg) {
		return cli.ShowSubcommandHelp(cmd)
	}

	// Snapshot user-typed flags BEFORE the toml merge. applyConfigScalar uses
	// cli.Set, which flips cmd.IsSet(...) to true — so any downstream check that
	// needs "did the *user* set this" must use this snapshot, not c.IsSet.
	// (Discovered during review: RollingFlagsExplicit was tripping for
	// toml-provided defaults, forcing strategy config onto requests that
	// would otherwise have been omitted.)
	userSet := captureUserSetFlags(cmd, "rolling-max-surge", "rolling-max-unavailable", "domain", "health-path", "multicluster", "cpu", "cpu-request", "cpu-limit")

	// Apply satusky.toml scalar fields to flags the user didn't explicitly set.
	// Precedence overall: CLI flag (c.IsSet) > satusky.toml > flag Value: default.
	// Structured sections ([volume], [hpa], [vpa], [pdb], [multicluster]) are
	// merged in prepareDeploymentOptions where they map to nested option structs.
	if cfg != nil {
		if cfg.App.CPURequest != "" && !userSet["cpu-request"] {
			if err := applyConfigScalar(cmd, "cpu-request", cfg.App.CPURequest); err != nil {
				return err
			}
		}
		if cfg.App.CPULimit != "" && !userSet["cpu"] && !userSet["cpu-limit"] {
			if err := applyConfigScalar(cmd, "cpu-limit", cfg.App.CPULimit); err != nil {
				return err
			}
		} else if cfg.App.CPU != "" && !userSet["cpu"] && !userSet["cpu-limit"] {
			if err := applyConfigScalar(cmd, "cpu-limit", cfg.App.CPU); err != nil {
				return err
			}
		}
		if err := applyConfigScalar(cmd, "memory", cfg.App.Memory); err != nil {
			return err
		}
		if cfg.App.Port != 0 {
			if err := applyConfigScalar(cmd, "port", fmt.Sprintf("%d", cfg.App.Port)); err != nil {
				return err
			}
		}
		if err := applyConfigScalar(cmd, "domain", cfg.App.Domain); err != nil {
			return err
		}
		if err := applyConfigScalar(cmd, "dockerfile", cfg.Build.Dockerfile); err != nil {
			return err
		}
		if cfg.App.Replicas > 0 {
			if err := applyConfigScalar(cmd, "replicas", fmt.Sprintf("%d", cfg.App.Replicas)); err != nil {
				return err
			}
		}
		if err := applyConfigScalar(cmd, "zone", cfg.App.Zone); err != nil {
			return err
		}
		if err := applyConfigScalar(cmd, "organization", cfg.App.Organization); err != nil {
			return err
		}
		if err := applyConfigScalar(cmd, "health-path", cfg.Checks.HealthPath); err != nil {
			return err
		}
		if err := applyConfigScalar(cmd, "strategy", cfg.Deploy.Strategy); err != nil {
			return err
		}
		if err := applyConfigScalar(cmd, "rolling-max-surge", cfg.Deploy.RollingMaxSurge); err != nil {
			return err
		}
		if err := applyConfigScalar(cmd, "rolling-max-unavailable", cfg.Deploy.RollingMaxUnavailable); err != nil {
			return err
		}
		// Volume scalars wire through the existing --volume-size / --volume-mount
		// flags so validateInputs / prepareDeploymentOptions see them uniformly.
		if err := applyConfigScalar(cmd, "volume-size", cfg.Volume.Size); err != nil {
			return err
		}
		if err := applyConfigScalar(cmd, "volume-mount", cfg.Volume.Mount); err != nil {
			return err
		}
		// wait-for is a StringSliceFlag, merged below.
		if len(cfg.Deploy.WaitFor) > 0 && !cmd.IsSet("wait-for") {
			for _, v := range cfg.Deploy.WaitFor {
				if err := cmd.Set("wait-for", v); err != nil {
					return utils.NewError(fmt.Sprintf("failed to set wait-for from config: %s", err.Error()), nil)
				}
			}
		}
	}

	// Validate inputs first
	if err := validateInputs(cmd); err != nil {
		return utils.NewError(fmt.Sprintf("validation failed: %s", err.Error()), nil)
	}

	// Prepare deployment options
	opts, err := prepareDeploymentOptions(cmd, cfg, userSet)
	if err != nil {
		return utils.NewError(fmt.Sprintf("deployment preparation failed: %s", err.Error()), nil)
	}

	// Execute deployment
	resp, err := deploy.Deploy(opts)
	if err != nil {
		// Check if this is a resource exhausted error - already formatted, just return it
		if _, ok := err.(*utils.ResourceExhaustedCLIError); ok {
			return err
		}
		return utils.NewError(fmt.Sprintf("deployment failed: %s", err.Error()), nil)
	}

	publicURL := checkPublicURLReadiness(resp)
	return reportDeployResult(resp, opts.Wait, publicURL, opts.SmokePath, opts.StrictSmoke)
}

func reportDeployResult(resp *api.CreateDeploymentResponse, waitForWorkload bool, publicURL publicURLReadiness, smokePath string, strictSmoke bool) error {
	if publicURL.Ready {
		utils.PrintSuccess("🚀 Deployment for %s is successful! Your app is live at: https://%s", resp.AppLabel, resp.Domain)
	} else {
		utils.PrintSuccess("Deployment for %s was accepted by the platform.", resp.AppLabel)
		utils.PrintWarning("Public URL is not ready yet: https://%s", resp.Domain)
		if publicURL.Reason != "" {
			utils.PrintStatusLine("Public URL reason", publicURL.Reason)
		}
		utils.PrintInfo("Run: 1ctl domains check %s --probe", resp.Domain)
	}
	utils.PrintStatusLine("Deployment ID", resp.DeploymentID.String())

	workloadReady := false
	if waitForWorkload {
		utils.PrintInfo("Waiting for deployment to become healthy...")
		status, err := api.WaitForDeployment(resp.DeploymentID.String(), 5*time.Minute)
		if err != nil {
			utils.PrintWarning("Timed out waiting for deployment: %s", err.Error())
		} else if status != nil && status.Status == "Running" {
			workloadReady = true
			utils.PrintSuccess("Deployment is healthy: pods Running")
		}
		if resp.Domain != "" {
			if publicURL.Ready {
				utils.PrintStatusLine("Public URL", fmt.Sprintf("ready: https://%s", resp.Domain))
			} else {
				utils.PrintStatusLine("Public URL", fmt.Sprintf("not ready: https://%s", resp.Domain))
				if publicURL.Reason != "" {
					utils.PrintStatusLine("Reason", publicURL.Reason)
				}
			}
		}
		if workloadReady && !publicURL.Ready && resp.Domain != "" {
			return utils.NewError(fmt.Sprintf("deployment workload is healthy, but public URL is not ready yet. Run: 1ctl domains check %s --probe", resp.Domain), nil)
		}
		if workloadReady && publicURL.Ready && resp.Domain != "" {
			smokeURL := "https://" + resp.Domain
			smokePaths := smokePathCandidates(smokePath)
			utils.PrintInfo("Running public URL smoke check against %s", smokeURL)
			smoke := checkPublicURLSmoke(smokeURL, smokePaths, strictSmoke)
			if smoke.Ready {
				if smoke.Path != "" {
					utils.PrintSuccess("Public URL smoke check passed: %s%s", smokeURL, smoke.Path)
				} else {
					utils.PrintSuccess("Public URL smoke check passed: %s", smokeURL)
				}
			} else {
				utils.PrintWarning("Public URL smoke check failed: %s", smoke.Reason)
				if strictSmoke {
					return utils.NewError(fmt.Sprintf("deployment workload is healthy, but the public URL smoke check failed for %s: %s", smokeURL, smoke.Reason), nil)
				}
			}
		}
	}

	return nil
}

func shouldShowDeployHelp(cmd *cli.Command, cfg *config.ProjectConfig) bool {
	if cmd.String("image") != "" {
		return false
	}
	if cmd.String("cpu-request") != "" && cmd.String("memory") != "" {
		return false
	}
	return cfg == nil || (cfg.App.CPU == "" && cfg.App.CPURequest == "" && cfg.App.Memory == "")
}

// resolveDockerfilePath returns the Dockerfile path actually used by the
// deploy, falling back from the --dockerfile flag value to FindDockerfile's
// common-location search. Empty result means a pre-built --image was given
// and no Dockerfile is required.
func resolveDockerfilePath(cmd *cli.Command) (string, error) {
	if cmd.IsSet("image") {
		return "", nil
	}
	dockerfilePath := cmd.String("dockerfile")
	if err := validator.ValidateDockerfile(dockerfilePath); err == nil {
		return dockerfilePath, nil
	}
	found, err := validator.FindDockerfile(".")
	if err != nil {
		return "", utils.NewError("no valid Dockerfile found: please ensure a Dockerfile exists in your project", err)
	}
	return found, nil
}

// validateInputs validates flag-driven inputs in place. It does NOT mutate
// cli.Context — Dockerfile resolution is the caller's responsibility via
// resolveDockerfilePath.
func validateInputs(cmd *cli.Command) error {
	// Validate Dockerfile only when not using a pre-built image
	if _, err := resolveDockerfilePath(cmd); err != nil {
		return err
	}

	// Validate CPU and Memory
	if cmd.String("cpu") != "" {
		if err := validator.ValidateCPU(cmd.String("cpu")); err != nil {
			return utils.NewError("invalid CPU value: %v", err)
		}
	}
	if err := validator.ValidateCPU(cmd.String("cpu-request")); err != nil {
		return utils.NewError("invalid CPU request value: %v", err)
	}
	if err := validator.ValidateCPU(cmd.String("cpu-limit")); err != nil {
		return utils.NewError("invalid CPU limit value: %v", err)
	}
	if err := validator.ValidateMemory(cmd.String("memory")); err != nil {
		return utils.NewError("invalid memory value: %v", err)
	}
	if err := validator.ValidateDomain(cmd.String("domain")); err != nil {
		return utils.NewError("invalid domain: %v", err)
	}
	if err := validator.ValidateURLPath(cmd.String("health-path")); err != nil {
		return utils.NewError("invalid health path: %v", err)
	}

	// Validate volume options
	if cmd.IsSet("volume-size") || cmd.IsSet("volume-mount") {
		if cmd.String("volume-size") == "" {
			return utils.NewError("volume-size is required when volume is enabled", nil)
		}
		if cmd.String("volume-mount") == "" {
			return utils.NewError("volume-mount is required when volume is enabled", nil)
		}
		if err := validator.ValidateMemory(cmd.String("volume-size")); err != nil {
			return utils.NewError("invalid volume size: %v", err)
		}
	}

	// Validate HA settings
	if cmd.IsSet("hpa") && cmd.IsSet("vpa") && cmd.Bool("hpa") && cmd.Bool("vpa") {
		vpaMode := cmd.String("vpa-mode")
		if vpaMode == "Auto" {
			return utils.NewError("HPA and VPA with mode 'Auto' cannot be used together - they both try to scale resources", nil)
		}
	}
	if cmd.IsSet("hpa-min-replicas") && cmd.IsSet("hpa-max-replicas") {
		if cmd.Int("hpa-min-replicas") > cmd.Int("hpa-max-replicas") {
			return utils.NewError("hpa-min-replicas cannot be greater than hpa-max-replicas", nil)
		}
	}
	if cmd.IsSet("pdb-percent") {
		if cmd.Int("pdb-percent") < 1 || cmd.Int("pdb-percent") > 100 {
			return utils.NewError("pdb-percent must be between 1 and 100", nil)
		}
	}

	// Reject --multicluster combined with a custom domain. The platform's
	// satusky-operator silently blocks replication of zone-specific ingress
	// classes (the class used by custom-domain ingresses), so the user would
	// get a deployment that "looks" multi-cluster but only routes traffic on
	// KUL via the public LoadBalancer. Backend enforces the same rule; this
	// client-side check just gives a friendlier error before the round trip.
	// See backend .devs/docs/MULTICLUSTER_CONSTRAINTS.md.
	// Validator checks the resolved domain value, not c.IsSet — so toml-only
	// domains are validated too. (Pre-fix this was implicitly skipped when
	// the domain came from satusky.toml.)
	if cmd.Bool("multicluster") {
		domain := strings.TrimSpace(strings.ToLower(cmd.String("domain")))
		domain = strings.TrimPrefix(domain, "*.")
		if domain != "" && domain != "satusky.com" && !strings.HasSuffix(domain, ".satusky.com") {
			return utils.NewError(fmt.Sprintf(
				"--multicluster is not supported with custom domains yet: %q is not a *.satusky.com hostname. "+
					"Use a *.satusky.com hostname or drop --multicluster.", cmd.String("domain")), nil)
		}
	}

	// Validate --wait-for entries
	for _, v := range cmd.StringSlice("wait-for") {
		if _, _, err := validator.ValidateWaitFor(v); err != nil {
			return err
		}
	}

	return nil
}

func prepareDeploymentOptions(cmd *cli.Command, cfg *config.ProjectConfig, userSet map[string]bool) (deploy.DeploymentOptions, error) {
	dockerfilePath, err := resolveDockerfilePath(cmd)
	if err != nil {
		return deploy.DeploymentOptions{}, err
	}
	opts := deploy.DeploymentOptions{
		CPU:               cmd.String("cpu"),
		CPURequest:        cmd.String("cpu-request"),
		CPULimit:          cmd.String("cpu-limit"),
		Memory:            cmd.String("memory"),
		Domain:            cmd.String("domain"),
		SmokePath:         cmd.String("health-path"),
		StrictSmoke:       cmd.IsSet("health-path"),
		Port:              cmd.Int("port"),
		DockerfilePath:    dockerfilePath,
		PrebuiltImage:     cmd.String("image"),
		FastBuild:         cmd.Bool("fast"),
	}

	if !cmd.IsSet("fast") && cfg != nil && cfg.Build.FastBuild {
		opts.FastBuild = true
	}
	if userSet["cpu"] && !userSet["cpu-limit"] {
		opts.CPULimit = cmd.String("cpu")
	}

	// App name precedence: --name flag > satusky.toml > git remote auto-detect.
	switch {
	case cmd.String("name") != "":
		opts.Name = cmd.String("name")
	case cfg != nil && cfg.App.Name != "":
		opts.Name = cfg.App.Name
	}

	// Organization precedence: --organization flag > current context namespace.
	if cmd.IsSet("organization") {
		opts.Organization = cmd.String("organization")
	} else {
		opts.Organization = satuskyctx.GetCurrentNamespace()
	}

	// Handle environment variables if enabled when --env are set
	if cmd.IsSet("env") {
		opts.EnvEnabled = true
		env := &api.Environment{
			KeyValues: parseEnvVars(cmd.StringSlice("env")),
		}
		opts.Environment = env
	}

	// Handle hostnames if enabled when --machine is set
	if cmd.IsSet("machine") {
		machineNames := cmd.StringSlice("machine")
		hostnameSet := make(map[string]bool) // Add deduplication for manually specified machines
		for _, machineName := range machineNames {
			machine, err := api.GetMachineByName(machineName)
			if err != nil {
				return deploy.DeploymentOptions{}, utils.NewError(fmt.Sprintf("failed to get machine by name: %s", err.Error()), nil)
			}

			// check if machine is owned by the current user (monetized machines can be used by anyone)
			if !machine.Monetized && machine.OwnerID.String() != satuskyctx.GetUserID() {
				return deploy.DeploymentOptions{}, utils.NewError(fmt.Sprintf("machine %s is not owned by you", machineName), nil)
			}

			// Only add hostname if we haven't seen it before (using machine ID instead of machine name)
			if !hostnameSet[machine.MachineID] {
				hostnameSet[machine.MachineID] = true
				opts.Hostnames = append(opts.Hostnames, machine.MachineID)
			}
		}
	}

	// Handle volume if enabled when --volume-size and --volume-mount are set
	if cmd.IsSet("volume-size") || cmd.IsSet("volume-mount") {
		opts.VolumeEnabled = true
		storageClass := cmd.String("volume-storage-class")
		if storageClass == "" {
			storageClass = "ceph-block" // default to the cluster default
		}
		vol := &api.Volume{
			StorageSize:  cmd.String("volume-size"),
			MountPath:    cmd.String("volume-mount"),
			StorageClass: storageClass,
		}
		opts.Volume = vol
	}

	// Handle zone targeting. The legacy "also set Region to the zone value"
	// fallback is gone (issue #24) — the backend now reads Zone directly.
	if cmd.IsSet("zone") {
		opts.Zone = cmd.String("zone")
	}

	// Handle --machine-tag: BYOA targeting by label. Resolves to a list of
	// owned, online machine IDs matching satusky.com/<tag>. Precedence:
	// --machine-tag flag > satusky.toml [deploy].machine_tag.
	tag := cmd.String("machine-tag")
	if tag == "" && cfg != nil {
		tag = cfg.Deploy.MachineTag
	}
	if tag != "" && !cmd.IsSet("machine") {
		hostnames, err := resolveMachineTag(tag)
		if err != nil {
			return deploy.DeploymentOptions{}, err
		}
		opts.Hostnames = hostnames
		utils.PrintInfo("Resolved --machine-tag %q to %d owned machine(s)", tag, len(hostnames))
	}

	// Handle multicluster configuration
	if cmd.Bool("multicluster") {
		opts.MulticlusterEnabled = true
		opts.MulticlusterMode = cmd.String("multicluster-mode")
		opts.BackupEnabled = cmd.Bool("backup-enabled")
		opts.BackupSchedule = cmd.String("backup-schedule")
		opts.BackupRetention = cmd.String("backup-retention")
		opts.BackupPriorityCluster = cmd.Int("backup-priority-cluster")
	}

	// Handle replica count override
	if cmd.IsSet("replicas") {
		opts.Replicas = cmd.Int("replicas")
	}

	// Handle PDB configuration
	if cmd.IsSet("pdb") && cmd.Bool("pdb") {
		pdbConfig := &deploy.PDBConfig{
			Enabled: true,
			Type:    deploy.PDBConfigType(cmd.String("pdb-type")),
		}
		if cmd.IsSet("pdb-min-available") {
			val, err := api.SafeInt32(cmd.Int("pdb-min-available"))
			if err != nil {
				return deploy.DeploymentOptions{}, utils.NewError("invalid pdb-min-available value", err)
			}
			pdbConfig.MinAvailable = &val
		}
		if cmd.IsSet("pdb-percent") {
			val, err := api.SafeInt32(cmd.Int("pdb-percent"))
			if err != nil {
				return deploy.DeploymentOptions{}, utils.NewError("invalid pdb-percent value", err)
			}
			pdbConfig.Percent = &val
		}
		opts.PDBConfig = pdbConfig
	}

	// Handle HPA configuration
	if cmd.IsSet("hpa") && cmd.Bool("hpa") {
		cpuTarget, err := api.SafeInt32(cmd.Int("hpa-cpu-target"))
		if err != nil {
			return deploy.DeploymentOptions{}, utils.NewError("invalid hpa-cpu-target value", err)
		}
		minReplicas, err := api.SafeInt32(cmd.Int("hpa-min-replicas"))
		if err != nil {
			return deploy.DeploymentOptions{}, utils.NewError("invalid hpa-min-replicas value", err)
		}
		maxReplicas, err := api.SafeInt32(cmd.Int("hpa-max-replicas"))
		if err != nil {
			return deploy.DeploymentOptions{}, utils.NewError("invalid hpa-max-replicas value", err)
		}
		hpaConfig := &api.HPAConfig{
			Enabled:     true,
			MinReplicas: minReplicas,
			MaxReplicas: maxReplicas,
			CPUTarget:   &cpuTarget,
		}
		if cmd.IsSet("hpa-memory-target") && cmd.Int("hpa-memory-target") > 0 {
			memTarget, err := api.SafeInt32(cmd.Int("hpa-memory-target"))
			if err != nil {
				return deploy.DeploymentOptions{}, utils.NewError("invalid hpa-memory-target value", err)
			}
			hpaConfig.MemoryTarget = &memTarget
		}
		opts.HPAConfig = hpaConfig
	}

	// Handle VPA configuration
	if cmd.IsSet("vpa") && cmd.Bool("vpa") {
		opts.VPAConfig = &api.VPAConfig{
			Enabled:    true,
			UpdateMode: cmd.String("vpa-mode"),
			MinCPU:     cmd.String("vpa-min-cpu"),
			MaxCPU:     cmd.String("vpa-max-cpu"),
			MinMemory:  cmd.String("vpa-min-memory"),
			MaxMemory:  cmd.String("vpa-max-memory"),
		}
	}

	// Handle --wait-for dependencies
	for _, v := range cmd.StringSlice("wait-for") {
		host, port, err := validator.ValidateWaitFor(v)
		if err != nil {
			return deploy.DeploymentOptions{}, err
		}
		opts.WaitFor = append(opts.WaitFor, api.WaitFor{Host: host, Port: port})
	}

	// Handle deployment strategy options
	opts.Strategy = cmd.String("strategy")
	opts.RollingMaxSurge = cmd.String("rolling-max-surge")
	opts.RollingMaxUnavailable = cmd.String("rolling-max-unavailable")
	// Record explicit-user-set so buildStrategyConfig doesn't drop a value
	// the user typed deliberately (e.g. --rolling-max-surge=25% to assert
	// the default in an audit log). Snapshot taken BEFORE the toml merge.
	opts.RollingFlagsExplicit = userSet["rolling-max-surge"] || userSet["rolling-max-unavailable"]
	opts.Wait = cmd.Bool("wait")

	// Validate strategy value
	switch opts.Strategy {
	case "rolling", "recreate":
		// valid
	default:
		return deploy.DeploymentOptions{}, utils.NewError(fmt.Sprintf("invalid --strategy %q: must be 'rolling' or 'recreate'", opts.Strategy), nil)
	}

	// Fall back to satusky.toml structured sections when the user didn't
	// pass the corresponding CLI flag. CLI-set values win; nothing is
	// overwritten here.
	if cfg != nil {
		applyConfigHPA(&opts, cmd, cfg.HPA)
		applyConfigVPA(&opts, cmd, cfg.VPA)
		applyConfigPDB(&opts, cmd, cfg.PDB)
		applyConfigMulticluster(&opts, cmd, cfg.Multicluster)
	}

	return opts, nil
}

// resolveMachineTag fetches the current user's owned machines, then filters
// them client-side to those tagged with satusky.com/<tag>. Returns the list
// of online machine IDs to send as hostnames. Errors clearly if no machines
// are online or tagged.
//
// Implementation note: this does N+1 round trips (one to list, one per
// machine for labels). Acceptable for the small-N owner-machine case;
// migrate to a server-side filter endpoint if the cost becomes meaningful.
func resolveMachineTag(tag string) ([]string, error) {
	userID := satuskyctx.GetUserID()
	if userID == "" {
		return nil, utils.NewError("not authenticated — run '1ctl auth login' first", nil)
	}
	userUUID, err := api.ParseUUID(userID)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("invalid user ID in context: %s", err.Error()), nil)
	}
	machines, err := api.GetMachinesByOwnerID(userUUID)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to list owned machines: %s", err.Error()), nil)
	}
	if len(machines) == 0 {
		return nil, utils.NewError("no owned machines found — register a machine before using --machine-tag", nil)
	}

	var hostnames []string
	for _, m := range machines {
		if m.Status != "online" {
			continue
		}
		labels, err := api.GetMachineLabels(m.MachineID)
		if err != nil {
			utils.PrintWarning("Could not read labels for machine %s: %s", m.MachineID, err.Error())
			continue
		}
		if api.MachineHasTag(labels, tag) {
			hostnames = append(hostnames, m.MachineID)
		}
	}
	if len(hostnames) == 0 {
		return nil, utils.NewError(fmt.Sprintf("no machines tagged %q are online — apply the satusky.com/%s label to at least one machine first", tag, tag), nil)
	}
	return hostnames, nil
}

// captureUserSetFlags records which of the named flags the user explicitly
// passed BEFORE any toml merge runs. applyConfigScalar later calls c.Set
// for toml-provided values, which would make c.IsSet return true and hide
// the user-vs-toml distinction from downstream code.
func captureUserSetFlags(cmd *cli.Command, names ...string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = cmd.IsSet(n)
	}
	return out
}

// applyConfigScalar sets a CLI flag from a non-empty satusky.toml value when
// the user didn't explicitly pass the flag. Keeps the legacy c.Set merge model
// while the broader prepareDeploymentOptions refactor lands.
func applyConfigScalar(cmd *cli.Command, flagName, cfgValue string) error {
	if cfgValue == "" {
		return nil
	}
	if cmd.IsSet(flagName) {
		return nil
	}
	if err := cmd.Set(flagName, cfgValue); err != nil {
		return utils.NewError(fmt.Sprintf("failed to set %s from config: %s", flagName, err.Error()), nil)
	}
	return nil
}

// applyConfigHPA merges [hpa] section into deploy options when --hpa wasn't
// set on the CLI. Flag-set HPA wins entirely; we don't merge piecewise to
// avoid surprising fallback values.
func applyConfigHPA(opts *deploy.DeploymentOptions, cmd *cli.Command, hpa config.HPAConfig) {
	if cmd.IsSet("hpa") || !hpa.Enabled {
		return
	}
	cfg := &api.HPAConfig{
		Enabled:     true,
		MinReplicas: defaultInt32(hpa.MinReplicas, 1),
		MaxReplicas: defaultInt32(hpa.MaxReplicas, 10),
	}
	cpu := defaultInt32(hpa.CPUTarget, 80)
	cfg.CPUTarget = &cpu
	if hpa.MemoryTarget > 0 {
		mem := hpa.MemoryTarget
		cfg.MemoryTarget = &mem
	}
	opts.HPAConfig = cfg
}

// applyConfigVPA merges [vpa] section into deploy options when --vpa wasn't set.
func applyConfigVPA(opts *deploy.DeploymentOptions, cmd *cli.Command, vpa config.VPAConfig) {
	if cmd.IsSet("vpa") || !vpa.Enabled {
		return
	}
	mode := vpa.Mode
	if mode == "" {
		mode = "Off"
	}
	opts.VPAConfig = &api.VPAConfig{
		Enabled:    true,
		UpdateMode: mode,
		MinCPU:     vpa.MinCPU,
		MaxCPU:     vpa.MaxCPU,
		MinMemory:  vpa.MinMemory,
		MaxMemory:  vpa.MaxMemory,
	}
}

// applyConfigPDB merges [pdb] section into deploy options when --pdb wasn't set.
func applyConfigPDB(opts *deploy.DeploymentOptions, cmd *cli.Command, pdb config.PDBConfig) {
	if cmd.IsSet("pdb") || !pdb.Enabled {
		return
	}
	typ := pdb.Type
	if typ == "" {
		typ = "auto"
	}
	cfg := &deploy.PDBConfig{Enabled: true, Type: deploy.PDBConfigType(typ)}
	if pdb.MinAvailable > 0 {
		v := pdb.MinAvailable
		cfg.MinAvailable = &v
	}
	if pdb.Percent > 0 {
		v := pdb.Percent
		cfg.Percent = &v
	}
	opts.PDBConfig = cfg
}

// applyConfigMulticluster merges [multicluster] section into deploy options
// when --multicluster wasn't set.
func applyConfigMulticluster(opts *deploy.DeploymentOptions, cmd *cli.Command, mc config.MulticlusterConfig) {
	if cmd.IsSet("multicluster") || !mc.Enabled {
		return
	}
	opts.MulticlusterEnabled = true
	opts.MulticlusterMode = mc.Mode
	if opts.MulticlusterMode == "" {
		opts.MulticlusterMode = "active-passive"
	}
	opts.BackupEnabled = mc.BackupEnabled
	opts.BackupSchedule = mc.BackupSchedule
	opts.BackupRetention = mc.BackupRetention
	opts.BackupPriorityCluster = mc.BackupPriorityCluster
	if opts.BackupPriorityCluster == 0 {
		opts.BackupPriorityCluster = 1
	}
}

func defaultInt32(v, fallback int32) int32 {
	if v == 0 {
		return fallback
	}
	return v
}

type publicURLReadiness struct {
	Ready  bool
	Reason string
}

type publicURLSmokeResult struct {
	Ready      bool
	StatusCode int
	Reason     string
	Path       string
}

func checkPublicURLReadiness(resp *api.CreateDeploymentResponse) publicURLReadiness {
	if resp == nil || resp.IngressID == uuid.Nil || resp.Domain == "" {
		return publicURLReadiness{Ready: true}
	}

	readiness := publicURLReadiness{Ready: true}
	utils.PrintInfo("Waiting for DNS propagation for https://%s...", resp.Domain)
	if _, err := api.WaitForIngressDNSStatus(resp.IngressID.String(), 2*time.Minute); err != nil {
		readiness.Ready = false
		readiness.Reason = fmt.Sprintf("DNS propagation timed out: %s", err.Error())
		utils.PrintWarning("DNS is still propagating for https://%s: %s", resp.Domain, err.Error())
	}

	status, err := api.GetDomainStatus(resp.IngressID.String(), resp.Domain, false)
	if err != nil {
		if readiness.Ready {
			readiness.Ready = false
			readiness.Reason = fmt.Sprintf("domain status unavailable: %s", err.Error())
		}
		return readiness
	}
	if !domainStatusReady(status) {
		readiness.Ready = false
		readiness.Reason = domainStatusReason(status)
	}
	return readiness
}

func domainStatusReady(status *api.DomainStatusResponse) bool {
	return status != nil &&
		status.Attached &&
		status.Route.Attached &&
		status.DNS.Status == api.DNSStatusResolved
}

func domainStatusReason(status *api.DomainStatusResponse) string {
	if status == nil {
		return "domain status unavailable"
	}
	if !status.Attached {
		return "domain is not attached in backend metadata"
	}
	if !status.Route.Attached {
		if status.Route.Message != "" {
			return "route is not attached: " + status.Route.Message
		}
		return "route is not attached"
	}
	if status.DNS.Status != api.DNSStatusResolved {
		if status.DNS.Message != "" {
			return fmt.Sprintf("DNS is %s: %s", status.DNS.Status, status.DNS.Message)
		}
		return fmt.Sprintf("DNS is %s", status.DNS.Status)
	}
	return "public URL is not ready"
}

func smokePathCandidates(explicitPath string) []string {
	if explicitPath != "" {
		return []string{explicitPath}
	}
	return []string{"/health", "/"}
}

func checkPublicURLSmoke(baseURL string, paths []string, strict bool) publicURLSmokeResult {
	if len(paths) == 0 {
		paths = smokePathCandidates("")
	}

	var failureReasons []string
	var lastFailure publicURLSmokeResult
	for _, path := range paths {
		result := checkPublicURLSmokeAtPath(baseURL, path, strict)
		if result.Ready {
			return result
		}
		failureReasons = append(failureReasons, fmt.Sprintf("%s: %s", path, result.Reason))
		lastFailure = result
	}

	if len(failureReasons) == 1 {
		return lastFailure
	}
	return publicURLSmokeResult{
		Ready:      false,
		Reason:     strings.Join(failureReasons, "; "),
		Path:       lastFailure.Path,
		StatusCode: lastFailure.StatusCode,
	}
}

func checkPublicURLSmokeAtPath(baseURL, path string, strict bool) publicURLSmokeResult {
	targetURL := strings.TrimRight(baseURL, "/") + path
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return publicURLSmokeResult{Ready: false, Reason: fmt.Sprintf("failed to build request: %s", err.Error()), Path: path}
	}

	resp, err := client.Do(req)
	if err != nil {
		return publicURLSmokeResult{Ready: false, Reason: fmt.Sprintf("request failed: %s", err.Error()), Path: path}
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck

	// In non-strict mode (default smoke without explicit --health-path),
	// 401, 403, and 404 prove DNS/TLS/routing worked — the platform is healthy.
	// Only 5xx and connection errors mean the route is truly broken.
	if resp.StatusCode >= http.StatusBadRequest {
		if !strict && isReachabilityStatus(resp.StatusCode) {
			return publicURLSmokeResult{
				Ready:      true,
				StatusCode: resp.StatusCode,
				Path:       path,
			}
		}
		return publicURLSmokeResult{
			Ready:      false,
			StatusCode: resp.StatusCode,
			Reason:     fmt.Sprintf("unexpected HTTP status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
			Path:       path,
		}
	}

	return publicURLSmokeResult{
		Ready:      true,
		StatusCode: resp.StatusCode,
		Path:       path,
	}
}

// isReachabilityStatus returns true for HTTP status codes that still prove
// DNS/TLS/routing worked (platform reachable), even if the app-level response
// is an error.
func isReachabilityStatus(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden || code == http.StatusNotFound
}

func deploymentStrategyText(strategy *api.DeploymentStrategyConfig) string {
	if strategy == nil {
		return "default"
	}
	if strategy.Rolling == nil {
		return string(strategy.Type)
	}
	return fmt.Sprintf("%s (maxSurge=%s, maxUnavailable=%s)",
		strategy.Type,
		strategy.Rolling.MaxSurge,
		strategy.Rolling.MaxUnavailable,
	)
}

func enabledText(enabled bool) string {
	if enabled {
		return "attached"
	}
	return "not attached"
}

func parseEnvVars(envVars []string) []api.KeyValuePair {
	var keyValues []api.KeyValuePair
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			keyValues = append(keyValues, api.KeyValuePair{
				Key:   parts[0],
				Value: parts[1],
			})
		}
	}
	return keyValues
}

func handleDeploymentStatus(ctx context.Context, cmd *cli.Command) error {
	deploymentID, err := resolveDeploymentID(cmd.String("deployment-id"), cmd.String("app"), cmd.String("config"))
	if err != nil {
		return err
	}
	watch := cmd.Bool("watch")

	if watch {
		status, err := api.WaitForDeployment(deploymentID, 5*time.Minute)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to watch deployment: %s", err.Error()), nil)
		}
		utils.PrintStatusLine("Final status", status.Status)
		if status.Message != "" {
			utils.PrintStatusLine("Message", status.Message)
		}
		return nil
	}

	status, err := api.GetDeploymentStatus(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get deployment status: %s", err.Error()), nil)
	}

	deployment, err := api.GetDeployment(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get deployment details: %s", err.Error()), nil)
	}

	var ingress *api.Ingress
	var domainStatus *api.DomainStatusResponse
	if ing, ingErr := api.GetIngressByDeploymentID(deploymentID); ingErr == nil {
		ingress = ing
		if ing.DomainName != "" {
			if ds, dsErr := api.GetDomainStatus(ing.IngressID.String(), ing.DomainName, false); dsErr == nil {
				domainStatus = ds
			}
		}
	}

	details := struct {
		Deployment   *api.Deployment           `json:"deployment"`
		Status       *api.DeploymentStatus     `json:"status"`
		Ingress      *api.Ingress              `json:"ingress,omitempty"`
		DomainStatus *api.DomainStatusResponse `json:"domain_status,omitempty"`
	}{
		Deployment:   deployment,
		Status:       status,
		Ingress:      ingress,
		DomainStatus: domainStatus,
	}
	if utils.TryPrintJSON(details) {
		return nil
	}

	utils.PrintHeader("Deployment Status")
	utils.PrintStatusLine("App", deployment.AppLabel)
	utils.PrintStatusLine("Deployment ID", deployment.DeploymentID.String())
	utils.PrintStatusLine("Namespace", deployment.Namespace)
	if ingress != nil && ingress.DomainName != "" {
		utils.PrintStatusLine("URL", "https://"+ingress.DomainName)
	}
	utils.PrintStatusLine("Workload", status.Status)
	if status.Message != "" {
		utils.PrintStatusLine("Message", status.Message)
	}
	utils.PrintStatusLine("Progress", fmt.Sprintf("%d%%", status.Progress))
	utils.PrintStatusLine("Image", deployment.Image)
	utils.PrintStatusLine("Replicas", fmt.Sprintf("%d desired", deployment.Replicas))
	utils.PrintStatusLine("Strategy", deploymentStrategyText(deployment.StrategyConfig))
	utils.PrintStatusLine("Environment", enabledText(deployment.EnvEnabled))
	utils.PrintStatusLine("Secrets", enabledText(deployment.SecretEnabled))
	utils.PrintStatusLine("Volume", enabledText(deployment.VolumeEnabled))
	if domainStatus != nil {
		utils.PrintStatusLine("Route", domainRouteText(domainStatus.Route))
		utils.PrintStatusLine("DNS", domainDNSText(domainStatus.DNS))
		utils.PrintStatusLine("TLS", domainTLSText(domainStatus.TLS))
	}
	utils.PrintStatusLine("Created", api.FormatTimeAgo(deployment.CreatedAt))
	utils.PrintStatusLine("Last Updated", api.FormatTimeAgo(deployment.UpdatedAt))

	return nil
}

// Note: deployment log streaming is implemented in `1ctl logs` for now;
// `1ctl deploy logs` will land alongside the backend WS endpoint in a
// follow-up (#3 G-01).

func handleListDeployments(ctx context.Context, cmd *cli.Command) error {
	namespace, err := satuskyctx.GetCurrentNamespaceOrError()
	if err != nil {
		return err
	}
	deployments, err := api.ListDeploymentsByNamespace(namespace)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list deployments: %s", err.Error()), nil)
	}

	if len(deployments) == 0 {
		utils.PrintInfo("No deployments found")
		return nil
	}

	if utils.TryPrintJSON(deployments) {
		return nil
	}

	// NAME column added (issue #29). Falls back to "-" for legacy
	// deployments that pre-date the app_label field.
	headers := []string{"NAME", "DEPLOYMENT ID", "HOSTNAMES", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(deployments))
	for _, d := range deployments {
		name := d.AppLabel
		if name == "" {
			name = "-"
		}
		rows = append(rows, []string{
			name,
			d.DeploymentID.String(),
			strings.Join(d.Hostnames, ", "),
			d.Status,
			api.FormatTimeAgo(d.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)

	return nil
}

func handleGetDeployment(ctx context.Context, cmd *cli.Command) error {
	deploymentID, err := resolveDeploymentID(cmd.String("deployment-id"), cmd.String("app"), cmd.String("config"))
	if err != nil {
		return err
	}
	deployment, err := api.GetDeployment(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get deployment: %s", err.Error()), nil)
	}

	// Enrich with ingress domain — best-effort, don't fail if not yet created
	if ingress, iErr := api.GetIngressByDeploymentID(deploymentID); iErr == nil && ingress != nil && ingress.DomainName != "" {
		deployment.Domain = "https://" + ingress.DomainName
	}

	if utils.TryPrintJSON(deployment) {
		return nil
	}

	// Print detailed deployment information
	utils.PrintHeader("Deployment Details")
	if deployment.MarketplaceAppName != "" {
		utils.PrintStatusLine("Application (from marketplace)", deployment.MarketplaceAppName)
	}
	utils.PrintStatusLine("Deployment ID", deployment.DeploymentID.String())
	utils.PrintStatusLine("Status", deployment.Status)
	if deployment.Domain != "" {
		utils.PrintStatusLine("URL", deployment.Domain)
	}
	utils.PrintStatusLine("Deployed to machines", strings.Join(deployment.Hostnames, ", "))
	utils.PrintStatusLine("Type", deployment.Type)
	utils.PrintStatusLine("Zone", deployment.Zone)
	// Image refs without a ":tag" (e.g., "nginx", "registry.example.com/app")
	// are legal; show "untagged" rather than indexing into a 1-element slice.
	version := "untagged"
	if parts := strings.SplitN(deployment.Image, ":", 2); len(parts) == 2 {
		version = parts[1]
	}
	utils.PrintStatusLine("Version", version)
	utils.PrintStatusLine("Port", fmt.Sprintf("%d", deployment.Port))
	utils.PrintStatusLine("CPU Request", deployment.CpuRequest)
	utils.PrintStatusLine("Memory Request", deployment.MemoryRequest)
	utils.PrintStatusLine("Memory Limit", deployment.MemoryLimit)
	utils.PrintStatusLine("Created", api.FormatTimeAgo(deployment.CreatedAt))
	utils.PrintStatusLine("Last Updated", api.FormatTimeAgo(deployment.UpdatedAt))
	return nil
}

func handleDestroyDeployment(ctx context.Context, cmd *cli.Command) error {
	deploymentID, err := resolveDeploymentID(cmd.String("deployment-id"), cmd.String("app"), cmd.String("config"))
	if err != nil {
		return err
	}
	if !utils.Confirm(
		fmt.Sprintf("Destroy deployment %s? This will delete all associated services, ingresses, and volumes. This cannot be undone.", deploymentID),
		cmd.Bool("yes"),
	) {
		fmt.Println("Aborted.")
		return nil
	}
	utils.PrintInfo("Destroying deployment %s...", deploymentID)
	result, err := api.DeleteDeployment(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to destroy deployment: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(result) {
		return nil
	}
	printDeletionResult(deploymentID, result)
	return nil
}

func printDeletionResult(deploymentID string, result *api.DeletionResult) {
	utils.PrintSuccess("Deployment %s destroy completed", deploymentID)
	if result == nil {
		return
	}
	utils.PrintHeader("Deleted Resources")
	if result.AppLabel != "" {
		utils.PrintStatusLine("App", result.AppLabel)
	}
	if result.Namespace != "" {
		utils.PrintStatusLine("Namespace", result.Namespace)
	}
	if len(result.DeletedDeployments) > 0 {
		utils.PrintStatusLine("Deployments", strings.Join(result.DeletedDeployments, ", "))
	} else {
		utils.PrintStatusLine("Deployments", "none reported")
	}
	if result.IsCNPGDeployment {
		utils.PrintStatusLine("CNPG", "database deployment cleanup applied")
	}
	if len(result.Volumes) == 0 {
		utils.PrintStatusLine("PVCs", "none reported")
		return
	}
	headers := []string{"PVC", "VOLUME", "STATUS", "POLICY", "MESSAGE"}
	rows := make([][]string, 0, len(result.Volumes))
	for _, volume := range result.Volumes {
		rows = append(rows, []string{
			volume.ClaimName,
			volume.VolumeName,
			volume.Status,
			volume.DestroyPolicy,
			volume.Message,
		})
	}
	utils.PrintTable(headers, rows)
}

func handleRestartDeployment(ctx context.Context, cmd *cli.Command) error {
	deploymentID, err := resolveDeploymentID(cmd.String("deployment-id"), cmd.String("app"), cmd.String("config"))
	if err != nil {
		return err
	}
	utils.PrintInfo("Initiating rolling restart for deployment %s...", deploymentID)
	if err := api.RestartDeployment(deploymentID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to restart: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Rolling restart initiated. Pods are being replaced one by one.")
	utils.PrintInfo("Use '1ctl deploy status --deployment-id %s' to monitor progress.", deploymentID)
	return nil
}

func handleListReleases(ctx context.Context, cmd *cli.Command) error {
	deploymentID, err := resolveDeploymentID(cmd.String("deployment-id"), cmd.String("app"), cmd.String("config"))
	if err != nil {
		return err
	}
	versions, err := api.ListDeploymentVersions(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list releases: %s", err.Error()), nil)
	}
	if len(versions) == 0 {
		utils.PrintInfo("No releases found")
		return nil
	}
	headers := []string{"VERSION", "IMAGE", "STATUS", "DEPLOYED"}
	rows := make([][]string, 0, len(versions))
	for _, v := range versions {
		rows = append(rows, []string{
			fmt.Sprintf("%d", v.VersionNumber),
			v.Image,
			v.Status,
			api.FormatTimeAgo(v.DeployedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleRollback(ctx context.Context, cmd *cli.Command) error {
	deploymentID, err := resolveDeploymentID(cmd.String("deployment-id"), cmd.String("app"), cmd.String("config"))
	if err != nil {
		return err
	}
	var version int
	if cmd.IsSet("version") {
		version = cmd.Int("version")
	} else {
		// Default: roll back to previous version (versions[0] is active, versions[1] is previous)
		versions, err := api.ListDeploymentVersions(deploymentID)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to fetch releases: %s", err.Error()), nil)
		}
		if len(versions) < 2 {
			return utils.NewError("no previous release to roll back to", nil)
		}
		version = versions[1].VersionNumber
	}

	if !utils.Confirm(fmt.Sprintf("Roll back deployment %s to version %d? This cannot be undone.", deploymentID, version), cmd.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := api.RollbackDeployment(deploymentID, version); err != nil {
		return utils.NewError(fmt.Sprintf("rollback failed: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Rollback to version %d initiated", version)
	utils.PrintInfo("Use '1ctl deploy status --deployment-id %s' to monitor progress.", deploymentID)
	return nil
}

// handleOpenDeployment opens the deployment's primary URL in the user's
// default browser. Resolves the URL from the ingress record, falling back
// to a clear error when no ingress is attached yet.
func handleOpenDeployment(ctx context.Context, cmd *cli.Command) error {
	deploymentID, err := resolveDeploymentID(cmd.String("deployment-id"), cmd.String("app"), cmd.String("config"))
	if err != nil {
		return err
	}
	ing, err := api.GetIngressByDeploymentID(deploymentID)
	if err != nil || ing == nil || ing.DomainName == "" {
		return utils.NewError(fmt.Sprintf("no domain attached to deployment %s — use '1ctl domains add' first", deploymentID), nil)
	}
	url := "https://" + ing.DomainName
	utils.PrintInfo("Opening %s", url)
	if err := openBrowser(url); err != nil {
		// Don't fail the command — print the URL so the user can copy it.
		utils.PrintWarning("Could not open browser: %s", err.Error())
		utils.PrintInfo("URL: %s", url)
	}
	return nil
}

// handleScaleDeployment sets the replica count on an existing deployment
// without rebuilding the image. Uses UpsertDeployment after fetching the
// current state so all other fields are preserved.
func handleScaleDeployment(ctx context.Context, cmd *cli.Command) error {
	deploymentID, err := resolveDeploymentID(cmd.String("deployment-id"), cmd.String("app"), cmd.String("config"))
	if err != nil {
		return err
	}
	replicas, err := api.SafeInt32(cmd.Int("replicas"))
	if err != nil {
		return utils.NewError("invalid --replicas value", err)
	}
	if replicas < 1 {
		return utils.NewError("--replicas must be >= 1", nil)
	}

	current, err := api.GetDeployment(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to fetch deployment: %s", err.Error()), nil)
	}
	// Refuse to scale autoscaler-managed deployments: HPA/VPA take over
	// replica counts, so a manual `scale` would race the controller and
	// the user should disable/adjust the autoscaler instead. Also a guard
	// against the GetDeployment-then-UpsertDeployment round trip silently
	// dropping nested config if the backend GET ever flattens fields.
	if current.HPAConfig != nil && current.HPAConfig.Enabled {
		return utils.NewError(fmt.Sprintf("deployment %s is managed by HPA — adjust --hpa-min-replicas / --hpa-max-replicas via `1ctl deploy` instead", deploymentID), nil)
	}
	if current.VPAConfig != nil && current.VPAConfig.Enabled {
		return utils.NewError(fmt.Sprintf("deployment %s is managed by VPA — adjust VPA config via `1ctl deploy` instead", deploymentID), nil)
	}
	if current.Replicas == replicas {
		utils.PrintInfo("Deployment %s already at %d replicas — no change.", deploymentID, replicas)
		return nil
	}
	current.Replicas = replicas

	var resp string
	if err := api.UpsertDeployment(*current, &resp); err != nil {
		return utils.NewError(fmt.Sprintf("failed to scale deployment: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Scaled deployment %s to %d replicas", deploymentID, replicas)
	return nil
}
