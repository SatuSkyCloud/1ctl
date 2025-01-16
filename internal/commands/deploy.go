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
			Required: true,
		},
		&cli.StringFlag{
			Name:     "memory",
			Usage:    "Memory allocation (e.g., '512Mi', '2Gi')",
			Required: true,
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
	}

	return &cli.Command{
		Name:  "deploy",
		Usage: "Deploy your application to Satusky Cloud",
		Subcommands: []*cli.Command{
			{
				Name:   "create",
				Usage:  "Create a new deployment",
				Flags:  deployFlags,
				Action: handleDeploy,
			},
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
			// {
			// 	Name:  "logs",
			// 	Usage: "View deployment logs",
			// 	Flags: []cli.Flag{
			// 		&cli.StringFlag{
			// 			Name:     "deployment-id",
			// 			Usage:    "Deployment ID",
			// 			Required: true,
			// 		},
			// 		&cli.BoolFlag{
			// 			Name:  "follow",
			// 			Usage: "Follow log output",
			// 		},
			// 	},
			// 	Action: handleDeploymentLogs,
			// },
		},
		Action: func(c *cli.Context) error {
			// If no subcommand is provided, show help
			return cli.ShowSubcommandHelp(c)
		},
	}
}

func handleDeploy(c *cli.Context) error {
	// Validate inputs first
	if err := validateInputs(c); err != nil {
		return utils.NewError("validation failed: %w", err)
	}

	// Prepare deployment options
	opts, err := prepareDeploymentOptions(c)
	if err != nil {
		return utils.NewError("deployment preparation failed: %w", err)
	}

	// Execute deployment
	resp, err := deploy.Deploy(opts)
	if err != nil {
		return utils.NewError("deployment failed: %w", err)
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
			return utils.NewError("failed to set dockerfile path: %w", err)
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
		for _, machineName := range machineNames {
			machine, err := api.GetMachineByName(machineName)
			if err != nil {
				return deploy.DeploymentOptions{}, utils.NewError("failed to get machine by name: %w", err)
			}

			// check if machine is owned by the current user
			if machine.OwnerID.String() != context.GetUserID() {
				return deploy.DeploymentOptions{}, utils.NewError(fmt.Sprintf("machine %s is not owned by you", machineName), nil)
			}

			opts.Hostnames = append(opts.Hostnames, machine.MachineName)
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
			return utils.NewError("failed to watch deployment: %w", err)
		}
		utils.PrintStatusLine("Final status", status.Status)
		if status.Message != "" {
			utils.PrintStatusLine("Message", status.Message)
		}
		return nil
	}

	status, err := api.GetDeploymentStatus(deploymentID)
	if err != nil {
		return utils.NewError("failed to get deployment status: %w", err)
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
// 		utils.PrintError("failed to get deployment logs: %w", err)
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
		return utils.NewError("failed to list deployments: %w", err)
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
		return utils.NewError("failed to get deployment: %w", err)
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
