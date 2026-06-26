package token

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

func handleTokenList(ctx context.Context) error {
	userID := satuskyctx.GetUserID()
	orgID := satuskyctx.GetCurrentOrgID()
	if userID == "" || orgID == "" {
		return utils.NewError("user or organization ID not found. Please run '1ctl auth login' first", nil)
	}

	tokens, err := api.GetCLITokens(userID, orgID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list tokens: %s", err.Error()), nil)
	}

	// Redact tokens in JSON output — never expose full JWTs in machine-readable output
	if utils.IsJSONOutput() {
		for i := range tokens {
			if len(tokens[i].Token) > 20 {
				tokens[i].Token = tokens[i].Token[:10] + "..." + tokens[i].Token[len(tokens[i].Token)-5:]
			}
		}
	}

	if utils.PrintListOrJSON(tokens, "No API tokens found") {
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
			utils.PrintStatusLine("Last Used", utils.FormatTimeAgo(*token.LastUsedAt))
		} else {
			utils.PrintStatusLine("Last Used", "Never")
		}
		if token.ExpiresAt != nil {
			utils.PrintStatusLine("Expires", token.ExpiresAt.Format("2006-01-02"))
		} else {
			utils.PrintStatusLine("Expires", "Never")
		}
		utils.PrintStatusLine("Created", utils.FormatTimeAgo(token.CreatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleTokenCreate(ctx context.Context, in tokenCreateInput) error {
	userID := satuskyctx.GetUserID()
	orgID := satuskyctx.GetCurrentOrgID()
	if userID == "" || orgID == "" {
		return utils.NewError("user or organization ID not found. Please run '1ctl auth login' first", nil)
	}

	req := api.CreateTokenRequest{
		Name:      in.Name,
		ExpiresIn: in.Expires,
	}

	token, err := api.CreateCLIToken(userID, orgID, req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create token: %s", err.Error()), nil)
	}

	// Redact token in JSON output — never expose full JWTs in machine-readable output
	if utils.IsJSONOutput() && token != nil && len(token.Token) > 20 {
		token.Token = token.Token[:10] + "..." + token.Token[len(token.Token)-5:]
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

func handleTokenGet(ctx context.Context, in tokenIDInput) error {
	userID := satuskyctx.GetUserID()
	if userID == "" {
		return utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
	}

	token, err := api.GetCLIToken(userID, in.TokenID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get token: %s", err.Error()), nil)
	}

	// Redact token in JSON output — never expose full JWTs in machine-readable output
	if utils.IsJSONOutput() && token != nil && len(token.Token) > 20 {
		token.Token = token.Token[:10] + "..." + token.Token[len(token.Token)-5:]
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
		utils.PrintStatusLine("Last Used", utils.FormatTimeAgo(*token.LastUsedAt))
	} else {
		utils.PrintStatusLine("Last Used", "Never")
	}
	if token.ExpiresAt != nil {
		utils.PrintStatusLine("Expires", token.ExpiresAt.Format("2006-01-02"))
	} else {
		utils.PrintStatusLine("Expires", "Never")
	}
	utils.PrintStatusLine("Created", utils.FormatTimeAgo(token.CreatedAt))
	return nil
}

func handleTokenEnable(ctx context.Context, in tokenIDInput) error {
	userID := satuskyctx.GetUserID()
	if userID == "" {
		return utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
	}

	if err := api.SetCLITokenState(userID, in.TokenID, true); err != nil {
		return utils.NewError(fmt.Sprintf("failed to enable token: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Token enabled successfully")
	return nil
}

func handleTokenDisable(ctx context.Context, in tokenIDInput) error {
	userID := satuskyctx.GetUserID()
	if userID == "" {
		return utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
	}

	if err := api.SetCLITokenState(userID, in.TokenID, false); err != nil {
		return utils.NewError(fmt.Sprintf("failed to disable token: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Token disabled successfully")
	return nil
}

func handleTokenDelete(ctx context.Context, in tokenDeleteInput) error {
	userID := satuskyctx.GetUserID()
	orgID := satuskyctx.GetCurrentOrgID()
	if userID == "" || orgID == "" {
		return utils.NewError("user or organization ID not found. Please run '1ctl auth login' first", nil)
	}

	if !utils.Confirm(fmt.Sprintf("Delete token %s? This cannot be undone.", in.TokenID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := api.DeleteCLIToken(userID, orgID, in.TokenID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete token: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Token deleted successfully")
	return nil
}
