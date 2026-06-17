package commands

import (
	"context"
	"1ctl/internal/config"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v3"
)

func defaultAPIURLDisplay() string {
	return fmt.Sprintf("(default: %s)", config.DefaultAPIURL())
}

func ProfileCommand() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "Manage named configuration profiles (API endpoints + credentials)",
		Commands: []*cli.Command{
			profileListCommand(),
			profileCreateCommand(),
			profileUseCommand(),
			profileCurrentCommand(),
			profileDeleteCommand(),
		},
	}
}

func profileListCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all profiles",
		Action: handleProfileList,
	}
}

func profileCreateCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new profile",
		ArgsUsage: "[--url <api-url>] <name>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "url",
				Usage: "API URL for this profile (e.g. http://localhost:8080/v1/cli)",
			},
		},
		Action: handleProfileCreate,
	}
}

func profileUseCommand() *cli.Command {
	return &cli.Command{
		Name:      "use",
		Usage:     "Switch to a profile",
		ArgsUsage: "<name>",
		Action:    handleProfileUse,
	}
}

func profileCurrentCommand() *cli.Command {
	return &cli.Command{
		Name:   "current",
		Usage:  "Show the active profile",
		Action: handleProfileCurrent,
	}
}

func profileDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a profile",
		ArgsUsage: "<name>",
		Action:    handleProfileDelete,
	}
}

func handleProfileList(ctx context.Context, cmd *cli.Command) error {
	profiles, err := satuskyctx.ListProfiles()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list profiles: %s", err.Error()), nil)
	}

	if len(profiles) == 0 {
		utils.PrintInfo("No profiles yet.")
		utils.PrintInfo("Create one with: 1ctl profile create [--url <api-url>] <name>")
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

func handleProfileCreate(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("profile name is required. Usage: 1ctl profile create [--url <url>] <name>", nil)
	}

	name := cmd.Args().First()
	apiURL := cmd.String("url")

	if err := satuskyctx.CreateProfile(name, apiURL); err != nil {
		return utils.NewError(fmt.Sprintf("failed to create profile: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Profile '%s' created", name)
	if apiURL != "" {
		utils.PrintStatusLine("API URL", apiURL)
	} else {
		utils.PrintStatusLine("API URL", defaultAPIURLDisplay())
	}
	utils.PrintInfo("Next steps:")
	utils.PrintInfo("  1ctl profile use %s", name)
	utils.PrintInfo("  1ctl auth login --token=<your-token>")
	return nil
}

func handleProfileUse(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("profile name is required. Usage: 1ctl profile use <name>", nil)
	}

	name := cmd.Args().First()

	if err := satuskyctx.UseProfile(name); err != nil {
		return utils.NewError(fmt.Sprintf("failed to switch profile: %s", err.Error()), nil)
	}

	// Show the resolved URL that commands will actually hit (runs the full
	// precedence chain: --api-url flag > SATUSKY_API_URL > profile > default).
	utils.PrintSuccess("Switched to profile '%s'", name)
	utils.PrintStatusLine("API URL", config.GetConfig().ApiURL)

	// Let user know if they haven't authenticated this profile yet
	if satuskyctx.GetToken() == "" {
		utils.PrintInfo("Run '1ctl auth login --token=<token>' to authenticate this profile")
	}

	return nil
}

func handleProfileCurrent(ctx context.Context, cmd *cli.Command) error {
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

func handleProfileDelete(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return utils.NewError("profile name is required. Usage: 1ctl profile delete <name>", nil)
	}

	name := cmd.Args().First()

	if err := satuskyctx.DeleteProfile(name); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete profile: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Profile '%s' deleted", name)
	return nil
}
