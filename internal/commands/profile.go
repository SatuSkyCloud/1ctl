package commands

import (
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v2"
)

func ProfileCommand() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "Manage named configuration profiles (API endpoints + credentials)",
		Subcommands: []*cli.Command{
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

func handleProfileList(c *cli.Context) error {
	profiles, err := context.ListProfiles()
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
			apiURL = "(default: https://api.satusky.com/v1/cli)"
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

func handleProfileCreate(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("profile name is required. Usage: 1ctl profile create [--url <url>] <name>", nil)
	}

	name := c.Args().First()
	apiURL := c.String("url")

	if err := context.CreateProfile(name, apiURL); err != nil {
		return utils.NewError(fmt.Sprintf("failed to create profile: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Profile '%s' created", name)
	if apiURL != "" {
		utils.PrintStatusLine("API URL", apiURL)
	} else {
		utils.PrintStatusLine("API URL", "(default: https://api.satusky.com/v1/cli)")
	}
	utils.PrintInfo("Next steps:")
	utils.PrintInfo("  1ctl profile use %s", name)
	utils.PrintInfo("  1ctl auth login --token=<your-token>")
	return nil
}

func handleProfileUse(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("profile name is required. Usage: 1ctl profile use <name>", nil)
	}

	name := c.Args().First()

	if err := context.UseProfile(name); err != nil {
		return utils.NewError(fmt.Sprintf("failed to switch profile: %s", err.Error()), nil)
	}

	// Show the active profile's API URL for confirmation
	apiURL := context.GetAPIURL()
	if apiURL == "" {
		apiURL = "(default: https://api.satusky.com/v1/cli)"
	}

	utils.PrintSuccess("Switched to profile '%s'", name)
	utils.PrintStatusLine("API URL", apiURL)

	// Let user know if they haven't authenticated this profile yet
	if context.GetToken() == "" {
		utils.PrintInfo("Run '1ctl auth login --token=<token>' to authenticate this profile")
	}

	return nil
}

func handleProfileCurrent(c *cli.Context) error {
	name := context.GetActiveProfileName()

	if name == "" {
		utils.PrintInfo("No profile active. Run '1ctl profile use <name>' to select one.")
		return nil
	}

	apiURL := context.GetAPIURL()
	if apiURL == "" {
		apiURL = "(default: https://api.satusky.com/v1/cli)"
	}

	utils.PrintHeader("Active Profile")
	utils.PrintStatusLine("Profile", name)
	utils.PrintStatusLine("API URL", apiURL)

	if email := context.GetEmail(); email != "" {
		utils.PrintStatusLine("Auth", email)
	} else {
		utils.PrintStatusLine("Auth", "(not logged in — run '1ctl auth login')")
	}

	if org := context.GetCurrentOrgName(); org != "" {
		utils.PrintStatusLine("Org", org)
	}

	return nil
}

func handleProfileDelete(c *cli.Context) error {
	if c.NArg() < 1 {
		return utils.NewError("profile name is required. Usage: 1ctl profile delete <name>", nil)
	}

	name := c.Args().First()

	if err := context.DeleteProfile(name); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete profile: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Profile '%s' deleted", name)
	return nil
}
