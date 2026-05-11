package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
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
						Name:    "kv",
						Aliases: []string{"env"},
						Usage:   "Secret key-value pairs (format: KEY=VALUE)",
					},
				},
				Action: handleCreateSecret,
			},
			{
				Name:   "list",
				Usage:  "List all secrets",
				Action: handleListSecrets,
			},
			{
				Name:  "unset",
				Usage: "Remove a specific key from a secret",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "config", Usage: "Config name or path"},
					&cli.StringFlag{Name: "deployment-id", Aliases: []string{"d"}, Usage: "Deployment ID"},
					&cli.StringFlag{Name: "key", Aliases: []string{"k"}, Usage: "Key to remove", Required: true},
				},
				Action: handleSecretUnset,
			},
		},
	}
}

func handleCreateSecret(c *cli.Context) error {
	deploymentIDStr, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
	if err != nil {
		return err
	}

	deploymentID, err := uuid.Parse(deploymentIDStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	envVars := c.StringSlice("kv")
	keyValues := make([]api.KeyValuePair, 0, len(envVars))

	for _, kv := range envVars {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return utils.NewError("invalid key-value format (expected KEY=VALUE)", nil)
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

	secret := api.Secret{
		DeploymentID: deploymentID,
		AppLabel:     appLabel,
		Namespace:    context.GetCurrentNamespace(),
		KeyValues:    keyValues,
	}

	secretResp, err := api.CreateSecret(secret)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create secret: %s", err.Error()), nil)
	}

	displayName := secretResp.AppLabel
	if displayName == "" {
		displayName = appLabel
	}
	utils.PrintSuccess("Secret %s created successfully\n", displayName)
	return nil
}

func handleListSecrets(c *cli.Context) error {
	secrets, err := api.ListSecrets()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list secrets: %s", err.Error()), nil)
	}

	if len(secrets) == 0 {
		utils.PrintInfo("No secrets found")
		return nil
	}

	if utils.TryPrintJSON(secrets) {
		return nil
	}

	headers := []string{"NAME", "SECRET ID", "DEPLOYMENT ID", "CREATED"}
	rows := make([][]string, 0, len(secrets))
	for _, secret := range secrets {
		rows = append(rows, []string{
			secret.AppLabel,
			secret.SecretID.String(),
			secret.DeploymentID.String(),
			api.FormatTimeAgo(secret.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleSecretUnset(c *cli.Context) error {
	key := c.String("key")

	deploymentID, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to resolve deployment: %s", err.Error()), nil)
	}

	secrets, err := api.GetSecretsByDeploymentID(deploymentID)
	if err != nil || len(secrets) == 0 {
		return utils.NewError("no secret found for this deployment", nil)
	}

	if err := api.UnsetSecretKey(secrets[0].SecretID.String(), key); err != nil {
		return utils.NewError(fmt.Sprintf("failed to unset key: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Key %q removed from secrets", key)
	return nil
}
