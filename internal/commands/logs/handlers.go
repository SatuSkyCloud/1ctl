package logs

import (
	"context"
	"fmt"
	"net/http"

	"1ctl/internal/api"
	"1ctl/internal/deploy"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"

	gorillaws "github.com/gorilla/websocket"
)

// --- Handlers -----------------------------------------------------------

func handleLogs(ctx context.Context, in logsInput) error {
	deploymentID, err := deploy.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}

	tail := in.Tail

	logs, meta, err := api.GetStoredLogs(deploymentID, tail)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get logs: %s", err.Error()), nil)
	}

	if meta != nil {
		switch {
		case meta.Degraded:
			utils.PrintWarning("Loki unavailable; using stored deployment logs")
			if meta.FallbackReason != "" {
				utils.PrintInfo("Fallback reason: %s", meta.FallbackReason)
			}
		case meta.Source == "loki":
			utils.PrintInfo("Showing logs from Loki")
		}
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

func handleLogsStream(ctx context.Context, in logsStreamInput) error {
	namespace := in.Namespace
	appLabel := in.App
	batchSize := in.BatchSize
	resolvedDeploymentID := ""

	// Resolve deployment-id from --config if not provided directly
	if in.DeploymentID == "" && in.Namespace == "" {
		id, err := deploy.ResolveDeploymentID("", in.App, in.Config)
		if err == nil && id != "" {
			in.DeploymentID = id
		}
	}

	// Resolve via deployment ID if explicit flags not given
	if in.DeploymentID != "" {
		deployment, err := api.GetDeployment(in.DeploymentID)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to get deployment: %s", err.Error()), nil)
		}
		namespace = deployment.Namespace
		appLabel = deployment.AppLabel
		resolvedDeploymentID = in.DeploymentID
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
	headers.Set("x-satusky-api-key", satuskyctx.GetToken())

	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to connect to log stream: %s", err.Error()), nil)
	}
	defer conn.Close() //nolint:errcheck // cleanup on exit, error unactionable

	if resolvedDeploymentID != "" {
		utils.PrintInfo("Resolved deployment %s to %s/%s", resolvedDeploymentID, namespace, appLabel)
	} else {
		utils.PrintInfo("Using explicit log target %s/%s", namespace, appLabel)
	}
	if batchSize > 0 {
		utils.PrintInfo("Showing historical replay from stored logs, then following live logs")
	} else {
		utils.PrintInfo("Following live Kubernetes logs only")
	}
	utils.PrintInfo("Streaming logs for %s/%s - press Ctrl+C to stop", namespace, appLabel)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// Normal close on Ctrl+C or server disconnect
			return nil
		}
		fmt.Println(string(msg))
	}
}
