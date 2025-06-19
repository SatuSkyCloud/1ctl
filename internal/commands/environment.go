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
						Name:     "deployment-id",
						Usage:    "Deployment ID",
						Required: true,
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
				},
				Action: handleDeleteEnvironment,
			},
		},
	}
}

func handleCreateEnvironment(c *cli.Context) error {
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
		DeploymentID: uuid.MustParse(c.String("deployment-id")),
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

	utils.PrintHeader("Environments")
	for _, env := range environments {
		utils.PrintStatusLine("Environment", env.EnvironmentID.String())
		utils.PrintStatusLine("Deployment ID", env.DeploymentID.String())
		utils.PrintStatusLine("App Label", env.AppLabel)
		if len(env.KeyValues) > 0 {
			utils.PrintInfo("Environment Variables:\n")
			for _, kv := range env.KeyValues {
				utils.PrintStatusLine(kv.Key, kv.Value)
			}
		}
		utils.PrintStatusLine("Created", api.FormatTimeAgo(env.CreatedAt))
		utils.PrintStatusLine("Last Updated", api.FormatTimeAgo(env.UpdatedAt))
		utils.PrintDivider()
	}
	return nil
}

// TODO: get data by id first before deleting to pass in the payload
func handleDeleteEnvironment(c *cli.Context) error {
	envID := c.String("env-id")
	if err := api.DeleteEnvironment(nil, envID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete environment: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Environment %s deleted successfully\n", envID)
	return nil
}
