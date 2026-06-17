package commands

import (
	"context"
	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v3"
)

func TokenCommand() *cli.Command {
	return &cli.Command{
		Name:    "token",
		Aliases: []string{"api-token"},
		Usage:   "Manage API tokens",
		Commands: []*cli.Command{
			tokenListCommand(),
			tokenCreateCommand(),
			tokenGetCommand(),
			tokenEnableCommand(),
			tokenDisableCommand(),
			tokenDeleteCommand(),
		},
	}
}

func tokenListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List API tokens",
		Action: handleTokenList,
	}
}

func tokenCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new API token",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Usage:    "Token name",
				Required: true,
			},
			&cli.IntFlag{
				Name:  "expires",
				Usage: "Token expiry in days (0 for no expiry)",
				Value: 0,
			},
		},
		Action: handleTokenCreate,
	}
}

func tokenGetCommand() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Get token details",
		ArgsUsage: "<token-id>",
		Action:    handleTokenGet,
	}
}

func tokenEnableCommand() *cli.Command {
	return &cli.Command{
		Name:      "enable",
		Usage:     "Enable a token",
		ArgsUsage: "<token-id>",
		Action:    handleTokenEnable,
	}
}

func tokenDisableCommand() *cli.Command {
	return &cli.Command{
		Name:      "disable",
		Usage:     "Disable a token",
		ArgsUsage: "<token-id>",
		Action:    handleTokenDisable,
	}
}

func tokenDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a token",
		ArgsUsage: "<token-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation prompt",
			},
		},
		Action: handleTokenDelete,
	}
}

func handleTokenList(ctx context.Context, cmd *cli.Command) error {
	userID := satuskyctx.GetUserID()
	orgID := satuskyctx.GetCurrentOrgID()
	if userID == "" || orgID == "" {
		return utils.NewError("user or organization ID not found. Please run '1ctl auth login' first", nil)
	}

	tokens, err := api.GetCLITokens(userID, orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list tokens: %s", err.Error()), nil)
	}

	if len(tokens) == 0 {
		utils.PrintInfo("No API tokens found")
		return nil
	}

	if utils.TryPrintJSON(tokens) {
		return nil
	}

	utils.PrintHeader("API Tokens")
	for _, token := range tokens {
		status := "Enabled"
		if !token.IsActive {
			status = "Disabled"
		}
		utils.PrintStatusLine("ID", token.ID.String())
		utils.PrintStatusLine("Name", token.Name)
		utils.PrintStatusLine("Status", status)
		if token.LastUsedAt != nil {
			utils.PrintStatusLine("Last Used", formatTimeAgo(*token.LastUsedAt))
		} else {
			utils.PrintStatusLine("Last Used", "Never")
		}
		if token.ExpiresAt != nil {
			utils.PrintStatusLine("Expires", token.ExpiresAt.Format("2006-01-02"))
		} else {
			utils.PrintStatusLine("Expires", "Never")
		}
		utils.PrintStatusLine("Created", formatTimeAgo(token.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleTokenCreate(ctx context.Context, cmd *cli.Command) error {
	userID := satuskyctx.GetUserID()
	orgID := satuskyctx.GetCurrentOrgID()
	if userID == "" || orgID == "" {
		return utils.NewError("user or organization ID not found. Please run '1ctl auth login' first", nil)
	}

	name := cmd.String("name")
	expires := cmd.Int("expires")

	req := api.CreateTokenRequest{
		Name:      name,
		ExpiresIn: expires,
	}

	token, err := api.CreateCLIToken(userID, orgID, req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create token: %s", err.Error()), nil)
	}

	utils.PrintSuccess("API token created successfully")
	utils.PrintStatusLine("ID", token.ID.String())
	utils.PrintStatusLine("Name", token.Name)
	if token.Token != "" {
		fmt.Println()
		utils.PrintWarning("IMPORTANT: Save this token now. You won't be able to see it again!")
		utils.PrintStatusLine("Token", token.Token)
	}
	if token.ExpiresAt != nil {
		utils.PrintStatusLine("Expires", token.ExpiresAt.Format("2006-01-02"))
	}
	return nil
}

func handleTokenGet(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("token ID is required", nil)
	}

	userID := satuskyctx.GetUserID()
	if userID == "" {
		return utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
	}

	tokenID := cmd.Args().First()

	token, err := api.GetCLIToken(userID, tokenID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get token: %s", err.Error()), nil)
	}

	status := "Enabled"
	if !token.IsActive {
		status = "Disabled"
	}

	utils.PrintHeader("API Token Details")
	utils.PrintStatusLine("ID", token.ID.String())
	utils.PrintStatusLine("Name", token.Name)
	utils.PrintStatusLine("Status", status)
	if token.LastUsedAt != nil {
		utils.PrintStatusLine("Last Used", formatTimeAgo(*token.LastUsedAt))
	} else {
		utils.PrintStatusLine("Last Used", "Never")
	}
	if token.ExpiresAt != nil {
		utils.PrintStatusLine("Expires", token.ExpiresAt.Format("2006-01-02"))
	} else {
		utils.PrintStatusLine("Expires", "Never")
	}
	utils.PrintStatusLine("Created", formatTimeAgo(token.CreatedAt))
	return nil
}

func handleTokenEnable(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("token ID is required", nil)
	}

	userID := satuskyctx.GetUserID()
	if userID == "" {
		return utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
	}

	tokenID := cmd.Args().First()

	if err := api.SetCLITokenState(userID, tokenID, true); err != nil {
		return utils.NewError(fmt.Sprintf("failed to enable token: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Token enabled successfully")
	return nil
}

func handleTokenDisable(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("token ID is required", nil)
	}

	userID := satuskyctx.GetUserID()
	if userID == "" {
		return utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
	}

	tokenID := cmd.Args().First()

	if err := api.SetCLITokenState(userID, tokenID, false); err != nil {
		return utils.NewError(fmt.Sprintf("failed to disable token: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Token disabled successfully")
	return nil
}

func handleTokenDelete(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("token ID is required", nil)
	}

	userID := satuskyctx.GetUserID()
	orgID := satuskyctx.GetCurrentOrgID()
	if userID == "" || orgID == "" {
		return utils.NewError("user or organization ID not found. Please run '1ctl auth login' first", nil)
	}

	tokenID := cmd.Args().First()

	if !utils.Confirm(fmt.Sprintf("Delete token %s? This cannot be undone.", tokenID), cmd.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := api.DeleteCLIToken(userID, orgID, tokenID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete token: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Token deleted successfully")
	return nil
}
