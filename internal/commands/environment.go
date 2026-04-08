package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/config"
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
						Name:     "name",
						Usage:    "Environment name",
						Required: true,
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
				Name:  "delete",
				Usage: "Delete an environment",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "env-id",
						Usage:    "Environment ID to delete",
						Required: true,
					},
					&cli.BoolFlag{
						Name:    "yes",
						Aliases: []string{"y"},
						Usage:   "Skip confirmation prompt",
					},
				},
				Action: handleDeleteEnvironment,
			},
		},
	}
}

func handleCreateEnvironment(c *cli.Context) error {
	deploymentIDStr, err := config.ResolveDeploymentID(c.String("deployment-id"), c.String("config"))
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

	env := api.Environment{
		DeploymentID: deploymentID,
		AppLabel:     c.String("name"),
		KeyValues:    keyValues,
	}

	envResp, err := api.UpsertEnvironment(env)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to upsert environment: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Environment %s created successfully\n", envResp.AppLabel)
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

func handleDeleteEnvironment(c *cli.Context) error {
	envID := c.String("env-id")
	if !utils.Confirm(fmt.Sprintf("Delete environment %s? This cannot be undone.", envID), c.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteEnvironment(envID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete environment: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Environment %s deleted successfully\n", envID)
	return nil
}
