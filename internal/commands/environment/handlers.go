package environment

import (
	"context"
	"fmt"
	"strings"

	"1ctl/internal/api"
	"1ctl/internal/deploy"
	"1ctl/internal/utils"

	"github.com/google/uuid"
)

// --- Handlers -----------------------------------------------------------

func handleCreateEnvironment(ctx context.Context, in envCreateInput) error {
	deploymentIDStr, err := deploy.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}

	deploymentID, err := uuid.Parse(deploymentIDStr)
	if err != nil {
		return utils.NewError(fmt.Sprintf("invalid deployment-id: %s", err.Error()), nil)
	}

	keyValues := make([]api.KeyValuePair, 0, len(in.Env))
	for _, env := range in.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return utils.NewError("invalid environment variable format", nil)
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

func handleListEnvironments(ctx context.Context, in envListInput) error {
	environments, err := api.ListEnvironments()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list environments: %s", err.Error()), nil)
	}

	// Filter by --app or --deployment-id if provided
	if in.App != "" || in.DeploymentID != "" {
		depID, err := deploy.ResolveDeploymentID(in.DeploymentID, in.App, "")
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to resolve deployment: %s", err.Error()), nil)
		}
		var filtered []api.Environment
		for _, env := range environments {
			if env.DeploymentID.String() == depID {
				filtered = append(filtered, env)
			}
		}
		environments = filtered
	}

	if utils.PrintListOrJSON(environments, "No environments found") {
		return nil
	}

	headers := []string{"NAME", "ENV ID", "DEPLOYMENT ID", "CREATED"}
	rows := make([][]string, 0, len(environments))
	for _, env := range environments {
		rows = append(rows, []string{
			env.AppLabel,
			env.EnvironmentID.String(),
			env.DeploymentID.String(),
			utils.FormatTimeAgo(env.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleEnvUnset(ctx context.Context, in envUnsetInput) error {
	deploymentID, err := deploy.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to resolve deployment: %s", err.Error()), nil)
	}

	envs, err := api.GetEnvironmentsByDeploymentID(deploymentID)
	if err != nil || len(envs) == 0 {
		return utils.NewError("no environment found for this deployment", nil)
	}

	if err := api.UnsetEnvironmentKey(envs[0].EnvironmentID.String(), in.Key); err != nil {
		return utils.NewError(fmt.Sprintf("failed to unset key: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Key %q removed from environment", in.Key)
	return nil
}
