package api

import (
	"1ctl/internal/config"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"strings"

	"github.com/google/uuid"
)

// httpClient is a shared HTTP client with a reasonable timeout.
// 30 seconds is sufficient for most API calls while preventing indefinite hangs.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Common response structure that matches backend
type apiResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Count   int         `json:"count,omitempty"`
	Data    interface{} `json:"data"`
}

// DeleteDeployment deletes a deployment
func DeleteDeployment(deploymentID string) (*DeletionResult, error) {
	var resp apiResponse
	if err := makeRequest("POST", fmt.Sprintf("/deployments/delete/%s", deploymentID), nil, &resp); err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal deletion result: %s", err.Error()), nil)
	}

	var result DeletionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal deletion result: %s", err.Error()), nil)
	}
	return &result, nil
}

// RestartDeployment triggers a rolling restart of a deployment
func RestartDeployment(deploymentID string) error {
	return makeRequest("POST", fmt.Sprintf("/deployments/%s/restart", deploymentID), nil, nil)
}

// ListDeployments lists all deployments for the current namespace.
// Thin wrapper around ListDeploymentsByNamespace using the active context.
func ListDeployments() ([]Deployment, error) {
	namespace, err := context.GetCurrentNamespaceOrError()
	if err != nil {
		return nil, err
	}
	return ListDeploymentsByNamespace(namespace)
}

