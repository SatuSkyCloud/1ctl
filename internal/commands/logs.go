package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"

	"github.com/urfave/cli/v2"
)

func LogsCommand() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "View and manage pod logs",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "deployment-id",
				Aliases:  []string{"d"},
				Usage:    "Deployment ID to view logs for",
				Required: true,
			},
			&cli.IntFlag{
				Name:    "tail",
				Aliases: []string{"n"},
				Usage:   "Number of lines to show (default: 100)",
				Value:   100,
			},
			&cli.BoolFlag{
				Name:  "stats",
				Usage: "Show log statistics instead of logs",
			},
		},
		Subcommands: []*cli.Command{
			logsStatsCommand(),
			logsDeleteCommand(),
		},
		Action: handleLogs,
	}
}

func logsStatsCommand() *cli.Command {
	return &cli.Command{
		Name:  "stats",
		Usage: "Show log statistics for a deployment",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "deployment-id",
				Aliases:  []string{"d"},
				Usage:    "Deployment ID",
				Required: true,
			},
		},
		Action: handleLogsStats,
	}
}

func logsDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete logs for a deployment",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "deployment-id",
				Aliases:  []string{"d"},
				Usage:    "Deployment ID",
				Required: true,
			},
		},
		Action: handleLogsDelete,
	}
}

func handleLogs(c *cli.Context) error {
	deploymentID := c.String("deployment-id")
	if deploymentID == "" {
		return utils.NewError("--deployment-id is required", nil)
	}

	// If stats flag is set, show stats instead
	if c.Bool("stats") {
		return handleLogsStats(c)
	}

	tail := c.Int("tail")

	logs, err := api.GetStoredLogs(deploymentID, tail)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get logs: %s", err.Error()), nil)
	}

	if len(logs) == 0 {
		utils.PrintInfo("No logs found for deployment %s", deploymentID)
		return nil
	}

	utils.PrintHeader("Pod Logs")
	for _, log := range logs {
		timestamp := log.Timestamp.Format("2006-01-02 15:04:05")
		podName := log.PodName
		if podName == "" {
			podName = "unknown"
		}

		// Format: [timestamp] [pod-name] message
		fmt.Printf("[%s] [%s] %s\n", timestamp, podName, log.Message)
	}
	utils.PrintDivider()
	fmt.Printf("Showing last %d lines\n", len(logs))
	return nil
}

func handleLogsStats(c *cli.Context) error {
	deploymentID := c.String("deployment-id")
	if deploymentID == "" {
		return utils.NewError("--deployment-id is required", nil)
	}

	stats, err := api.GetLogStats(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get log stats: %s", err.Error()), nil)
	}

	utils.PrintHeader("Log Statistics")
	utils.PrintStatusLine("Deployment ID", stats.DeploymentID.String())
	utils.PrintStatusLine("Total Lines", fmt.Sprintf("%d", stats.TotalLines))
	utils.PrintStatusLine("Total Size", api.FormatBytes(stats.TotalSize))
	if !stats.OldestLog.IsZero() {
		utils.PrintStatusLine("Oldest Log", stats.OldestLog.Format("2006-01-02 15:04:05"))
	}
	if !stats.NewestLog.IsZero() {
		utils.PrintStatusLine("Newest Log", stats.NewestLog.Format("2006-01-02 15:04:05"))
	}

	// Calculate log rate if we have time range
	if !stats.OldestLog.IsZero() && !stats.NewestLog.IsZero() {
		duration := stats.NewestLog.Sub(stats.OldestLog)
		if duration.Hours() > 0 {
			rate := float64(stats.TotalLines) / duration.Hours()
			utils.PrintStatusLine("Log Rate", fmt.Sprintf("~%.0f lines/hour", rate))
		}
	}
	return nil
}

func handleLogsDelete(c *cli.Context) error {
	deploymentID := c.String("deployment-id")
	if deploymentID == "" {
		return utils.NewError("--deployment-id is required", nil)
	}

	if err := api.DeleteLogs(deploymentID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete logs: %s", err.Error()), nil)
	}

	utils.PrintSuccess("Logs for deployment %s deleted successfully", deploymentID)
	return nil
}
