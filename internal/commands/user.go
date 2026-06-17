package commands

import (
	"context"
	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func UserCommand() *cli.Command {
	return &cli.Command{
		Name:  "user",
		Usage: "Manage user account",
		Commands: []*cli.Command{
			userMeCommand(),
			userUpdateCommand(),
			userPasswordCommand(),
			userPermissionsCommand(),
			userSessionsCommand(),
		},
	}
}

func userMeCommand() *cli.Command {
	return &cli.Command{
		Name:   "me",
		Usage:  "Show current user profile",
		Action: handleUserMe,
	}
}

func userUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update user profile",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "name",
				Usage: "New display name",
			},
			&cli.StringFlag{
				Name:  "email",
				Usage: "New email address",
			},
		},
		Action: handleUserUpdate,
	}
}

func userPasswordCommand() *cli.Command {
	return &cli.Command{
		Name:   "password",
		Usage:  "Change password (interactive)",
		Action: handleUserPassword,
	}
}

func userPermissionsCommand() *cli.Command {
	return &cli.Command{
		Name:   "permissions",
		Usage:  "Show current permissions",
		Action: handleUserPermissions,
	}
}

func userSessionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "sessions",
		Usage: "Manage sessions",
		Commands: []*cli.Command{
			{
				Name:   "revoke",
				Usage:  "Revoke all sessions",
				Action: handleUserSessionsRevoke,
			},
		},
	}
}

func handleUserMe(ctx context.Context, cmd *cli.Command) error {
	user, err := api.GetCurrentUser()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get user profile: %s", err.Error()), nil)
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
	utils.PrintStatusLine("Created", formatTimeAgo(user.CreatedAt))
	return nil
}

func handleUserUpdate(ctx context.Context, cmd *cli.Command) error {
	name := cmd.String("name")
	email := cmd.String("email")

	if name == "" && email == "" {
		return utils.NewError("at least one of --name or --email is required", nil)
	}

	user, err := api.GetCurrentUser()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get current user: %s", err.Error()), nil)
	}

	req := api.UpdateUserRequest{
		Name:  name,
		Email: email,
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

func handleUserPassword(ctx context.Context, cmd *cli.Command) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Current password: ")
	currentPasswordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		// Fallback to regular input if terminal password reading fails
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

func handleUserPermissions(ctx context.Context, cmd *cli.Command) error {
	orgID := satuskyctx.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	permsList, err := api.GetUserPermissions(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get permissions: %s", err.Error()), nil)
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

func handleUserSessionsRevoke(ctx context.Context, cmd *cli.Command) error {
	if err := api.RevokeAllSessions(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to revoke sessions: %s", err.Error()), nil)
	}

	utils.PrintSuccess("All sessions have been revoked")
	utils.PrintInfo("You will need to log in again")
	return nil
}
