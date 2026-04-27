package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func EnvironmentCommand() *cli.Command {
	return &cli.Command{
		Name:    "env",
		Aliases: []string{"environment"},
		Usage:   "Manage environments for a deployment",
		Subcommands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Create a new environment",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "deployment-id",
						Usage: "Deployment ID",
					},
					&cli.StringFlag{
						Name:  "config",
						Usage: "Config name or path (e.g. staging, satusky.staging.toml). Default: satusky.toml",
					},
					&cli.StringFlag{
						Name:  "name",
						Usage: "App label (defaults to deployment name, auto-resolved from deployment-id)",
					},
					&cli.StringSliceFlag{
						Name:  "env",
						Usage: "Environment variables (format: KEY=VALUE)",
					},
				},
				Action: handleCreateEnvironment,
			},
			{
				Name:  "list",
				Usage: "List all environments",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "deployment-id",
						Usage: "Filter by deployment ID",
					},
				},
				Action: handleListEnvironments,
			},
			{
				Name:  "unset",
				Usage: "Remove a specific key from an environment",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "config", Usage: "Config name or path"},
					&cli.StringFlag{Name: "deployment-id", Aliases: []string{"d"}, Usage: "Deployment ID"},
					&cli.StringFlag{Name: "key", Aliases: []string{"k"}, Usage: "Key to remove", Required: true},
				},
				Action: handleEnvUnset,
			},
		},
	}
}

func handleCreateEnvironment(c *cli.Context) error {
	deploymentIDStr, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
	if err != nil {
		return err
	}

	deploymentID, err := uuid.Parse(deploymentIDStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	envVars := c.StringSlice("env")
	keyValues := make([]api.KeyValuePair, 0, len(envVars))

	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return utils.NewError("invalid environment variable format", nil)
		}
		keyValues = append(keyValues, api.KeyValuePair{
			Key:   parts[0],
			Value: parts[1],
		})
	}

	appLabel := c.String("name")
	if appLabel == "" {
		deployment, err := api.GetDeployment(deploymentIDStr)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to resolve deployment name: %s", err.Error()), nil)
		}
		appLabel = deployment.AppLabel
	}

	env := api.Environment{
		DeploymentID: deploymentID,
		AppLabel:     appLabel,
		KeyValues:    keyValues,
	}

	envResp, err := api.UpsertEnvironment(env)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to upsert environment: %s", err.Error()), nil)
	}

	displayName := envResp.AppLabel
	if displayName == "" {
		displayName = appLabel
	}
	utils.PrintSuccess("Environment %s created successfully\n", displayName)
	return nil
}

func handleListEnvironments(c *cli.Context) error {
	environments, err := api.ListEnvironments()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list environments: %s", err.Error()), nil)
	}

	if len(environments) == 0 {
		utils.PrintInfo("No environments found")
		return nil
	}

	if utils.TryPrintJSON(environments) {
		return nil
	}

	headers := []string{"NAME", "ENV ID", "DEPLOYMENT ID", "CREATED"}
	rows := make([][]string, 0, len(environments))
	for _, env := range environments {
		rows = append(rows, []string{
			env.AppLabel,
			env.EnvironmentID.String(),
			env.DeploymentID.String(),
			api.FormatTimeAgo(env.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleEnvUnset(c *cli.Context) error {
	key := c.String("key")

	deploymentID, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to resolve deployment: %s", err.Error()), nil)
	}

	envs, err := api.GetEnvironmentsByDeploymentID(deploymentID)
	if err != nil || len(envs) == 0 {
		return utils.NewError("no environment found for this deployment", nil)
	}

	if err := api.UnsetEnvironmentKey(envs[0].EnvironmentID.String(), key); err != nil {
		return utils.NewError(fmt.Sprintf("failed to unset key: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Key %q removed from environment", key)
	return nil
}

