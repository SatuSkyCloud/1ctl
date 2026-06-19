package profile

import (
	"context"
	"fmt"

	"1ctl/internal/config"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

func defaultAPIURLDisplay() string {
	return fmt.Sprintf("(default: %s)", config.DefaultAPIURL())
}

func handleProfileList(ctx context.Context) error {
	profiles, err := satuskyctx.ListProfiles()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list profiles: %s", err.Error()), nil)
	}

	if len(profiles) == 0 {
		utils.PrintInfo("No profiles yet.")
		utils.PrintInfo("Create one with: 1ctl profile create --name <name> [--url <api-url>]")
		return nil
	}

	utils.PrintHeader("Profiles")
	for _, p := range profiles {
		marker := "  "
		if p.IsActive {
			marker = "* "
		}

		apiURL := p.APIURL
		if apiURL == "" {
			apiURL = defaultAPIURLDisplay()
		}

		fmt.Printf("%s%s\n", marker, p.Name)
		utils.PrintStatusLine("  API URL", apiURL)
		if p.Email != "" {
			utils.PrintStatusLine("  Auth", p.Email)
		} else {
			utils.PrintStatusLine("  Auth", "(not logged in)")
		}
		if p.OrgName != "" {
			utils.PrintStatusLine("  Org", p.OrgName)
		}
		utils.PrintDivider()
	}
	return nil
}

func handleProfileCreate(ctx context.Context, in profileCreateInput) error {
	if err := satuskyctx.CreateProfile(in.Name, in.URL); err != nil {
		return utils.NewError(fmt.Sprintf("failed to create profile: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Profile '%s' created", in.Name)
	if in.URL != "" {
		utils.PrintStatusLine("API URL", in.URL)
	} else {
		utils.PrintStatusLine("API URL", defaultAPIURLDisplay())
	}
	utils.PrintInfo("Next steps:")
	utils.PrintInfo("  1ctl profile use %s", in.Name)
	utils.PrintInfo("  1ctl auth login --token=<your-token>")
	return nil
}

func handleProfileUse(ctx context.Context, in profileNameInput) error {
	if err := satuskyctx.UseProfile(in.Name); err != nil {
		return utils.NewError(fmt.Sprintf("failed to switch profile: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Switched to profile '%s'", in.Name)
	utils.PrintStatusLine("API URL", config.GetConfig().ApiURL)

	if satuskyctx.GetToken() == "" {
		utils.PrintInfo("Run '1ctl auth login --token=<token>' to authenticate this profile")
	}

	return nil
}

func handleProfileCurrent(ctx context.Context) error {
	name := satuskyctx.GetActiveProfileName()

	if name == "" {
		utils.PrintInfo("No profile active. Run '1ctl profile use <name>' to select one.")
		return nil
	}

	utils.PrintHeader("Active Profile")
	utils.PrintStatusLine("Profile", name)
	utils.PrintStatusLine("API URL", config.GetConfig().ApiURL)

	if email := satuskyctx.GetEmail(); email != "" {
		utils.PrintStatusLine("Auth", email)
	} else {
		utils.PrintStatusLine("Auth", "(not logged in — run '1ctl auth login')")
	}

	if org := satuskyctx.GetCurrentOrgName(); org != "" {
		utils.PrintStatusLine("Org", org)
	}

	return nil
}

func handleProfileDelete(ctx context.Context, in profileNameInput) error {
	if err := satuskyctx.DeleteProfile(in.Name); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete profile: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Profile '%s' deleted", in.Name)
	return nil
}
