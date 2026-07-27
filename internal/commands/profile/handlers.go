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

	type profileOutput struct {
		Name     string `json:"name"`
		APIURL   string `json:"api_url"`
		Email    string `json:"email,omitempty"`
		OrgName  string `json:"organization,omitempty"`
		IsActive bool   `json:"active"`
	}
	output := make([]profileOutput, 0, len(profiles))
	for _, profile := range profiles {
		output = append(output, profileOutput{
			Name:     profile.Name,
			APIURL:   profile.APIURL,
			Email:    profile.Email,
			OrgName:  profile.OrgName,
			IsActive: profile.IsActive,
		})
	}
	if utils.PrintListOrJSON(output, "No profiles yet.") {
		return nil
	}

	if len(profiles) == 0 {
		utils.PrintInfo("No profiles yet.")
		utils.PrintInfo("Create one with: 1ctl profile create <name> [--url <api-url>]")
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
		if utils.TryPrintJSON(nil) {
			return nil
		}
		utils.PrintInfo("No profile active. Run '1ctl profile use <name>' to select one.")
		return nil
	}

	if utils.TryPrintJSON(struct {
		Profile      string `json:"profile"`
		APIURL       string `json:"api_url"`
		Email        string `json:"email,omitempty"`
		Organization string `json:"organization,omitempty"`
	}{
		Profile:      name,
		APIURL:       config.GetConfig().ApiURL,
		Email:        satuskyctx.GetEmail(),
		Organization: satuskyctx.GetCurrentOrgName(),
	}) {
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
