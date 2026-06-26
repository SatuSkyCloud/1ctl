package user

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"

	"golang.org/x/term"
)

func handleUserMe(ctx context.Context) error {
	user, err := api.GetCurrentUser()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get user profile: %s", err.Error()), nil)
	}

	if utils.TryPrintJSON(user) {
		return nil
	}

	utils.PrintHeader("User Profile")
	utils.PrintStatusLine("ID", user.UserID)
	utils.PrintStatusLine("Email", user.Email)
	name := ""
	if user.Name != nil {
		name = *user.Name
	}
	utils.PrintStatusLine("Name", name)
	if user.Organization != "" {
		utils.PrintStatusLine("Organization", user.Organization)
	}
	if user.Role != "" {
		utils.PrintStatusLine("Role", user.Role)
	}
	utils.PrintStatusLine("Created", utils.FormatTimeAgo(user.CreatedAt))
	return nil
}

func handleUserUpdate(ctx context.Context, in userUpdateInput) error {
	if in.Name == "" && in.Email == "" {
		return utils.NewError("at least one of --name or --email is required", nil)
	}

	user, err := api.GetCurrentUser()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get current user: %s", err.Error()), nil)
	}

	req := api.UpdateUserRequest{
		Name:  in.Name,
		Email: in.Email,
	}

	updatedUser, err := api.UpdateUser(user.UserID, req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to update user: %s", err.Error()), nil)
	}

	utils.PrintSuccess("User profile updated successfully")
	updatedName := ""
	if updatedUser.Name != nil {
		updatedName = *updatedUser.Name
	}
	utils.PrintStatusLine("Name", updatedName)
	utils.PrintStatusLine("Email", updatedUser.Email)
	return nil
}

func handleUserPassword(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Current password: ")
	currentPasswordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		currentPassword, readErr := reader.ReadString('\n')
		if readErr != nil {
			return utils.NewError("failed to read current password", nil)
		}
		currentPasswordBytes = []byte(strings.TrimSpace(currentPassword))
	} else {
		fmt.Println()
	}

	fmt.Print("New password: ")
	newPasswordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		newPassword, readErr := reader.ReadString('\n')
		if readErr != nil {
			return utils.NewError("failed to read new password", nil)
		}
		newPasswordBytes = []byte(strings.TrimSpace(newPassword))
	} else {
		fmt.Println()
	}

	fmt.Print("Confirm new password: ")
	confirmPasswordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		confirmPassword, readErr := reader.ReadString('\n')
		if readErr != nil {
			return utils.NewError("failed to read password confirmation", nil)
		}
		confirmPasswordBytes = []byte(strings.TrimSpace(confirmPassword))
	} else {
		fmt.Println()
	}

	currentPassword := string(currentPasswordBytes)
	newPassword := string(newPasswordBytes)
	confirmPassword := string(confirmPasswordBytes)

	if newPassword != confirmPassword {
		return utils.NewError("new passwords do not match", nil)
	}

	if len(newPassword) < 8 {
		return utils.NewError("new password must be at least 8 characters", nil)
	}

	if err := api.ChangePassword(currentPassword, newPassword); err != nil {
		return utils.NewError(fmt.Sprintf("failed to change password: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Password changed successfully")
	return nil
}

func handleUserPermissions(ctx context.Context) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	permsList, err := api.GetUserPermissions(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get permissions: %s", err.Error()), nil)
	}

	if utils.TryPrintJSON(permsList) {
		return nil
	}

	utils.PrintHeader("User Permissions")
	if len(permsList) == 0 {
		utils.PrintInfo("No permissions assigned")
		return nil
	}
	for _, p := range permsList {
		fmt.Printf("  %-30s  %s\n", p.Name, p.Description)
	}

	return nil
}

func handleUserSessionsRevoke(ctx context.Context) error {
	if err := api.RevokeAllSessions(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to revoke sessions: %s", err.Error()), nil)
	}

	utils.PrintSuccess("All sessions have been revoked")
	utils.PrintInfo("You will need to log in again")
	return nil
}
