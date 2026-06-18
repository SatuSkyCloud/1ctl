package commands

import (
	"context"
	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func SecretCommand() *cli.Command {
	return &cli.Command{
		Name:  "secret",
		Usage: "Manage secrets",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Create a new secret",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "deployment-id",
						Usage: "Deployment ID",
					},
					&cli.StringFlag{
						Name:  "app",
						Usage: "App name to resolve (alternative to --deployment-id)",
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
				Name:      "get",
				Usage:     "Show secret details including key names",
				ArgsUsage: "<secret-id>",
				Action:    handleGetSecret,
			},
			{
				Name:  "unset",
				Usage: "Remove a specific key from a secret",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "app", Usage: "App name to resolve (alternative to --deployment-id)"},
					&cli.StringFlag{Name: "config", Usage: "Config name or path"},
					&cli.StringFlag{Name: "deployment-id", Aliases: []string{"d"}, Usage: "Deployment ID"},
					&cli.StringFlag{Name: "key", Aliases: []string{"k"}, Usage: "Key to remove", Required: true},
				},
				Action: handleSecretUnset,
			},
		},
	}
}

func handleCreateSecret(ctx context.Context, cmd *cli.Command) error {
	deploymentIDStr, err := resolveDeploymentID(cmd.String("deployment-id"), cmd.String("app"), cmd.String("config"))
	if err != nil {
		return err
	}

	deploymentID, err := uuid.Parse(deploymentIDStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	envVars := cmd.StringSlice("kv")
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

	appLabel := cmd.String("name")
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
		Namespace:    satuskyctx.GetCurrentNamespace(),
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

	// Auto-restart the deployment so new secrets are picked up immediately.
	// This follows Kubernetes best practice: kubectl create secret + kubectl rollout restart.
	utils.PrintInfo("Restarting deployment to activate secrets...")
	if err := api.RestartDeployment(deploymentIDStr); err != nil {
		utils.PrintWarning("Secret created, but restart failed: %s", err.Error())
		utils.PrintInfo("Run: 1ctl deploy restart --app %s", displayName)
	} else {
		utils.PrintSuccess("Deployment restarting — secrets will be available shortly")
	}
	return nil
}

func handleListSecrets(ctx context.Context, cmd *cli.Command) error {
	secrets, err := api.ListSecrets()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list secrets: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(secrets, "No secrets found") {
		return nil
	}

	headers := []string{"NAME", "SECRET ID", "DEPLOYMENT ID", "KEYS", "CREATED"}
	rows := make([][]string, 0, len(secrets))
	for _, secret := range secrets {
		rows = append(rows, []string{
			secret.AppLabel,
			secret.SecretID.String(),
			secret.DeploymentID.String(),
			fmt.Sprintf("%d", len(secret.KeyValues)),
			api.FormatTimeAgo(secret.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleSecretUnset(ctx context.Context, cmd *cli.Command) error {
	key := cmd.String("key")

	deploymentID, err := resolveDeploymentID(cmd.String("deployment-id"), cmd.String("app"), cmd.String("config"))
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

func handleGetSecret(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("secret ID is required. Usage: 1ctl secret get <secret-id>", nil)
	}
	secretID := cmd.Args().First()

	secrets, err := api.ListSecrets()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to fetch secrets: %%s", err.Error()), nil)
	}

	for _, s := range secrets {
		if s.SecretID.String() == secretID {
			if utils.TryPrintJSON(s) {
				return nil
			}
			utils.PrintHeader("Secret %s", s.AppLabel)
			utils.PrintStatusLine("Secret ID", s.SecretID.String())
			utils.PrintStatusLine("Deployment ID", s.DeploymentID.String())
			utils.PrintStatusLine("Namespace", s.Namespace)
			utils.PrintStatusLine("Created", api.FormatTimeAgo(s.CreatedAt))
			utils.PrintStatusLine("Keys", fmt.Sprintf("%d", len(s.KeyValues)))
			for _, kv := range s.KeyValues {
				utils.PrintStatusLine("  "+kv.Key, "********")
			}
			return nil
		}
	}
	return utils.NewError(fmt.Sprintf("secret %%s not found", secretID), nil)
}