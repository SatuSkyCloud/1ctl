package secret

import (
	"context"
	"fmt"
	"strings"

	"1ctl/internal/api"
	"1ctl/internal/deploy"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"

	"github.com/google/uuid"
)

// --- Handlers -----------------------------------------------------------

func handleCreateSecret(ctx context.Context, in secretCreateInput) error {
	deploymentIDStr, err := deploy.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}

	deploymentID, err := uuid.Parse(deploymentIDStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	keyValues := make([]api.KeyValuePair, 0, len(in.KV))
	for _, kv := range in.KV {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return utils.NewError("invalid key-value format (expected KEY=VALUE)", nil)
		}
		keyValues = append(keyValues, api.KeyValuePair{
			Key:   parts[0],
			Value: parts[1],
		})
	}

	appLabel := in.Name
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

	utils.PrintInfo("Restarting deployment to activate secrets...")
	if err := api.RestartDeployment(deploymentIDStr); err != nil {
		utils.PrintWarning("Secret created, but restart failed: %s", err.Error())
		utils.PrintInfo("Run: 1ctl deploy restart --app %s", displayName)
	} else {
		utils.PrintSuccess("Deployment restarting — secrets will be available shortly")
	}
	return nil
}

func handleListSecrets(ctx context.Context) error {
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
			utils.FormatTimeAgo(secret.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleSecretUnset(ctx context.Context, in secretUnsetInput) error {
	deploymentID, err := deploy.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to resolve deployment: %s", err.Error()), nil)
	}

	secrets, err := api.GetSecretsByDeploymentID(deploymentID)
	if err != nil || len(secrets) == 0 {
		return utils.NewError("no secret found for this deployment", nil)
	}

	if err := api.UnsetSecretKey(secrets[0].SecretID.String(), in.Key); err != nil {
		return utils.NewError(fmt.Sprintf("failed to unset key: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Key %q removed from secrets", in.Key)
	return nil
}

func handleGetSecret(ctx context.Context, in secretGetInput) error {
	secrets, err := api.ListSecrets()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to fetch secrets: %s", err.Error()), nil)
	}

	for _, s := range secrets {
		if s.SecretID.String() == in.SecretID {
			if utils.TryPrintJSON(s) {
				return nil
			}
			utils.PrintHeader("Secret %s", s.AppLabel)
			utils.PrintStatusLine("Secret ID", s.SecretID.String())
			utils.PrintStatusLine("Deployment ID", s.DeploymentID.String())
			utils.PrintStatusLine("Namespace", s.Namespace)
			utils.PrintStatusLine("Created", utils.FormatTimeAgo(s.CreatedAt))
			utils.PrintStatusLine("Keys", fmt.Sprintf("%d", len(s.KeyValues)))
			for _, kv := range s.KeyValues {
				utils.PrintStatusLine("  "+kv.Key, "********")
			}
			return nil
		}
	}
	return utils.NewError(fmt.Sprintf("secret %s not found", in.SecretID), nil)
}
