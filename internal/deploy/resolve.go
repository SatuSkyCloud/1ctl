// Package deploy provides deployment orchestration and shared helpers
// for resolving deployment IDs from various inputs.
package deploy

import (
	"fmt"

	"1ctl/internal/api"
	"1ctl/internal/config"
	"1ctl/internal/context"
)

// ResolveDeploymentID returns the deployment ID to use for a command.
// Precedence: explicit depID > appFlag > lookup by app name from config file.
// Mirrors the original resolveDeploymentID in internal/commands/resolve.go
// but exported so sub-package commands can use it without circular imports.
func ResolveDeploymentID(depID, appFlag, configArg string) (string, error) {
	if depID != "" {
		return depID, nil
	}

	if appFlag != "" {
		ns := context.GetCurrentNamespace()
		if ns == "" {
			return "", fmt.Errorf("not authenticated — run '1ctl auth login' first")
		}
		dep, err := api.GetDeploymentByAppLabel(ns, appFlag)
		if err != nil {
			return "", fmt.Errorf("app %q not found in organization %s\nRun '1ctl app list' to see deployed apps", appFlag, ns)
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
		return "", fmt.Errorf("app %q not found in organization %s\nRun '1ctl deploy' first or pass --deployment-id\nRun '1ctl app list' to see deployed apps", cfg.App.Name, ns)
	}

	return dep.DeploymentID.String(), nil
}
