package api

import (
	"fmt"
	"net/http"
	"net/url"
)

const requestIDHeader = "X-Request-ID"

// DeploymentAdoptionRequest supplies the exact live-object preconditions
// required before the backend transfers Deployment field ownership.
type DeploymentAdoptionRequest struct {
	Reason                  string `json:"reason"`
	ExpectedUID             string `json:"expected_uid"`
	ExpectedResourceVersion string `json:"expected_resource_version"`
	ExpectedGeneration      int64  `json:"expected_generation"`
}

// DeploymentAdoptionResult describes the durable reconciliation state after
// a successful ownership transfer.
type DeploymentAdoptionResult struct {
	DeploymentID          string `json:"deployment_id"`
	Namespace             string `json:"namespace"`
	AppLabel              string `json:"app_label"`
	ClusterID             string `json:"cluster_id"`
	LiveUID               string `json:"live_uid"`
	LiveResourceVersion   string `json:"live_resource_version"`
	DesiredGeneration     int64  `json:"desired_generation"`
	ObservedGeneration    int64  `json:"observed_generation"`
	ReconciliationState   string `json:"reconciliation_state"`
	ReconciliationManaged bool   `json:"reconciliation_managed"`
	FieldManager          string `json:"field_manager"`
	Force                 bool   `json:"force"`
	AlreadyManaged        bool   `json:"already_managed"`
	RequestID             string `json:"request_id"`
}

// AdoptDeployment requests a guarded adoption through the main API. The
// request ID is caller supplied so a retry is traceable and stable.
func AdoptDeployment(deploymentID, requestID string, request DeploymentAdoptionRequest) (*DeploymentAdoptionResult, error) {
	var response struct {
		Error bool                     `json:"error"`
		Data  DeploymentAdoptionResult `json:"data"`
	}
	headers := make(http.Header)
	headers.Set(requestIDHeader, requestID)
	path := fmt.Sprintf("/admin/deployments/%s/adopt", url.PathEscape(deploymentID))
	if err := makeMainAPIRequestWithHeaders(http.MethodPost, path, request, &response, headers); err != nil {
		return nil, err
	}
	return &response.Data, nil
}
