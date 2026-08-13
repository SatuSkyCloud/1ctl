package api

import (
	"1ctl/internal/utils"
	"fmt"
	"strings"
	"time"
)

// DeploymentEvent is one observed transition in a deployment's lifecycle.
type DeploymentEvent struct {
	ID         string            `json:"id"`
	At         time.Time         `json:"at"`
	Category   string            `json:"category"`
	Type       string            `json:"type"`
	Level      string            `json:"level"`
	Message    string            `json:"message"`
	Detail     map[string]string `json:"detail,omitempty"`
	Generation *int64            `json:"generation,omitempty"`
}

// Key identifies an event across polls. The timeline is derived from durable
// rows rather than an append-only log, so a re-poll returns the same events
// again; --follow needs a stable identity to print each one once.
//
// Those rows are mutable: re-completing a reconciliation task rewrites
// completed_at, which made one event print again on every poll with a new
// timestamp. The server-supplied ID is derived from the source row and is the
// identity to trust; the timestamp is only a fallback for an older server.
func (e DeploymentEvent) Key() string {
	if id := strings.TrimSpace(e.ID); id != "" {
		return id
	}
	return fmt.Sprintf("%d|%s|%s", e.At.UnixNano(), e.Category, e.Type)
}

// DeploymentEventsResponse is the chronological timeline for one deployment.
type DeploymentEventsResponse struct {
	DeploymentID string            `json:"deployment_id"`
	AppLabel     string            `json:"app_label"`
	Events       []DeploymentEvent `json:"events"`
}

// GetDeploymentEvents returns the deployment's lifecycle timeline, oldest first.
func GetDeploymentEvents(deploymentID string) (*DeploymentEventsResponse, error) {
	var resp struct {
		Error bool                     `json:"error"`
		Data  DeploymentEventsResponse `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/deployments/%s/events", deploymentID), nil, &resp); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to get deployment events: %s", err.Error()), nil)
	}
	return &resp.Data, nil
}
