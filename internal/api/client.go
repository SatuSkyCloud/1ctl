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
	"time"

	"github.com/google/uuid"
)

// Common response structure that matches backend
type apiResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Count   int         `json:"count,omitempty"`
	Data    interface{} `json:"data"`
}

// DeleteDeployment deletes a deployment
func DeleteDeployment(req interface{}, deploymentID string) error {
	return makeRequest("POST", fmt.Sprintf("/deployments/delete/%s", deploymentID), req, nil)
}

// ListDeployments lists all deployments for current namespace
func ListDeployments() ([]Deployment, error) {
	namespace := context.GetCurrentNamespace()
	if namespace == "" {
		return nil, utils.NewError("no namespace selected", nil)
	}

	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/deployments/namespace/%s", namespace), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var deployments []Deployment
	if err := json.Unmarshal(data, &deployments); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal deployments: %s", err.Error()), nil)
	}
	return deployments, nil
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

// GetDeployment gets details for a specific deployment
func GetDeployment(deploymentID string) (*Deployment, error) {
	var resp apiResponse
	resp.Data = &Deployment{}
	err := makeRequest("GET", fmt.Sprintf("/deployments/id/%s", deploymentID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var deployment Deployment
	if err := json.Unmarshal(data, &deployment); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal deployment: %s", err.Error()), nil)
	}
	return &deployment, nil
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

func DeleteService(req interface{}, serviceID string) error {
	var resp apiResponse
	return makeRequest("POST", fmt.Sprintf("/services/delete/%s", serviceID), req, &resp)
}

// Secret methods
func CreateSecret(secret Secret) (*Secret, error) {
	var resp apiResponse
	var secretResp Secret
	resp.Data = &secretResp

	// Always use the current namespace
	secret.Namespace = context.GetCurrentNamespace()

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

func DeleteIngress(req interface{}, ingressID string) error {
	var resp apiResponse
	return makeRequest("POST", fmt.Sprintf("/ingresses/delete/%s", ingressID), req, &resp)
}

// Environment methods
func UpsertEnvironment(env Environment) (*Environment, error) {
	var resp apiResponse
	var envResp Environment
	resp.Data = &envResp

	// Always use the current namespace
	env.Namespace = context.GetCurrentNamespace()

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

func DeleteEnvironment(req interface{}, environmentID string) error {
	var resp apiResponse
	return makeRequest("POST", fmt.Sprintf("/environments/delete/%s", environmentID), req, &resp)
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

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to send request: %s", err.Error()), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
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

// GetDeploymentStatus gets the current status of a deployment
func GetDeploymentStatus(deploymentID string) (*DeploymentStatus, error) {
	var resp apiResponse
	resp.Data = &DeploymentStatus{}
	err := makeRequest("GET", fmt.Sprintf("/deployments/status/%s", deploymentID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var status DeploymentStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal status: %s", err.Error()), nil)
	}
	return &status, nil
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

		switch status.Status {
		case StatusCompleted:
			return status, nil
		case StatusFailed:
			return status, utils.NewError(fmt.Sprintf("deployment failed: %s", status.Message), nil)
		case StatusPending, StatusCreating, StatusRunning:
			utils.PrintInfo("Deployment status: %s (%d%%)\n", status.Status, status.Progress)
		default:
			return nil, utils.NewError(fmt.Sprintf("unknown deployment status: %s", status.Status), nil)
		}

		<-ticker.C
	}
}

// makeRequest is a helper function to make HTTP requests
func makeRequest(method, path string, body interface{}, response interface{}) error {
	config := config.GetConfig()
	url := fmt.Sprintf("%s%s", config.ApiURL, path)

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

	userConfigKey := context.GetUserConfigKey()
	// if userConfigKey == "" {
	// 	return utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate")
	// }

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-satusky-api-key", token)
	req.Header.Set("x-satusky-config", userConfigKey)
	// TODO: configure x-satusky-user-email for custom domain lets' encrypt

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to make request: %s", err.Error()), nil)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to read response body: %s", err.Error()), nil)
	}

	if resp.StatusCode >= 400 {
		var apiError APIError
		if err := json.Unmarshal(respBody, &apiError); err != nil {
			return utils.NewError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(respBody)), nil)
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

// Deployment methods
func GetDeploymentsByNamespace(namespace string) ([]Deployment, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/deployments/namespace/%s", namespace), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var deployments []Deployment
	if err := json.Unmarshal(data, &deployments); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal deployments: %s", err.Error()), nil)
	}
	return deployments, nil
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
	err := makeRequest("GET", fmt.Sprintf("/ingresses/domainName/%s", domainName), nil, &resp)
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
