package commands

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"fmt"
	"net/http"

	gorillaws "github.com/gorilla/websocket"
	"github.com/urfave/cli/v2"
)

func LogsCommand() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "View and manage pod logs",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "deployment-id",
				Aliases: []string{"d"},
				Usage:   "Deployment ID to view logs for",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "Config name or path (e.g. staging, satusky.staging.toml). Default: satusky.toml",
			},
			&cli.IntFlag{
				Name:    "tail",
				Aliases: []string{"n"},
				Usage:   "Number of lines to show (default: 100)",
				Value:   100,
			},
		},
		Subcommands: []*cli.Command{
			logsStreamCommand(),
		},
		Action: handleLogs,
	}
}


func handleLogs(c *cli.Context) error {
	deploymentID, err := resolveDeploymentID(c.String("deployment-id"), c.String("config"))
	if err != nil {
		return err
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

// requireUserContext returns the userID from context or an error
func requireUserContext() (string, error) {
	userID := context.GetUserID()
	if userID == "" {
		return "", utils.NewError("user ID not found. Please run '1ctl auth login' first", nil)
	}
	return userID, nil
}

func handleLogsStream(c *cli.Context) error {
	namespace := c.String("namespace")
	appLabel := c.String("app")
	batchSize := c.Int("batch-size")

	// Resolve deployment-id from --config if not provided directly
	if c.String("deployment-id") == "" && c.String("namespace") == "" {
		id, err := resolveDeploymentID("", c.String("config"))
		if err == nil && id != "" {
			if err := c.Set("deployment-id", id); err != nil {
				return utils.NewError(fmt.Sprintf("failed to set deployment-id: %s", err.Error()), nil)
			}
		}
	}

	// Resolve via deployment ID if explicit flags not given
	if deploymentID := c.String("deployment-id"); deploymentID != "" {
		deployment, err := api.GetDeployment(deploymentID)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to get deployment: %s", err.Error()), nil)
		}
		namespace = deployment.Namespace
		appLabel = deployment.AppLabel
	}

	if namespace == "" || appLabel == "" {
		return utils.NewError("provide --deployment-id, or both --namespace and --app", nil)
	}

	wsURL, err := api.StreamPodLogsWSURL(namespace, appLabel)
	if err != nil {
		return err
	}

	if batchSize > 0 {
		wsURL = fmt.Sprintf("%s?batchSize=%d", wsURL, batchSize)
	}

	headers := http.Header{}
	headers.Set("x-satusky-api-key", context.GetToken())
	headers.Set("x-satusky-config", context.GetUserConfigKey())

	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to connect to log stream: %s", err.Error()), nil)
	}
	defer conn.Close() //nolint:errcheck // cleanup on exit, error unactionable

	utils.PrintInfo("Streaming logs for %s/%s — press Ctrl+C to stop", namespace, appLabel)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// Normal close on Ctrl+C or server disconnect
			return nil
		}
		fmt.Println(string(msg))
	}
}

// logsStreamCommand streams live pod logs over WebSocket
func logsStreamCommand() *cli.Command {
	return &cli.Command{
		Name:  "stream",
		Usage: "Stream live pod logs (like kubectl logs -f)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "deployment-id",
				Aliases: []string{"d"},
				Usage:   "Deployment ID (resolves namespace and app label automatically)",
			},
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n"},
				Usage:   "Kubernetes namespace (use with --app)",
			},
			&cli.StringFlag{
				Name:    "app",
				Aliases: []string{"a"},
				Usage:   "App label (use with --namespace)",
			},
			&cli.IntFlag{
				Name:  "batch-size",
				Usage: "Log lines per batch sent by the server",
				Value: 100,
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "Config name or path (e.g. staging, satusky.staging.toml). Default: satusky.toml",
			},
		},
		Action: handleLogsStream,
	}
}
