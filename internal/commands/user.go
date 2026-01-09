package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

func UserCommand() *cli.Command {
	return &cli.Command{
		Name:    "user",
		Aliases: []string{"profile"},
		Usage:   "Manage user profile",
		Subcommands: []*cli.Command{
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
		Subcommands: []*cli.Command{
			{
				Name:   "revoke",
				Usage:  "Revoke all sessions",
				Action: handleUserSessionsRevoke,
			},
		},
	}
}

func handleUserMe(c *cli.Context) error {
	user, err := api.GetCurrentUser()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get user profile: %s", err.Error()), nil)
	}

	utils.PrintHeader("User Profile")
	utils.PrintStatusLine("ID", user.ID.String())
	utils.PrintStatusLine("Email", user.Email)
	utils.PrintStatusLine("Name", user.Name)
	if user.AvatarURL != "" {
		utils.PrintStatusLine("Avatar", user.AvatarURL)
	}
	utils.PrintStatusLine("Created", formatTimeAgo(user.CreatedAt))
	utils.PrintStatusLine("Updated", formatTimeAgo(user.UpdatedAt))
	return nil
}

func handleUserUpdate(c *cli.Context) error {
	name := c.String("name")
	email := c.String("email")

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

	updatedUser, err := api.UpdateUser(user.ID.String(), req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to update user: %s", err.Error()), nil)
	}

	utils.PrintSuccess("User profile updated successfully")
	utils.PrintStatusLine("Name", updatedUser.Name)
	utils.PrintStatusLine("Email", updatedUser.Email)
	return nil
}

func handleUserPassword(c *cli.Context) error {
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

func handleUserPermissions(c *cli.Context) error {
	orgID := context.GetCurrentOrgID()
	if orgID == "" {
		return utils.NewError("organization ID not found. Please run '1ctl auth login' first", nil)
	}

	perms, err := api.GetUserPermissions(orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get permissions: %s", err.Error()), nil)
	}

	utils.PrintHeader("User Permissions")
	utils.PrintStatusLine("User ID", perms.UserID.String())
	utils.PrintStatusLine("Organization ID", perms.OrganizationID.String())
	utils.PrintStatusLine("Role", perms.Role)

	if len(perms.Permissions) > 0 {
		fmt.Println()
		utils.PrintHeader("Permissions")
		for _, perm := range perms.Permissions {
			fmt.Printf("  - %s\n", perm)
		}
	}

	return nil
}

func handleUserSessionsRevoke(c *cli.Context) error {
	if err := api.RevokeAllSessions(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to revoke sessions: %s", err.Error()), nil)
	}

	utils.PrintSuccess("All sessions have been revoked")
	utils.PrintInfo("You will need to log in again")
	return nil
}
