package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v2"
)

func GithubCommand() *cli.Command {
	return &cli.Command{
		Name:    "github",
		Aliases: []string{"gh"},
		Usage:   "Manage GitHub integration",
		Subcommands: []*cli.Command{
			githubStatusCommand(),
			githubConnectCommand(),
			githubDisconnectCommand(),
			githubReposCommand(),
			githubInstallationCommand(),
		},
	}
}

func githubStatusCommand() *cli.Command {
	return &cli.Command{
		Name:   "status",
		Usage:  "Show GitHub connection status",
		Action: handleGitHubStatus,
	}
}

func githubConnectCommand() *cli.Command {
	return &cli.Command{
		Name:   "connect",
		Usage:  "Connect GitHub account",
		Action: handleGitHubConnect,
	}
}

func githubDisconnectCommand() *cli.Command {
	return &cli.Command{
		Name:   "disconnect",
		Usage:  "Disconnect GitHub account",
		Action: handleGitHubDisconnect,
	}
}

func githubReposCommand() *cli.Command {
	return &cli.Command{
		Name:  "repos",
		Usage: "Manage GitHub repositories",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List synced repositories",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "page",
						Usage: "Page number",
						Value: 1,
					},
					&cli.IntFlag{
						Name:  "limit",
						Usage: "Items per page",
						Value: 20,
					},
				},
				Action: handleGitHubReposList,
			},
			{
				Name:   "sync",
				Usage:  "Sync repositories from GitHub",
				Action: handleGitHubReposSync,
			},
			{
				Name:  "get",
				Usage: "Get repository details",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "repo-id",
						Usage:    "Repository ID",
						Required: true,
					},
				},
				Action: handleGitHubRepoGet,
			},
		},
		Action: handleGitHubReposList,
	}
}

func githubInstallationCommand() *cli.Command {
	return &cli.Command{
		Name:  "installation",
		Usage: "Manage GitHub App installation",
		Subcommands: []*cli.Command{
			{
				Name:   "info",
				Usage:  "Show installation info",
				Action: handleGitHubInstallationInfo,
			},
			{
				Name:   "revoke",
				Usage:  "Revoke installation",
				Action: handleGitHubInstallationRevoke,
			},
		},
		Action: handleGitHubInstallationInfo,
	}
}

func handleGitHubStatus(c *cli.Context) error {
	conn, err := api.GetGitHubConnection()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get GitHub status: %s", err.Error()), nil)
	}

	utils.PrintHeader("GitHub Connection Status")
	if conn.Connected {
		utils.PrintStatusLine("Status", "Connected")
		utils.PrintStatusLine("Username", conn.Username)
		utils.PrintStatusLine("Email", conn.Email)
		utils.PrintStatusLine("Connected At", formatTimeAgo(conn.ConnectedAt))
		if conn.AppInstalled {
			utils.PrintStatusLine("App Installed", "Yes")
			utils.PrintStatusLine("Installation ID", fmt.Sprintf("%d", conn.InstallationID))
		} else {
			utils.PrintStatusLine("App Installed", "No")
		}
	} else {
		utils.PrintStatusLine("Status", "Not Connected")
		utils.PrintInfo("Run '1ctl github connect' to connect your GitHub account")
	}
	return nil
}

func handleGitHubConnect(c *cli.Context) error {
	authURL, err := api.ConnectGitHub()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to initiate GitHub connection: %s", err.Error()), nil)
	}

	utils.PrintSuccess("GitHub OAuth initiated!")
	utils.PrintStatusLine("Auth URL", authURL)
	utils.PrintInfo("Open the URL above in your browser to complete the connection")
	return nil
}

func handleGitHubDisconnect(c *cli.Context) error {
	if err := api.DisconnectGitHub(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to disconnect GitHub: %s", err.Error()), nil)
	}

	utils.PrintSuccess("GitHub account disconnected successfully")
	return nil
}

func handleGitHubReposList(c *cli.Context) error {
	page := c.Int("page")
	limit := c.Int("limit")

	repos, err := api.GetGitHubRepositories(page, limit)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list repositories: %s", err.Error()), nil)
	}

	if len(repos) == 0 {
		utils.PrintInfo("No repositories found. Run '1ctl github repos sync' to sync from GitHub")
		return nil
	}

	utils.PrintHeader("GitHub Repositories")
	for _, repo := range repos {
		visibility := "public"
		if repo.Private {
			visibility = "private"
		}
		utils.PrintStatusLine("Name", repo.FullName)
		utils.PrintStatusLine("ID", repo.ID.String())
		utils.PrintStatusLine("Visibility", visibility)
		utils.PrintStatusLine("Branch", repo.DefaultBranch)
		if repo.Language != "" {
			utils.PrintStatusLine("Language", repo.Language)
		}
		utils.PrintStatusLine("Updated", formatTimeAgo(repo.UpdatedAt))
		utils.PrintDivider()
	}
	return nil
}

func handleGitHubReposSync(c *cli.Context) error {
	if err := api.SyncGitHubRepositories(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to sync repositories: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Repositories synced successfully from GitHub")
	return nil
}

func handleGitHubRepoGet(c *cli.Context) error {
	repoID := c.String("repo-id")
	if repoID == "" {
		return utils.NewError("--repo-id is required", nil)
	}

	repo, err := api.GetGitHubRepository(repoID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get repository: %s", err.Error()), nil)
	}

	visibility := "public"
	if repo.Private {
		visibility = "private"
	}

	utils.PrintHeader("Repository Details")
	utils.PrintStatusLine("Name", repo.FullName)
	utils.PrintStatusLine("ID", repo.ID.String())
	utils.PrintStatusLine("GitHub ID", fmt.Sprintf("%d", repo.GitHubID))
	utils.PrintStatusLine("Visibility", visibility)
	utils.PrintStatusLine("Default Branch", repo.DefaultBranch)
	if repo.Description != "" {
		utils.PrintStatusLine("Description", repo.Description)
	}
	if repo.Language != "" {
		utils.PrintStatusLine("Language", repo.Language)
	}
	utils.PrintStatusLine("Clone URL", repo.CloneURL)
	utils.PrintStatusLine("Web URL", repo.HTMLURL)
	utils.PrintStatusLine("Updated", formatTimeAgo(repo.UpdatedAt))
	return nil
}

func handleGitHubInstallationInfo(c *cli.Context) error {
	info, err := api.GetGitHubInstallationInfo()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get installation info: %s", err.Error()), nil)
	}

	utils.PrintHeader("GitHub App Installation")
	utils.PrintStatusLine("Installation ID", fmt.Sprintf("%d", info.InstallationID))
	utils.PrintStatusLine("Account", info.AccountLogin)
	utils.PrintStatusLine("Account Type", info.AccountType)
	utils.PrintStatusLine("Created", formatTimeAgo(info.CreatedAt))
	if len(info.Permissions) > 0 {
		fmt.Println()
		utils.PrintHeader("Permissions")
		for perm, level := range info.Permissions {
			utils.PrintStatusLine(perm, level)
		}
	}
	return nil
}

func handleGitHubInstallationRevoke(c *cli.Context) error {
	if err := api.RevokeGitHubInstallation(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to revoke installation: %s", err.Error()), nil)
	}

	utils.PrintSuccess("GitHub App installation revoked successfully")
	return nil
}
