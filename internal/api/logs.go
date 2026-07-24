package api

import (
	"1ctl/internal/config"
	"1ctl/internal/utils"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DeploymentLog represents a log entry for a deployment
type DeploymentLog struct {
	LogID        uuid.UUID `json:"log_id"`
	DeploymentID uuid.UUID `json:"deployment_id"`
	Timestamp    time.Time `json:"timestamp"`
	Message      string    `json:"message"`
	PodName      string    `json:"pod_name"`
	Container    string    `json:"container"`
	Level        string    `json:"level"`
}

// DeploymentLogsMeta carries explicit source/degradation information for logs.
type DeploymentLogsMeta struct {
	Source         string `json:"source"`
	Degraded       bool   `json:"degraded"`
	FallbackReason string `json:"fallback_reason,omitempty"`
	FallbackSource string `json:"fallback_source,omitempty"`
	Message        string `json:"message"`
}

type deploymentLogsResponse struct {
	Error          bool            `json:"error"`
	Message        string          `json:"message"`
	Data           []DeploymentLog `json:"data"`
	Source         string          `json:"source"`
	Degraded       bool            `json:"degraded"`
	FallbackReason string          `json:"fallback_reason,omitempty"`
	FallbackSource string          `json:"fallback_source,omitempty"`
	Count          int             `json:"count"`
}

// GetStoredLogs retrieves logs for a deployment via Loki (populated by Promtail).
func GetStoredLogs(deploymentID string, tail int) ([]DeploymentLog, *DeploymentLogsMeta, error) {
	path := fmt.Sprintf("/loki/logs/%s", deploymentID)
	if tail > 0 {
		path = fmt.Sprintf("%s?tail=%d", path, tail)
	}

	var resp deploymentLogsResponse
	err := makeRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, nil, err
	}

	meta := &DeploymentLogsMeta{
		Source:         resp.Source,
		Degraded:       resp.Degraded,
		FallbackReason: resp.FallbackReason,
		FallbackSource: resp.FallbackSource,
		Message:        resp.Message,
	}
	return resp.Data, meta, nil
}

// StreamPodLogsWSURL returns the WebSocket URL for streaming pod logs.
// The WS endpoint lives outside the CLI path (/v1/pods/stream/...) so we
// reconstruct the host from the CLI API base URL.
func StreamPodLogsWSURL(namespace, appLabel string) (string, error) {
	cfg := config.GetConfig()
	// cfg.ApiURL is e.g. "https://api.satusky.com/v1/cli"
	// Strip the /v1/cli suffix to get the base host
	base := cfg.ApiURL
	for _, suffix := range []string{"/v1/cli", "/v1"} {
		if idx := len(base) - len(suffix); idx >= 0 && base[idx:] == suffix {
			base = base[:idx]
			break
		}
	}
	// Swap http→ws and https→wss
	switch {
	case len(base) >= 8 && base[:8] == "https://":
		base = "wss://" + base[8:]
	case len(base) >= 7 && base[:7] == "http://":
		base = "ws://" + base[7:]
	default:
		return "", utils.NewError("unexpected API URL scheme: "+cfg.ApiURL, nil)
	}
	return fmt.Sprintf("%s/v1/pods/stream/logs/%s/%s", base, namespace, appLabel), nil
}
