package commands

import (
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/deploy"
	"1ctl/internal/utils"
)

// resolveDeploymentID returns the deployment ID to use for a command.
// Delegates to the shared implementation in the deploy package.
func resolveDeploymentID(depIDFlag, appFlag, configArg string) (string, error) {
	return deploy.ResolveDeploymentID(depIDFlag, appFlag, configArg)
}

// requireUserContext returns the userID from context or an error.
// Used by machine.go (legacy) and the logs sub-package.
func requireUserContext() (string, error) {
	userID := satuskyctx.GetUserID()
	if userID == "" {
		return "", utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
	}
	return userID, nil
}