// ListDeploymentVersions returns the release history for a deployment.
func ListDeploymentVersions(deploymentID string) ([]DeploymentVersion, error) {
	var resp struct {
		Error   bool                `json:"error"`
		Message string              `json:"message"`
		Data    []DeploymentVersion `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/deployments/%s/versions", deploymentID), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// RollbackDeployment initiates a rollback to the specified version number.
func RollbackDeployment(deploymentID string, versionNumber int) error {
	return makeRequest("POST", fmt.Sprintf("/deployments/%s/rollback/%d", deploymentID, versionNumber), nil, nil)
}

// ListDeploymentsByNamespace lists deployments in a specific namespace
func ListDeploymentsByNamespace(namespace string) ([]Deployment, error) {
	var response struct {
		Error   bool         `json:"error"`
		Message string       `json:"message"`
		Count   int          `json:"count"`
		Data    []Deployment `json:"data"`
	}
	err := makeRequest("GET", fmt.Sprintf("/deployments/namespace/%s", namespace), nil, &response)
	return response.Data, err
}

// GetDeploymentByAppLabel looks up a deployment by its app label within a namespace.
// This is the primary way the CLI resolves deployment IDs from satusky.toml.
func GetDeploymentByAppLabel(namespace, appLabel string) (*Deployment, error) {
	var resp struct {
		Error bool       `json:"error"`
		Data  Deployment `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/deployments/namespace/%s/app/%s", namespace, appLabel), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetDeployment gets details for a specific deployment.
// Uses the typed-inline-struct pattern: the response is unmarshalled once
// directly into the Deployment struct, avoiding the legacy apiResponse +
// json.Marshal + json.Unmarshal double-encoding round trip.
func GetDeployment(deploymentID string) (*Deployment, error) {
	var resp struct {
		Error bool       `json:"error"`
		Data  Deployment `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/deployments/id/%s", deploymentID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Service methods

func ListServices() ([]Service, error) {
	namespace := context.GetCurrentNamespace()
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/services/namespace/%s", namespace), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var services []Service
	if err := json.Unmarshal(data, &services); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal services: %s", err.Error()), nil)
	}
	return services, nil
}

func DeleteService(serviceID string) error {
	var resp apiResponse
	return makeRequest("POST", fmt.Sprintf("/services/delete/%s", serviceID), nil, &resp)
}

// Secret methods
func CreateSecret(secret Secret) (*Secret, error) {
	var resp apiResponse
	var secretResp Secret
	resp.Data = &secretResp

	// Only fall back to context namespace if caller didn't set one. Overriding
	// a non-empty value silently routes resources to the wrong namespace when
	// the deploy was scoped via --organization.
	if secret.Namespace == "" {
		secret.Namespace = context.GetCurrentNamespace()
	}

	err := makeRequest("POST", "/secrets/upsert", secret, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	if err := json.Unmarshal(data, &secretResp); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal secret response: %s", err.Error()), nil)
	}
	return &secretResp, nil
}

func ListSecrets() ([]Secret, error) {
	namespace := context.GetCurrentNamespace()
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/secrets/namespace/%s", namespace), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var secrets []Secret
	if err := json.Unmarshal(data, &secrets); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal secrets: %s", err.Error()), nil)
	}
	return secrets, nil
}

func DeleteSecret(secretID string) error {
	var resp apiResponse
	return makeRequest("POST", fmt.Sprintf("/secrets/delete/%s", secretID), nil, &resp)
}

// GetSecretsByDeploymentID returns secrets for a given deployment ID.
func GetSecretsByDeploymentID(deploymentID string) ([]Secret, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/secrets/deploymentId/%s", deploymentID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var secrets []Secret
	if err := json.Unmarshal(data, &secrets); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal secrets: %s", err.Error()), nil)
	}
	return secrets, nil
}

// UnsetSecretKey removes a single key from a secret.
func UnsetSecretKey(secretID, key string) error {
	body := map[string]string{"key": key}
	var resp struct{}
	return makeRequest("POST", fmt.Sprintf("/secrets/unset/%s", secretID), body, &resp)
}

// Ingress methods

// ListIngresses lists all ingresses for current namespace
func ListIngresses() ([]Ingress, error) {
	namespace := context.GetCurrentNamespace()
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/ingresses/namespace/%s", namespace), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var ingresses []Ingress
	if err := json.Unmarshal(data, &ingresses); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal ingresses: %s", err.Error()), nil)
	}
	return ingresses, nil
}

func DeleteIngress(ingressID string) error {
	var resp apiResponse
	return makeRequest("POST", fmt.Sprintf("/ingresses/delete/%s", ingressID), nil, &resp)
}

// Environment methods
func UpsertEnvironment(env Environment) (*Environment, error) {
	var resp apiResponse
	var envResp Environment
	resp.Data = &envResp

	// Only fall back to context namespace if caller didn't set one. Overriding
	// a non-empty value silently routes resources to the wrong namespace when
	// the deploy was scoped via --organization.
	if env.Namespace == "" {
		env.Namespace = context.GetCurrentNamespace()
	}

	err := makeRequest("POST", "/environments/upsert", env, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	if err := json.Unmarshal(data, &envResp); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal environment response: %s", err.Error()), nil)
	}
	return &envResp, nil
}

// ListEnvironments lists all environments for current namespace
func ListEnvironments() ([]Environment, error) {
	namespace := context.GetCurrentNamespace()
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/environments/namespace/%s", namespace), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var environments []Environment
	if err := json.Unmarshal(data, &environments); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal environments: %s", err.Error()), nil)
	}
	return environments, nil
}

func DeleteEnvironment(environmentID string) error {
	var resp apiResponse
	return makeRequest("POST", fmt.Sprintf("/environments/delete/%s", environmentID), nil, &resp)
}

// GetEnvironmentsByDeploymentID returns environments for a given deployment ID.
func GetEnvironmentsByDeploymentID(deploymentID string) ([]Environment, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/environments/deploymentId/%s", deploymentID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var environments []Environment
	if err := json.Unmarshal(data, &environments); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal environments: %s", err.Error()), nil)
	}
	return environments, nil
}

// UnsetEnvironmentKey removes a single key from an environment's ConfigMap.
func UnsetEnvironmentKey(environmentID, key string) error {
	body := map[string]string{"key": key}
	var resp struct{}
	return makeRequest("POST", fmt.Sprintf("/environments/unset/%s", environmentID), body, &resp)
}

// LoginCLI logs in the CLI with the API token
func LoginCLI(token string) (*TokenValidate, error) {
	cfg := config.GetConfig()

	body := map[string]string{"token": token}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal request body: %s", err.Error()), nil)
	}

	url := fmt.Sprintf("%s/auth/login", cfg.ApiURL)
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to create request: %s", err.Error()), nil)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-satusky-api-key", token)

	resp, err := httpClient.Do(request)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to send request: %s", err.Error()), nil)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body) //nolint:errcheck
		return nil, utils.NewError(fmt.Sprintf("login failed with status %d: %s", resp.StatusCode, string(bodyBytes)), nil)
	}

	var result TokenValidate
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to decode response: %s", err.Error()), nil)
	}

	if !result.Valid {
		return nil, utils.NewError("invalid token", nil)
	}

	return &result, nil
}

// CreateVolume creates a new volume for a deployment
func CreateVolume(volume Volume) error {
	return makeRequest("POST", "/volumes/create", volume, nil)
}

// GetAllVolumes returns all volumes across all namespaces.
func GetAllVolumes() ([]Volume, error) {
	var resp struct {
		Error bool     `json:"error"`
		Data  []Volume `json:"data"`
	}
	if err := makeRequest("GET", "/volumes/all", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetVolumeLifecycleStatus reports DB, PVC, and mount state for a volume.
func GetVolumeLifecycleStatus(volumeID string) (*VolumeLifecycleStatus, error) {
	var resp struct {
		Error bool                  `json:"error"`
		Data  VolumeLifecycleStatus `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/volumes/id/%s/status", volumeID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetDeploymentVolumeLifecycleStatuses reports all volume lifecycle state for a deployment.
func GetDeploymentVolumeLifecycleStatuses(deploymentID string) ([]VolumeLifecycleStatus, error) {
	var resp struct {
		Error bool                    `json:"error"`
		Data  []VolumeLifecycleStatus `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/volumes/deploymentId/%s/status", deploymentID), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// DetachVolume removes the deployment mount reference without deleting the PVC.
func DetachVolume(volumeID string) (*VolumeLifecycleStatus, error) {
	var resp struct {
		Error bool                  `json:"error"`
		Data  VolumeLifecycleStatus `json:"data"`
	}
	if err := makeRequest("POST", fmt.Sprintf("/volumes/%s/detach", volumeID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// DeleteVolumePVC detaches and destroys the live PVC, then removes the DB record.
func DeleteVolumePVC(volumeID string) (*VolumeLifecycleStatus, error) {
	var resp struct {
		Error bool                  `json:"error"`
		Data  VolumeLifecycleStatus `json:"data"`
	}
	if err := makeRequest("DELETE", fmt.Sprintf("/volumes/%s/pvc", volumeID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetDeploymentStatus gets the current status of a deployment.
// Uses the typed-inline-struct pattern (see GetDeployment for rationale).
func GetDeploymentStatus(deploymentID string) (*DeploymentStatus, error) {
	var resp struct {
		Error bool             `json:"error"`
		Data  DeploymentStatus `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/deployments/status/%s", deploymentID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// WaitForDeployment waits for a deployment to reach a terminal state
func WaitForDeployment(deploymentID string, timeout time.Duration) (*DeploymentStatus, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return nil, utils.NewError("timeout waiting for deployment", nil)
		}

		status, err := GetDeploymentStatus(deploymentID)
		if err != nil {
			return nil, err
		}

		// Two distinct "running" tokens coexist intentionally:
		//   StatusRunning    ("running") — backend lifecycle: in-progress, not yet healthy
		//   StatusRunningK8s ("Running") — K8s pod phase: healthy, terminal-success
		// Same casing distinction for Failed/failed.
		switch status.Status {
		case StatusCompleted, StatusRunningK8s:
			return status, nil
		case StatusFailed, StatusFailedK8s:
			return status, utils.NewError(fmt.Sprintf("deployment failed: %s", status.Message), nil)
		case StatusPending, StatusCreating, StatusRunning, StatusNotReady, StatusProgressing, StatusUnknown:
			utils.PrintInfo("Deployment status: %s (%d pct)", status.Status, status.Progress)
		default:
			// Forward-compatibility: a new status string added on the backend
			// must not break --wait on older CLI versions. Treat unknown
			// values as non-terminal and keep polling.
			utils.PrintInfo("Deployment status: %s (waiting...)", status.Status)
		}

		<-ticker.C
	}
}

// makeRequest is a helper function to make HTTP requests
func makeRequest(method, path string, body interface{}, response interface{}) error {
	config := config.GetConfig()
	url := fmt.Sprintf("%s%s", config.ApiURL, path)
	return makeRequestURL(method, url, body, response)
}

func makeMainAPIRequest(method, path string, body interface{}, response interface{}) error {
	cfg := config.GetConfig()
	baseURL := strings.TrimSuffix(cfg.ApiURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/cli")
	baseURL = strings.TrimSuffix(baseURL, "/")
	url := fmt.Sprintf("%s%s", baseURL, path)
	return makeRequestURL(method, url, body, response)
}

func makeRequestURL(method, url string, body interface{}, response interface{}) error {
	// Enforce HTTPS for non-localhost API URLs to prevent token leakage over plaintext
	if !utils.IsLocalhostURL(url) && !strings.HasPrefix(url, "https://") {
		return utils.NewError(fmt.Sprintf("refusing to send auth token over insecure connection (%s). Use HTTPS or http://localhost for local development", url), nil)
	}

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to marshal request body: %s", err.Error()), nil)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create request: %s", err.Error()), nil)
	}

	token := context.GetToken()
	if token == "" {
		return utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate", nil)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-satusky-api-key", token)
	if email := context.GetEmail(); email != "" {
		req.Header.Set("x-satusky-user-email", email)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to make request: %s", err.Error()), nil)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to read response body: %s", err.Error()), nil)
	}

	if resp.StatusCode >= 400 {
		// Check for resource exhausted error (422 Unprocessable Entity)
		resourceErr, parseErr := utils.ParseResourceExhaustedFromBytes(respBody, resp.StatusCode)
		if parseErr == nil && resourceErr != nil {
			return utils.NewResourceExhaustedCLIError(resourceErr)
		}

		var apiError APIError
		if err := json.Unmarshal(respBody, &apiError); err != nil {
			return utils.NewError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(respBody)), nil)
		}
		if resp.StatusCode == 500 {
			return utils.NewError(fmt.Sprintf("%s — check backend logs for details", apiError.Message), nil)
		}
		return utils.NewError(apiError.Message, nil)
	}

	if response != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, response); err != nil {
			return utils.NewError(fmt.Sprintf("failed to parse response: %s", err.Error()), nil)
		}
	}

	return nil
}

