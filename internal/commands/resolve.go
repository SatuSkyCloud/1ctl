package commands

import (
	"fmt"

	"1ctl/internal/api"
	"1ctl/internal/config"
	"1ctl/internal/context"
)

// resolveDeploymentID returns the deployment ID to use for a command.
// Precedence: explicit --deployment-id flag > --app flag > lookup by app name from satusky.toml.
// When using the config path, derives namespace from the active auth context and
// calls the backend to resolve the name → ID. This mirrors how Fly.io resolves
// app names from fly.toml without storing generated IDs in the config file.
func resolveDeploymentID(depIDFlag, appFlag, configArg string) (string, error) {
	if depIDFlag != "" {
		return depIDFlag, nil
	}

	if appFlag != "" {
		ns := context.GetCurrentNamespace()
		if ns == "" {
			return "", fmt.Errorf("not authenticated — run '1ctl auth login' first")
		}
		dep, err := api.GetDeploymentByAppLabel(ns, appFlag)
		if err != nil {
			return "", fmt.Errorf("app %q not found in namespace %s", appFlag, ns)
		}
		return dep.DeploymentID.String(), nil
	}

	cfg, err := config.FindConfig(configArg)
	if err != nil {
		return "", fmt.Errorf("no --deployment-id and config file error: %w", err)
	}
	if cfg == nil {
		hint := ""
		if configArg == "" {
			hint = "\nRun '1ctl deploy' in your project directory or pass --deployment-id"
		}
		return "", fmt.Errorf("no --deployment-id and no satusky.toml found%s", hint)
	}
	if cfg.App.Name == "" {
		return "", fmt.Errorf("satusky.toml is missing 'name' — add:\n  [app]\n  name = \"your-app\"")
	}

	ns := context.GetCurrentNamespace()
	if ns == "" {
		return "", fmt.Errorf("not authenticated — run '1ctl auth login' first")
	}

	dep, err := api.GetDeploymentByAppLabel(ns, cfg.App.Name)
	if err != nil {
		return "", fmt.Errorf("app %q not found in namespace %s\nRun '1ctl deploy' first or pass --deployment-id", cfg.App.Name, ns)
	}

	return dep.DeploymentID.String(), nil
}
