package secret

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/deploy"
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

	// Collect key-value pairs from BOTH --kv/--env flags AND positional args.
	// This matches Fly.io's `fly secrets set KEY=VALUE` convention where
	// secrets are passed as positional arguments, not flags.
	allKV := append(in.KV, in.Args...)
	if in.FromFile != "" {
		fileKV, err := readSecretPairs(in.FromFile)
		if err != nil {
			return err
		}
		allKV = append(allKV, fileKV...)
	}

	if len(allKV) == 0 {
		return utils.NewError("at least one KEY=VALUE pair is required", nil)
	}

	keyValues := make([]api.KeyValuePair, 0, len(allKV))
	for _, kv := range allKV {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return utils.NewError(fmt.Sprintf("invalid key-value format (expected KEY=VALUE): %s", kv), nil)
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
	utils.PrintInfo("The deployment will restart after the secret is projected safely.")
	return nil
}

func readSecretPairs(path string) ([]string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to inspect secret file: %s", err.Error()), nil)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, utils.NewError("secret file must not be a symbolic link", nil)
	}
	if !info.Mode().IsRegular() {
		return nil, utils.NewError("secret file must be a regular file", nil)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return nil, utils.NewError("secret file must be owner-only (mode 0600)", nil)
	}

	file, err := os.Open(path) // #nosec G304 -- explicit user-provided secret path
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to open secret file: %s", err.Error()), nil)
	}
	defer file.Close() //nolint:errcheck // read-only close error is not actionable
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to inspect opened secret file: %s", err.Error()), nil)
	}
	if !os.SameFile(info, openedInfo) {
		return nil, utils.NewError("secret file changed while it was being opened", nil)
	}

	var pairs []string
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSuffix(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, utils.NewError(
				fmt.Sprintf("invalid secret file line %d: expected KEY=VALUE", lineNumber),
				nil,
			)
		}
		if strings.TrimSpace(parts[0]) == "" {
			return nil, utils.NewError(
				fmt.Sprintf("invalid secret file line %d: key must not be empty", lineNumber),
				nil,
			)
		}
		pairs = append(pairs, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to read secret file: %s", err.Error()), nil)
	}
	return pairs, nil
}

func handleListSecrets(ctx context.Context, in secretListInput) error {
	secrets, err := api.ListSecrets()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list secrets: %s", err.Error()), nil)
	}

	// Filter by app name if requested
	if in.App != "" {
		var filtered []api.Secret
		for _, s := range secrets {
			if s.AppLabel == in.App {
				filtered = append(filtered, s)
			}
		}
		secrets = filtered
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
			fmt.Sprintf("%d", secretMetadataKeyCount(secret)),
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
	// --- Path 1: Lookup by --id (escape hatch) ---
	if in.ID != "" {
		secrets, err := api.ListSecrets()
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to fetch secrets: %s", err.Error()), nil)
		}
		for _, s := range secrets {
			if s.SecretID.String() == in.ID {
				return printSecretBundle(&s)
			}
		}
		return utils.NewError(fmt.Sprintf("secret %s not found", in.ID), nil)
	}

	// --- Path 2: Lookup by --app [key] ---
	deploymentID, err := deploy.ResolveDeploymentID("", in.App, "")
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to resolve app: %s", err.Error()), nil)
	}

	secrets, err := api.GetSecretsByDeploymentID(deploymentID)
	if err != nil || len(secrets) == 0 {
		return utils.NewError(fmt.Sprintf("no secrets found for app %q", in.App), nil)
	}
	secret := secrets[0] // deployment-id is unique; there should be one secret bundle

	if in.Key != "" {
		// Show just the specified key
		for _, key := range secret.Keys {
			if key == in.Key {
				if utils.TryPrintJSON(map[string]interface{}{"key": key, "exists": true}) {
					return nil
				}
				utils.PrintHeader("Secret %s", in.Key)
				utils.PrintStatusLine("App", in.App)
				utils.PrintStatusLine("Value", "********")
				utils.PrintStatusLine("Created", utils.FormatTimeAgo(secret.CreatedAt))
				return nil
			}
		}
		return utils.NewError(fmt.Sprintf("key %q not found in secrets for app %q", in.Key, in.App), nil)
	}

	// Show the full bundle metadata (no key specified)
	return printSecretBundle(&secret)
}

func printSecretBundle(s *api.Secret) error {
	if s == nil {
		return nil
	}
	if utils.TryPrintJSON(s) {
		return nil
	}
	utils.PrintHeader("Secret %s", s.AppLabel)
	utils.PrintStatusLine("Secret ID", s.SecretID.String())
	utils.PrintStatusLine("Deployment ID", s.DeploymentID.String())
	utils.PrintStatusLine("Namespace", s.Namespace)
	utils.PrintStatusLine("Created", utils.FormatTimeAgo(s.CreatedAt))
	utils.PrintStatusLine("Keys", fmt.Sprintf("%d", secretMetadataKeyCount(*s)))
	for _, key := range s.Keys {
		utils.PrintStatusLine("  "+key, "********")
	}
	return nil
}

func secretMetadataKeyCount(secret api.Secret) int {
	if secret.KeyCount > 0 {
		return secret.KeyCount
	}
	return len(secret.Keys)
}
