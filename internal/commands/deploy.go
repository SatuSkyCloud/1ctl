package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/config"
	"1ctl/internal/context"
	"1ctl/internal/deploy"
	"1ctl/internal/utils"
	"1ctl/internal/validator"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

func DeployCommand() *cli.Command {
	deployFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  "cpu",
			Usage: "CPU cores allocation (e.g., '2')",
			Value: "0.5",
		},
		&cli.StringFlag{
			Name:  "memory",
			Usage: "Memory allocation (e.g., '512Mi', '2Gi')",
			Value: "256Mi",
		},
		&cli.StringSliceFlag{
			Name:  "machine",
			Usage: "Machine name to deploy to (e.g., 'machine1, machine2')",
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
			Name:  "dockerfile",
			Usage: "Dockerfile path for cloud build (default: Dockerfile)",
			Value: "Dockerfile",
		},
		&cli.StringFlag{
			Name:  "image",
			Usage: "Pre-built image reference — skips cloud build entirely (e.g. registry.satusky.com/satusky-container-registry/myapp:abc1234)",
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

Images are built in the cloud via Kaniko — no local Docker installation required.
Your source directory is packaged and sent to the build service, which builds
and pushes the image directly to the registry.

   1ctl deploy --cpu 1 --memory 512Mi --port 8080
   1ctl deploy --cpu 2 --memory 1Gi --port 3000 --env KEY=VALUE
   1ctl deploy --image registry.satusky.com/.../myapp:tag   # skip cloud build

Config file (satusky.toml) can persist deploy settings across runs.
Run '1ctl init' to create one.

Subcommands manage existing deployments:
   1ctl deploy list
   1ctl deploy status --deployment-id <id>
   1ctl deploy restart --deployment-id <id>
   1ctl deploy releases --deployment-id <id>
   1ctl deploy rollback --deployment-id <id>
   1ctl deploy destroy --deployment-id <id>`,
		Flags: deployFlags,
		Subcommands: []*cli.Command{
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
					&cli.StringFlag{Name: "config", Usage: "Config name or path. Default: satusky.toml"},
				},
				Action: handleListReleases,
			},
			{
				Name:  "rollback",
				Usage: "Roll back to a previous release",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "deployment-id", Usage: "Deployment ID"},
					&cli.StringFlag{Name: "config", Usage: "Config name or path. Default: satusky.toml"},
					&cli.IntFlag{Name: "version", Usage: "Version number to roll back to (default: previous version)"},
					&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"},
				},
				Action: handleRollback,
			},
		},
		Action: func(c *cli.Context) error {
			// If subcommand is provided, let it handle
			if c.NArg() > 0 {
				return cli.ShowSubcommandHelp(c)
			}

			// If no flags and no satusky.toml with cpu/memory, show help
			if !c.IsSet("cpu") && !c.IsSet("memory") && !c.IsSet("image") {
				if cfg, err := config.FindConfig(c.String("config")); err != nil || cfg == nil || (cfg.App.CPU == "" && cfg.App.Memory == "") {
					return cli.ShowSubcommandHelp(c)
				}
			}

			// Otherwise, handle deploy
			return handleDeploy(c)
		},
	}
}

func handleDeploy(c *cli.Context) error {
	// Check token expiry before any work begins to fail fast with a clear message
	if err := context.CheckTokenExpiry(); err != nil {
		return err
	}

	// Load satusky.toml defaults for flags not set on the CLI
	if cfg, err := config.FindConfig(c.String("config")); err == nil && cfg != nil {
		if !c.IsSet("cpu") && cfg.App.CPU != "" {
			if err := c.Set("cpu", cfg.App.CPU); err != nil {
				return utils.NewError(fmt.Sprintf("failed to set cpu from config: %s", err.Error()), nil)
			}
		}
		if !c.IsSet("memory") && cfg.App.Memory != "" {
			if err := c.Set("memory", cfg.App.Memory); err != nil {
				return utils.NewError(fmt.Sprintf("failed to set memory from config: %s", err.Error()), nil)
			}
		}
		if !c.IsSet("port") && cfg.App.Port != 0 {
			if err := c.Set("port", fmt.Sprintf("%d", cfg.App.Port)); err != nil {
				return utils.NewError(fmt.Sprintf("failed to set port from config: %s", err.Error()), nil)
			}
		}
		if !c.IsSet("domain") && cfg.App.Domain != "" {
			if err := c.Set("domain", cfg.App.Domain); err != nil {
				return utils.NewError(fmt.Sprintf("failed to set domain from config: %s", err.Error()), nil)
			}
		}
	}

	// Validate inputs first
	if err := validateInputs(c); err != nil {
		return utils.NewError(fmt.Sprintf("validation failed: %s", err.Error()), nil)
	}

	// Prepare deployment options
	opts, err := prepareDeploymentOptions(c)
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

	utils.PrintSuccess("🚀 Deployment for %s is successful! Your app is live at: https://%s", resp.AppLabel, resp.Domain)
	utils.PrintStatusLine("Deployment ID", resp.DeploymentID.String())

	if opts.Wait {
		utils.PrintInfo("Waiting for deployment to become healthy...")
		status, err := api.WaitForDeployment(resp.DeploymentID.String(), 5*time.Minute)
		if err != nil {
			utils.PrintWarning("Timed out waiting for deployment: %s", err.Error())
		} else if status != nil && status.Status == "Running" {
			utils.PrintSuccess("Deployment is healthy — pods Running")
		}
	}

	return nil
}

func validateInputs(c *cli.Context) error {
	// Validate Dockerfile only when not using a pre-built image
	if !c.IsSet("image") {
		dockerfilePath := c.String("dockerfile")
		if err := validator.ValidateDockerfile(dockerfilePath); err != nil {
			// Try to find Dockerfile in common locations
			var err error
			dockerfilePath, err = validator.FindDockerfile(".")
			if err != nil {
				return utils.NewError("no valid Dockerfile found: please ensure a Dockerfile exists in your project", err)
			}
			// Update the context with the found Dockerfile path
			if err := c.Set("dockerfile", dockerfilePath); err != nil {
				return utils.NewError(fmt.Sprintf("failed to set dockerfile path: %s", err.Error()), nil)
			}
		}
	}

	// Validate CPU and Memory
	if err := validator.ValidateCPU(c.String("cpu")); err != nil {
		return utils.NewError("invalid CPU value: %v", err)
	}
	if err := validator.ValidateMemory(c.String("memory")); err != nil {
		return utils.NewError("invalid memory value: %v", err)
	}
	if err := validator.ValidateDomain(c.String("domain")); err != nil {
		return utils.NewError("invalid domain: %v", err)
	}

	// Validate volume options
	if c.IsSet("volume-size") || c.IsSet("volume-mount") {
		if c.String("volume-size") == "" {
			return utils.NewError("volume-size is required when volume is enabled", nil)
		}
		if c.String("volume-mount") == "" {
			return utils.NewError("volume-mount is required when volume is enabled", nil)
		}
		if err := validator.ValidateMemory(c.String("volume-size")); err != nil {
			return utils.NewError("invalid volume size: %v", err)
		}
	}

	// Validate HA settings
	if c.IsSet("hpa") && c.IsSet("vpa") && c.Bool("hpa") && c.Bool("vpa") {
		vpaMode := c.String("vpa-mode")
		if vpaMode == "Auto" {
			return utils.NewError("HPA and VPA with mode 'Auto' cannot be used together - they both try to scale resources", nil)
		}
	}
	if c.IsSet("hpa-min-replicas") && c.IsSet("hpa-max-replicas") {
		if c.Int("hpa-min-replicas") > c.Int("hpa-max-replicas") {
			return utils.NewError("hpa-min-replicas cannot be greater than hpa-max-replicas", nil)
		}
	}
	if c.IsSet("pdb-percent") {
		if c.Int("pdb-percent") < 1 || c.Int("pdb-percent") > 100 {
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
	if c.Bool("multicluster") && c.IsSet("domain") {
		domain := strings.TrimSpace(strings.ToLower(c.String("domain")))
		domain = strings.TrimPrefix(domain, "*.")
		if domain != "" && domain != "satusky.com" && !strings.HasSuffix(domain, ".satusky.com") {
			return utils.NewError(fmt.Sprintf(
				"--multicluster is not supported with custom domains yet: %q is not a *.satusky.com hostname. "+
					"Use a *.satusky.com hostname or drop --multicluster.", c.String("domain")), nil)
		}
	}

	// Validate --wait-for entries
	for _, v := range c.StringSlice("wait-for") {
		if _, _, err := validator.ValidateWaitFor(v); err != nil {
			return err
		}
	}

	return nil
}

func prepareDeploymentOptions(c *cli.Context) (deploy.DeploymentOptions, error) {
	opts := deploy.DeploymentOptions{
		CPU:            c.String("cpu"),
		Memory:         c.String("memory"),
		Domain:         c.String("domain"),
		Organization:   c.String("organization"),
		Port:           c.Int("port"),
		DockerfilePath: c.String("dockerfile"),
		PrebuiltImage:  c.String("image"),
	}

	// Load app name from satusky.toml if present
	if cfg, err := config.FindConfig(c.String("config")); err == nil && cfg != nil && cfg.App.Name != "" {
		opts.Name = cfg.App.Name
	}

	// Handle project organization for future use (multi-tenant)
	if c.IsSet("organization") {
		opts.Organization = c.String("organization")
	} else {
		opts.Organization = context.GetCurrentNamespace()
	}

	// Handle environment variables if enabled when --env are set
	if c.IsSet("env") {
		opts.EnvEnabled = true
		env := &api.Environment{
			KeyValues: parseEnvVars(c.StringSlice("env")),
		}
		opts.Environment = env
	}

	// Handle hostnames if enabled when --machine is set
	if c.IsSet("machine") {
		machineNames := c.StringSlice("machine")
		hostnameSet := make(map[string]bool) // Add deduplication for manually specified machines
		for _, machineName := range machineNames {
			machine, err := api.GetMachineByName(machineName)
			if err != nil {
				return deploy.DeploymentOptions{}, utils.NewError(fmt.Sprintf("failed to get machine by name: %s", err.Error()), nil)
			}

			// check if machine is owned by the current user (monetized machines can be used by anyone)
			if !machine.Monetized && machine.OwnerID.String() != context.GetUserID() {
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
	if c.IsSet("volume-size") || c.IsSet("volume-mount") {
		opts.VolumeEnabled = true
		vol := &api.Volume{
			StorageSize: c.String("volume-size"),
			MountPath:   c.String("volume-mount"),
		}
		opts.Volume = vol
	}

	// Handle zone targeting
	if c.IsSet("zone") {
		opts.Zone = c.String("zone")
		// For backward compatibility, also set Region to the zone value
		// The backend uses Region for machine filtering when Zone is not explicitly handled
		opts.Region = c.String("zone")
	}

	// Handle multicluster configuration
	if c.Bool("multicluster") {
		opts.MulticlusterEnabled = true
		opts.MulticlusterMode = c.String("multicluster-mode")
		opts.BackupEnabled = c.Bool("backup-enabled")
		opts.BackupSchedule = c.String("backup-schedule")
		opts.BackupRetention = c.String("backup-retention")
		opts.BackupPriorityCluster = c.Int("backup-priority-cluster")
	}

	// Handle replica count override
	if c.IsSet("replicas") {
		opts.Replicas = c.Int("replicas")
	}

	// Handle PDB configuration
	if c.IsSet("pdb") && c.Bool("pdb") {
		pdbConfig := &deploy.PDBConfig{
			Enabled: true,
			Type:    deploy.PDBConfigType(c.String("pdb-type")),
		}
		if c.IsSet("pdb-min-available") {
			val, err := api.SafeInt32(c.Int("pdb-min-available"))
			if err != nil {
				return deploy.DeploymentOptions{}, utils.NewError("invalid pdb-min-available value", err)
			}
			pdbConfig.MinAvailable = &val
		}
		if c.IsSet("pdb-percent") {
			val, err := api.SafeInt32(c.Int("pdb-percent"))
			if err != nil {
				return deploy.DeploymentOptions{}, utils.NewError("invalid pdb-percent value", err)
			}
			pdbConfig.Percent = &val
		}
		opts.PDBConfig = pdbConfig
	}

	// Handle HPA configuration
	if c.IsSet("hpa") && c.Bool("hpa") {
		cpuTarget, err := api.SafeInt32(c.Int("hpa-cpu-target"))
		if err != nil {
			return deploy.DeploymentOptions{}, utils.NewError("invalid hpa-cpu-target value", err)
		}
		minReplicas, err := api.SafeInt32(c.Int("hpa-min-replicas"))
		if err != nil {
			return deploy.DeploymentOptions{}, utils.NewError("invalid hpa-min-replicas value", err)
		}
		maxReplicas, err := api.SafeInt32(c.Int("hpa-max-replicas"))
		if err != nil {
			return deploy.DeploymentOptions{}, utils.NewError("invalid hpa-max-replicas value", err)
		}
		hpaConfig := &api.HPAConfig{
			Enabled:     true,
			MinReplicas: minReplicas,
			MaxReplicas: maxReplicas,
			CPUTarget:   &cpuTarget,
		}
		if c.IsSet("hpa-memory-target") && c.Int("hpa-memory-target") > 0 {
			memTarget, err := api.SafeInt32(c.Int("hpa-memory-target"))
			if err != nil {
				return deploy.DeploymentOptions{}, utils.NewError("invalid hpa-memory-target value", err)
			}
			hpaConfig.MemoryTarget = &memTarget
		}
		opts.HPAConfig = hpaConfig
	}

	// Handle VPA configuration
	if c.IsSet("vpa") && c.Bool("vpa") {
		opts.VPAConfig = &api.VPAConfig{
			Enabled:    true,
			UpdateMode: c.String("vpa-mode"),
			MinCPU:     c.String("vpa-min-cpu"),
			MaxCPU:     c.String("vpa-max-cpu"),
			MinMemory:  c.String("vpa-min-memory"),
			MaxMemory:  c.String("vpa-max-memory"),
		}
	}

	// Handle --wait-for dependencies
	for _, v := range c.StringSlice("wait-for") {
		host, port, err := validator.ValidateWaitFor(v)
		if err != nil {
			return deploy.DeploymentOptions{}, err
		}
		opts.WaitFor = append(opts.WaitFor, api.WaitFor{Host: host, Port: port})
	}

	// Handle deployment strategy options
	opts.Strategy = c.String("strategy")
	opts.RollingMaxSurge = c.String("rolling-max-surge")
	opts.RollingMaxUnavailable = c.String("rolling-max-unavailable")
	opts.Wait = c.Bool("wait")

	// Validate strategy value
	switch opts.Strategy {
	case "rolling", "recreate":
		// valid
	default:
		return deploy.DeploymentOptions{}, utils.NewError(fmt.Sprintf("invalid --strategy %q: must be 'rolling' or 'recreate'", opts.Strategy), nil)
	}

	return opts, nil
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

func handleDeploymentStatus(c *cli.Context) error {
	deploymentID, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
	if err != nil {
		return err
	}
	watch := c.Bool("watch")

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

	if utils.TryPrintJSON(status) {
		return nil
	}

	utils.PrintStatusLine("Status", status.Status)
	if status.Message != "" {
		utils.PrintStatusLine("Message", status.Message)
	}
	utils.PrintStatusLine("Progress", fmt.Sprintf("%d%%", status.Progress))

	return nil
}

// TODO: deployment logs is still WIP on the backend... (/ws)
// func handleDeploymentLogs(c *cli.Context) error {
// 	deploymentID := c.String("deployment-id")
// 	follow := c.Bool("follow")

// 	if follow {
// 		utils.PrintInfo("Streaming logs for deployment %s...\n", deploymentID)
// 		return api.StreamDeploymentLogs(deploymentID, true)
// 	}

// 	logs, err := api.GetDeploymentLogs(deploymentID)
// 	if err != nil {
// 		utils.PrintError(fmt.Sprintf("failed to get deployment logs: %s", err.Error()), nil)
// 		return nil
// 	}

// 	for _, line := range logs {
// 		utils.PrintInfo(line)
// 	}
// 	return nil
// }

func handleListDeployments(c *cli.Context) error {
	var deployments []api.Deployment
	var err error
	namespace := context.GetCurrentNamespace()

	if namespace != "" {
		deployments, err = api.ListDeploymentsByNamespace(namespace)
	} else {
		deployments, err = api.ListDeployments()
	}

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

	headers := []string{"DEPLOYMENT ID", "HOSTNAMES", "TYPE", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(deployments))
	for _, d := range deployments {
		rows = append(rows, []string{
			d.DeploymentID.String(),
			strings.Join(d.Hostnames, ", "),
			d.Type,
			d.Status,
			api.FormatTimeAgo(d.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)

	return nil
}

func handleGetDeployment(c *cli.Context) error {
	deploymentID, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
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
	utils.PrintStatusLine("Region", deployment.Region)
	utils.PrintStatusLine("Zone", deployment.Zone)
	utils.PrintStatusLine("Version", strings.Split(deployment.Image, ":")[1])
	utils.PrintStatusLine("Port", fmt.Sprintf("%d", deployment.Port))
	utils.PrintStatusLine("CPU Request", deployment.CpuRequest)
	utils.PrintStatusLine("Memory Request", deployment.MemoryRequest)
	utils.PrintStatusLine("Memory Limit", deployment.MemoryLimit)
	utils.PrintStatusLine("Created", api.FormatTimeAgo(deployment.CreatedAt))
	utils.PrintStatusLine("Last Updated", api.FormatTimeAgo(deployment.UpdatedAt))
	return nil
}

func handleDestroyDeployment(c *cli.Context) error {
	deploymentID, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
	if err != nil {
		return err
	}
	if !utils.Confirm(
		fmt.Sprintf("Destroy deployment %s? This will delete all associated services, ingresses, and volumes. This cannot be undone.", deploymentID),
		c.Bool("yes"),
	) {
		fmt.Println("Aborted.")
		return nil
	}
	utils.PrintInfo("Destroying deployment %s...", deploymentID)
	if err := api.DeleteDeployment(deploymentID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to destroy deployment: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Deployment %s destroyed successfully", deploymentID)
	return nil
}

func handleRestartDeployment(c *cli.Context) error {
	deploymentID, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
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

func handleListReleases(c *cli.Context) error {
	deploymentID, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
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

func handleRollback(c *cli.Context) error {
	deploymentID, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
	if err != nil {
		return err
	}
	var version int
	if c.IsSet("version") {
		version = c.Int("version")
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

	if !utils.Confirm(fmt.Sprintf("Roll back deployment %s to version %d? This cannot be undone.", deploymentID, version), c.Bool("yes")) {
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
