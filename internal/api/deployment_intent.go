package api

import (
	"1ctl/internal/config"
	"1ctl/internal/utils"
	"fmt"
	"net/http"
	"strings"
)

// DeploymentIntent is the durable desired-state contract accepted by
// POST /v1/cli/deployments/intent. It intentionally carries secret key names,
// never secret values.
type DeploymentIntent struct {
	Deployment  Deployment                   `json:"deployment"`
	Environment []KeyValuePair               `json:"environment,omitempty"`
	Config      DeploymentDesiredStateConfig `json:"config,omitempty" tstype:",required"`
	Volumes     []DeploymentIntentVolume     `json:"volumes,omitempty"`
	Service     *DeploymentIntentService     `json:"service,omitempty"`
	PublicRoute *DeploymentIntentPublicRoute `json:"public_route,omitempty"`
}

type DeploymentDesiredStateConfig struct {
	StartupProbe    *DeploymentProbe           `json:"startup_probe,omitempty"`
	ReadinessProbe  *DeploymentProbe           `json:"readiness_probe,omitempty"`
	LivenessProbe   *DeploymentProbe           `json:"liveness_probe,omitempty"`
	RequiredSecrets []DeploymentRequiredSecret `json:"required_secrets,omitempty"`
}

type DeploymentRequiredSecret struct {
	Key string `json:"key"`
}

type DeploymentProbe struct {
	HTTPGet   *DeploymentHTTPGetProbe   `json:"http_get,omitempty"`
	TCPSocket *DeploymentTCPSocketProbe `json:"tcp_socket,omitempty"`
	Exec      *DeploymentExecProbe      `json:"exec,omitempty"`

	InitialDelaySeconds *int32 `json:"initial_delay_seconds,omitempty"`
	TimeoutSeconds      *int32 `json:"timeout_seconds,omitempty"`
	PeriodSeconds       *int32 `json:"period_seconds,omitempty"`
	SuccessThreshold    *int32 `json:"success_threshold,omitempty"`
	FailureThreshold    *int32 `json:"failure_threshold,omitempty"`
}

type DeploymentHTTPGetProbe struct {
	Path string `json:"path"`
	Port int32  `json:"port"`
}

type DeploymentTCPSocketProbe struct {
	Port int32 `json:"port"`
}

type DeploymentExecProbe struct {
	Command []string `json:"command"`
}

type DeploymentIntentVolume struct {
	VolumeName   string `json:"volume_name"`
	ClaimName    string `json:"claim_name"`
	StorageClass string `json:"storage_class"`
	StorageSize  string `json:"storage_size"`
	MountPath    string `json:"mount_path"`
}

type DeploymentIntentService struct {
	Name string `json:"name"`
	Port int32  `json:"port"`
}

type DeploymentIntentPublicRoute struct {
	Kind string `json:"kind"`
}

// DeploymentIntentAccepted is the backend's asynchronous acceptance record.
// A pending state means the desired state was persisted, not that the workload
// has already become healthy.
type DeploymentIntentAccepted struct {
	OperationID            string   `json:"operation_id"`
	DeploymentID           string   `json:"deployment_id"`
	Namespace              string   `json:"namespace"`
	AppLabel               string   `json:"app_label"`
	Generation             int64    `json:"generation"`
	ConfigRevision         int64    `json:"config_revision"`
	ConfigSHA256           string   `json:"config_sha256"`
	State                  string   `json:"state"`
	StatusURL              string   `json:"status_url"`
	MissingRequiredSecrets []string `json:"missing_required_secrets,omitempty"`
}

// CreateDeploymentIntent sends one logical, idempotent desired-state mutation.
// Transient failures are retried with the same request ID; 202 is a successful
// asynchronous acceptance and is returned without being mistaken for readiness.
func CreateDeploymentIntent(intent DeploymentIntent, requestID string) (*DeploymentIntentAccepted, error) {
	if strings.TrimSpace(requestID) == "" {
		return nil, utils.NewError("X-Request-ID is required", nil)
	}

	response := &DeploymentIntentAccepted{}
	headers := make(http.Header)
	headers.Set(requestIDHeader, requestID)
	endpoint := strings.TrimSuffix(config.GetConfig().ApiURL, "/") + "/deployments/intent"
	if err := makeRequestURLWithHeadersRetry(http.MethodPost, endpoint, intent, response, headers); err != nil {
		return nil, fmt.Errorf("create deployment intent: %w", err)
	}
	return response, nil
}
