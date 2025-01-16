package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"strings"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func SecretCommand() *cli.Command {
	return &cli.Command{
		Name:  "secret",
		Usage: "Manage secrets",
		Subcommands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Create a new secret",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "deployment-id",
						Usage:    "Deployment ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "name",
						Usage:    "Secret name",
						Required: true,
					},
					&cli.StringSliceFlag{
						Name:  "env",
						Usage: "Environment variables (format: KEY=VALUE)",
					},
				},
				Action: handleCreateSecret,
			},
			{
				Name:   "list",
				Usage:  "List all secrets",
				Action: handleListSecrets,
			},
		},
	}
}

func handleCreateSecret(c *cli.Context) error {
	envVars := c.StringSlice("env")
	keyValues := make([]api.KeyValuePair, 0, len(envVars))

	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return utils.NewError("invalid environment variable format: %s", nil)
		}
		keyValues = append(keyValues, api.KeyValuePair{
			Key:   parts[0],
			Value: parts[1],
		})
	}

	secret := api.Secret{
		DeploymentID: uuid.MustParse(c.String("deployment-id")),
		AppLabel:     c.String("name"),
		Namespace:    context.GetCurrentNamespace(),
		KeyValues:    keyValues,
	}

	secretResp, err := api.CreateSecret(secret)
	if err != nil {
		return utils.NewError("failed to create secret: %w", err)
	}

	utils.PrintSuccess("Secret %s created successfully\n", secretResp.AppLabel)
	return nil
}

func handleListSecrets(c *cli.Context) error {
	secrets, err := api.ListSecrets()
	if err != nil {
		return utils.NewError("failed to list secrets: %w", err)
	}

	if len(secrets) == 0 {
		utils.PrintInfo("No secrets found")
		return nil
	}

	utils.PrintHeader("Secrets")
	for _, secret := range secrets {
		utils.PrintStatusLine("Secret", secret.SecretID.String())
		utils.PrintStatusLine("Deployment ID", secret.DeploymentID.String())
		utils.PrintStatusLine("Namespace", secret.Namespace)
		utils.PrintStatusLine("App Label", secret.AppLabel)
		if len(secret.KeyValues) > 0 {
			utils.PrintInfo("Key-Value Pairs:\n")
			for _, kv := range secret.KeyValues {
				utils.PrintInfo("  %s: %s\n", kv.Key, kv.Value)
			}
		}
		utils.PrintStatusLine("Created", api.FormatTimeAgo(secret.CreatedAt))
		utils.PrintStatusLine("Last Updated", api.FormatTimeAgo(secret.UpdatedAt))
		utils.PrintDivider()
	}
	return nil
}
