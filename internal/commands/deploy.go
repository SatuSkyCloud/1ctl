package commands

import (
	"1ctl/internal/api"
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
			Name:     "cpu",
			Usage:    "CPU cores allocation (e.g., '2')",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "memory",
			Usage:    "Memory allocation (e.g., '512Mi', '2Gi')",
			Required: false,
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
			Usage: "Path to Dockerfile (default: ./Dockerfile)",
			Value: "Dockerfile",
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
	}

	return &cli.Command{
		Name:  "deploy",
		Usage: "Deploy your application to Satusky Cloud",
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
						Name:     "deployment-id",
						Usage:    "Deployment ID to get details for",
						Required: true,
					},
				},
				Action: handleGetDeployment,
			},
			{
				Name:  "status",
				Usage: "Check deployment status",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "deployment-id",
						Usage:    "Deployment ID to check",
						Required: true,
					},
					&cli.BoolFlag{
						Name:  "watch",
						Usage: "Watch deployment status in real-time",
					},
				},
				Action: handleDeploymentStatus,
			},
		},
		Action: func(c *cli.Context) error {
			// If no subcommand is provided and no flags, show help
			if c.NArg() == 0 && !c.IsSet("cpu") && !c.IsSet("memory") {
				return cli.ShowCommandHelp(c, "deploy")
			}

			// If subcommand is provided, let it handle
			if c.NArg() > 0 {
				return cli.ShowSubcommandHelp(c)
			}

			// Otherwise, handle deploy
			return handleDeploy(c)
		},
	}
}

func handleDeploy(c *cli.Context) error {
	// Check required flags for deploy
	if c.String("cpu") == "" {
		return utils.NewError("--cpu flag is required for deployment", nil)
	}
	if c.String("memory") == "" {
		return utils.NewError("--memory flag is required for deployment", nil)
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
		return utils.NewError(fmt.Sprintf("deployment failed: %s", err.Error()), nil)
	}

	utils.PrintSuccess("🚀 Deployment for %s is successful! Your app is live at: https://%s", resp.AppLabel, resp.Domain)
	return nil
}

func validateInputs(c *cli.Context) error {
	// Validate Dockerfile first
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

			// check if machine is owned by the current user
			if machine.OwnerID.String() != context.GetUserID() {
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

	// Handle multicluster configuration
	if c.Bool("multicluster") {
		opts.MulticlusterEnabled = true
		opts.MulticlusterMode = c.String("multicluster-mode")
		opts.BackupEnabled = c.Bool("backup-enabled")
		opts.BackupSchedule = c.String("backup-schedule")
		opts.BackupRetention = c.String("backup-retention")
		opts.BackupPriorityCluster = c.Int("backup-priority-cluster")
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
	deploymentID := c.String("deployment-id")
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

	utils.PrintHeader("Deployments")
	for _, d := range deployments {
		utils.PrintStatusLine("Deployment ID", d.DeploymentID.String())
		utils.PrintStatusLine("Hostnames", strings.Join(d.Hostnames, ", "))
		utils.PrintStatusLine("Type", d.Type)
		utils.PrintStatusLine("Created", api.FormatTimeAgo(d.CreatedAt))
		utils.PrintDivider()
	}

	return nil
}

func handleGetDeployment(c *cli.Context) error {
	deploymentID := c.String("deployment-id")
	deployment, err := api.GetDeployment(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get deployment: %s", err.Error()), nil)
	}

	// Print detailed deployment information
	utils.PrintHeader("Deployment Details")
	if deployment.MarketplaceAppName != "" {
		utils.PrintStatusLine("Application (from marketplace)", deployment.MarketplaceAppName)
	}
	utils.PrintStatusLine("Deployment ID", deployment.DeploymentID.String())
	utils.PrintStatusLine("Status", deployment.Status)
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