// GetDeploymentLogs gets deployment logs
func GetDeploymentLogs(deploymentID string) ([]string, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/logs/%s", deploymentID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var logs struct {
		Messages []string `json:"messages"`
	}
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal logs: %s", err.Error()), nil)
	}
	return logs.Messages, nil
}

// Add Issuer methods
func CreateIssuer(issuer Issuer) (*Issuer, error) {
	var resp apiResponse
	var issuerResp Issuer
	resp.Data = &issuerResp

	err := makeRequest("POST", "/issuers/upsert", issuer, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	if err := json.Unmarshal(data, &issuerResp); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal issuer response: %s", err.Error()), nil)
	}
	return &issuerResp, nil
}

func ListIssuers() ([]Issuer, error) {
	namespace := context.GetCurrentNamespace()
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/issuers/namespace/%s", namespace), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var issuers []Issuer
	if err := json.Unmarshal(data, &issuers); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal issuers: %s", err.Error()), nil)
	}
	return issuers, nil
}

func DeleteIssuer(issuerID string) error {
	var resp apiResponse
	return makeRequest("POST", fmt.Sprintf("/issuers/delete/%s", issuerID), nil, &resp)
}

// GetOrganizationByID gets organization details by ID
func GetOrganizationByID(orgID uuid.UUID) (*Organization, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/organizations/id/%s", orgID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var org Organization
	if err := json.Unmarshal(data, &org); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal organization: %s", err.Error()), nil)
	}
	return &org, nil
}

// User methods
func GetUserByEmail(email string) (*User, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/users/email/%s", email), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal user: %s", err.Error()), nil)
	}
	return &user, nil
}

// GetUserProfile gets the current user's profile with organization information
func GetUserProfile() (*UserProfile, error) {
	var resp apiResponse
	err := makeRequest("GET", "/users/profile", nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var profile UserProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal user profile: %s", err.Error()), nil)
	}
	return &profile, nil
}

// API Token methods
func GetUserTokens(userID string, orgID string) ([]APIToken, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/api-tokens/list/%s/%s", userID, orgID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var tokens []APIToken
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal tokens: %s", err.Error()), nil)
	}
	return tokens, nil
}

// Ingress methods
func GetIngressByDomainName(domainName string) (*Ingress, error) {
	var resp apiResponse
	// PathEscape so domains with characters that would otherwise be reserved
	// (e.g. ?, #, /) don't break the request URL. The backend validates the
	// decoded value separately.
	err := makeRequest("GET", fmt.Sprintf("/ingresses/domainName/%s", url.PathEscape(domainName)), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var ingress Ingress
	if err := json.Unmarshal(data, &ingress); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal ingress: %s", err.Error()), nil)
	}
	return &ingress, nil
}

func ListDomainAliases(ingressID string) ([]IngressAlias, error) {
	var resp struct {
		Error bool           `json:"error"`
		Data  []IngressAlias `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/ingresses/%s/aliases", ingressID), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func AttachDomain(ingressID string, req AttachDomainRequest) (*IngressAlias, error) {
	var resp struct {
		Error bool         `json:"error"`
		Data  IngressAlias `json:"data"`
	}
	if err := makeRequest("POST", fmt.Sprintf("/ingresses/%s/domains", ingressID), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func DetachDomain(ingressID string, req DetachDomainRequest) error {
	var resp apiResponse
	return makeRequest("POST", fmt.Sprintf("/ingresses/%s/domains/detach", ingressID), req, &resp)
}

// GetIngressByDeploymentID gets existing ingress by deployment ID
func GetIngressByDeploymentID(deploymentID string) (*Ingress, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/ingresses/deploymentId/%s", deploymentID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var ingress Ingress
	if err := json.Unmarshal(data, &ingress); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal ingress: %s", err.Error()), nil)
	}
	return &ingress, nil
}

// GetDomainStatus returns consolidated backend, route, DNS, TLS, and optional HTTP status.
func GetDomainStatus(ingressID, domain string, probe bool) (*DomainStatusResponse, error) {
	query := url.Values{}
	if domain != "" {
		query.Set("domain", domain)
	}
	if probe {
		query.Set("probe", "true")
	}
	path := fmt.Sprintf("/ingresses/%s/domain-status", ingressID)
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp struct {
		Error bool                 `json:"error"`
		Data  DomainStatusResponse `json:"data"`
	}
	if err := makeRequest("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Environment methods
func GetEnvironmentsByNamespace(namespace string) ([]Environment, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/environments/namespace/%s", namespace), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var environments []Environment
	if err := json.Unmarshal(data, &environments); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal environments: %s", err.Error()), nil)
	}
	return environments, nil
}

// Machine methods

// MachineTagLabelPrefix is the namespaced label key prefix used to record
// user-defined machine tags. A machine tagged "production" has the K8s node
// label `satusky.com/production` set on it.
const MachineTagLabelPrefix = "satusky.com/"

// GetMachineLabels returns the satusky.com/* labels for a machine. Used by
// the deploy `--machine-tag` resolver to filter owned machines client-side
// without a new backend endpoint.
func GetMachineLabels(machineID string) (map[string]string, error) {
	var resp struct {
		Error bool `json:"error"`
		Data  struct {
			Custom map[string]string `json:"custom"`
		} `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/machines/%s/labels", url.PathEscape(machineID)), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Custom, nil
}

type updateLabelsRequest struct {
	Labels map[string]string `json:"labels"`
}

// UpdateMachineLabels sets or removes satusky.com/* labels on a machine.
// To remove a label, set its value to empty string.
func UpdateMachineLabels(machineID string, labels map[string]string) error {
	var resp apiResponse
	req := updateLabelsRequest{Labels: labels}
	return makeRequest("PATCH", fmt.Sprintf("/machines/%s/labels", machineID), req, &resp)
}

// QueryMachinesByLabel finds machines that have ALL the specified labels.
func QueryMachinesByLabel(labels map[string]string) ([]Machine, error) {
	var resp struct {
		Error bool      `json:"error"`
		Data  []Machine `json:"data"`
	}
	if err := makeRequest("POST", "/machines/label-query", labels, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetAvailableLabelKeys returns all satusky.com/* label keys currently in use.
func GetAvailableLabelKeys() ([]string, error) {
	var resp struct {
		Error bool     `json:"error"`
		Data  []string `json:"data"`
	}
	if err := makeRequest("GET", "/machines/label-keys", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func CreateMachine(machine Machine) (int64, error) {
	var resp struct {
		Error bool  `json:"error"`
		Data  int64 `json:"data"`
	}
	if err := makeRequest("POST", "/machines/create", machine, &resp); err != nil {
		return 0, err
	}
	return resp.Data, nil
}

func UpdateMachine(machineID string, machine Machine) error {
	var resp apiResponse
	return makeRequest("POST", fmt.Sprintf("/machines/update/%s", machineID), machine, &resp)
}

// DeleteMachine decommissions a machine by UUID. Uses the main API route
// (DELETE /machines/:machineId) rather than the CLI route
// (POST /machines/delete/:id) because the CLI route expects a numeric
// database ID, but the CLI operates with UUID machine IDs.
func DeleteMachine(machineID string) error {
	var resp apiResponse
	return makeMainAPIRequest("DELETE", fmt.Sprintf("/machines/%s", url.PathEscape(machineID)), nil, &resp)
}

func GetMachineHardware(machineID string) (map[string]interface{}, error) {
	var resp struct {
		Error bool                   `json:"error"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/machines/%s/hardware", url.PathEscape(machineID)), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func RefreshMachineHardware(machineID string) (map[string]interface{}, error) {
	var resp struct {
		Error bool                   `json:"error"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := makeRequest("POST", fmt.Sprintf("/machines/%s/hardware/refresh", url.PathEscape(machineID)), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func GetMachineTalosStatus(machineID string) (map[string]interface{}, error) {
	var resp struct {
		Error bool                   `json:"error"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/machines/%s/talos/status", url.PathEscape(machineID)), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func GetMachineDetails(machineID string) (map[string]interface{}, error) {
	var resp struct {
		Error bool                   `json:"error"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := makeMainAPIRequest("GET", fmt.Sprintf("/machines/%s/details", url.PathEscape(machineID)), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func FetchMachineLogs(machineID string, req MachineLogFetchRequest) (*MachineLogsResponse, error) {
	var resp struct {
		Error bool                `json:"error"`
		Data  MachineLogsResponse `json:"data"`
	}
	if err := makeMainAPIRequest("POST", fmt.Sprintf("/machines/%s/logs/fetch", url.PathEscape(machineID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func GetMachineEvents(machineID string, tail int) (*MachineEventsResponse, error) {
	var resp struct {
		Error bool                  `json:"error"`
		Data  MachineEventsResponse `json:"data"`
	}
	path := fmt.Sprintf("/machines/%s/events", url.PathEscape(machineID))
	if tail > 0 {
		path = fmt.Sprintf("%s?tail=%d", path, tail)
	}
	if err := makeMainAPIRequest("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// MachineHasTag reports whether the given label set contains the satusky.com/<tag> key.
func MachineHasTag(labels map[string]string, tag string) bool {
	if tag == "" || labels == nil {
		return false
	}
	_, ok := labels[MachineTagLabelPrefix+tag]
	return ok
}

func GetMachinesByOwnerID(ownerID uuid.UUID) ([]Machine, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/machines/ownerId/%s", ownerID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var machines []Machine
	if err := json.Unmarshal(data, &machines); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal machines: %s", err.Error()), nil)
	}
	return machines, nil
}

func GetMachineByID(machineID uuid.UUID) (*Machine, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/machines/id/%s", machineID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var machine Machine
	if err := json.Unmarshal(data, &machine); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal machine: %s", err.Error()), nil)
	}
	return &machine, nil
}

func GetMachineByName(machineName string) (*Machine, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/machines/name/%s", machineName), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var machine Machine
	if err := json.Unmarshal(data, &machine); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal machine: %s", err.Error()), nil)
	}
	return &machine, nil
}

func GetAvailableMachines() ([]Machine, error) {
	var resp apiResponse
	err := makeRequest("GET", "/machines/monetized", nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var machines []Machine
	if err := json.Unmarshal(data, &machines); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal machines: %s", err.Error()), nil)
	}
	return machines, nil
}

// SendMachineCommand sends a command to a Mac agent machine
func SendMachineCommand(machineID string, req SendCommandRequest) (*SendCommandResponse, error) {
	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/machines/%s/command", machineID), req, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var cmdResp SendCommandResponse
	if err := json.Unmarshal(data, &cmdResp); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal command response: %s", err.Error()), nil)
	}
	return &cmdResp, nil
}

// UpsertDeployment creates or updates a deployment and returns the deployment ID
func UpsertDeployment(req Deployment, response *string) error {
	var resp apiResponse
	resp.Data = response

	path := fmt.Sprintf("/deployments/upsert/%s/%s", req.Namespace, req.AppLabel)
	return makeRequest("POST", path, req, &resp)
}

// UpsertService creates or updates a service and returns the service ID
func UpsertService(service Service, response *string) error {
	var resp apiResponse
	resp.Data = response

	path := fmt.Sprintf("/services/upsert/%s/%s", service.Namespace, service.ServiceName)
	return makeRequest("POST", path, service, &resp)
}

// UpsertIngress creates or updates an ingress and returns the ingress
func UpsertIngress(ingress Ingress) (*Ingress, error) {
	var resp apiResponse
	var ingressIDStr string
	resp.Data = &ingressIDStr

	path := fmt.Sprintf("/ingresses/upsert/%s/%s", ingress.Namespace, ingress.AppLabel)
	err := makeRequest("POST", path, ingress, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	// The backend returns just the ingress ID as a string
	if err := json.Unmarshal(data, &ingressIDStr); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal ingress ID: %s", err.Error()), nil)
	}

	// Parse the ingress ID
	ingressID, err := uuid.Parse(ingressIDStr)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to parse ingress ID: %s", err.Error()), nil)
	}

	// Return the ingress object with the ID set
	ingress.IngressID = ingressID
	return &ingress, nil
}

// ============================================================
// Machine Usage API
// ============================================================

// GetUserMachineUsages lists machine usage records for a user
func GetUserMachineUsages(userID string) ([]MachineUsageRecord, error) {
	var resp apiResponse
	if err := makeRequest("GET", fmt.Sprintf("/machine-usage/user/%s", userID), nil, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var usages []MachineUsageRecord
	if err := json.Unmarshal(data, &usages); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal machine usages: %s", err.Error()), nil)
	}
	return usages, nil
}

// GetMachineUsageByID retrieves a single machine usage record by ID
func GetMachineUsageByID(usageID string) (*MachineUsageRecord, error) {
	var resp apiResponse
	if err := makeRequest("GET", fmt.Sprintf("/machine-usage/%s", usageID), nil, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var usage MachineUsageRecord
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal machine usage: %s", err.Error()), nil)
	}
	return &usage, nil
}

// GetUsageCost calculates cost for a machine usage record
func GetUsageCost(usageID string) (*UsageCostResponse, error) {
	var resp apiResponse
	if err := makeRequest("GET", fmt.Sprintf("/machine-usage/%s/cost", usageID), nil, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var cost UsageCostResponse
	if err := json.Unmarshal(data, &cost); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal usage cost: %s", err.Error()), nil)
	}
	return &cost, nil
}

// ============================================================
// Pricing Config API
// ============================================================

// ListPricingConfigs lists all pricing configurations
func ListPricingConfigs() ([]PricingConfig, error) {
	var resp apiResponse
	if err := makeRequest("GET", "/pricing/configs", nil, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var configs []PricingConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal pricing configs: %s", err.Error()), nil)
	}
	return configs, nil
}

// GetPricingConfig retrieves a single pricing config by ID
func GetPricingConfig(configID string) (*PricingConfig, error) {
	var resp apiResponse
	if err := makeRequest("GET", fmt.Sprintf("/pricing/configs/%s", configID), nil, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var config PricingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal pricing config: %s", err.Error()), nil)
	}
	return &config, nil
}

// GetPricingByRegionAndType looks up pricing for a specific region, machine type, and SLA tier
func GetPricingByRegionAndType(region, machineType, slaTier string) (*PricingConfig, error) {
	var resp apiResponse
	if err := makeRequest("GET", fmt.Sprintf("/pricing/lookup/%s/%s/%s", region, machineType, slaTier), nil, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var config PricingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal pricing config: %s", err.Error()), nil)
	}
	return &config, nil
}

// CalculateMachineCost calculates cost for a machine over a time range
func CalculateMachineCost(machineRefID, machineID string, req CostCalculationRequest) (*CostCalculationResponse, error) {
	var resp apiResponse
	if err := makeRequest("POST", fmt.Sprintf("/pricing/calculate/%s/%s", machineRefID, machineID), req, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var result CostCalculationResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal cost calculation: %s", err.Error()), nil)
	}
	return &result, nil
}

// ============================================================
// Billing Settings API
// ============================================================

// GetAutoTopupSettings retrieves auto-topup settings for an organization
func GetAutoTopupSettings(orgID string) (*AutoTopupSettings, error) {
	var resp apiResponse
	if err := makeRequest("GET", fmt.Sprintf("/billing/organizations/%s/auto-topup", orgID), nil, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var settings AutoTopupSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal auto-topup settings: %s", err.Error()), nil)
	}
	return &settings, nil
}

// UpdateAutoTopupSettings updates auto-topup settings for an organization
func UpdateAutoTopupSettings(orgID string, req AutoTopupSettingsRequest) (*AutoTopupSettings, error) {
	var resp apiResponse
	if err := makeRequest("PUT", fmt.Sprintf("/billing/organizations/%s/auto-topup", orgID), req, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var settings AutoTopupSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal auto-topup settings: %s", err.Error()), nil)
	}
	return &settings, nil
}

// GetNotificationPreferences retrieves billing notification preferences for an organization
func GetNotificationPreferences(orgID string) (*NotificationPreferences, error) {
	var resp apiResponse
	if err := makeRequest("GET", fmt.Sprintf("/billing/organizations/%s/notifications", orgID), nil, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var prefs NotificationPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal notification preferences: %s", err.Error()), nil)
	}
	return &prefs, nil
}

// UpdateNotificationPreferences updates billing notification preferences for an organization
func UpdateNotificationPreferences(orgID string, req NotificationPreferencesRequest) (*NotificationPreferences, error) {
	var resp apiResponse
	if err := makeRequest("PUT", fmt.Sprintf("/billing/organizations/%s/notifications", orgID), req, &resp); err != nil {
		return nil, err
	}
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response: %s", err.Error()), nil)
	}
	var prefs NotificationPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal notification preferences: %s", err.Error()), nil)
	}
	return &prefs, nil
}
