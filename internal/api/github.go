package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GitHubConnection represents the GitHub connection status
type GitHubConnection struct {
	Connected      bool      `json:"connected"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	AvatarURL      string    `json:"avatar_url"`
	ConnectedAt    time.Time `json:"connected_at"`
	TokenExpiry    time.Time `json:"token_expiry"`
	AppInstalled   bool      `json:"app_installed"`
	InstallationID int64     `json:"installation_id"`
}

// GitHubRepository represents a GitHub repository
type GitHubRepository struct {
	ID            uuid.UUID `json:"id"`
	GitHubID      int64     `json:"github_id"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Private       bool      `json:"private"`
	Description   string    `json:"description"`
	DefaultBranch string    `json:"default_branch"`
	CloneURL      string    `json:"clone_url"`
	HTMLURL       string    `json:"html_url"`
	Language      string    `json:"language"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GitHubInstallationInfo represents GitHub App installation info
type GitHubInstallationInfo struct {
	InstallationID int64             `json:"installation_id"`
	AccountLogin   string            `json:"account_login"`
	AccountType    string            `json:"account_type"`
	Permissions    map[string]string `json:"permissions"`
	CreatedAt      time.Time         `json:"created_at"`
}

// GitHubDeployRequest represents a deploy from GitHub request
type GitHubDeployRequest struct {
	RepositoryID   string   `json:"repository_id"`
	Branch         string   `json:"branch"`
	Namespace      string   `json:"namespace"`
	CPUCores       string   `json:"cpu_cores"`
	Memory         string   `json:"memory"`
	Hostnames      []string `json:"hostnames"`
	Port           int32    `json:"port"`
	DockerfilePath string   `json:"dockerfile_path"`
}

// GetGitHubConnection gets the GitHub connection status
func GetGitHubConnection() (*GitHubConnection, error) {
	var resp apiResponse
	err := makeRequest("GET", "/github/connection", nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var conn GitHubConnection
	if err := json.Unmarshal(data, &conn); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal connection: %s", err.Error()), nil)
	}
	return &conn, nil
}

// ConnectGitHub initiates GitHub OAuth connection
func ConnectGitHub() (string, error) {
	var resp apiResponse
	err := makeRequest("POST", "/github/connect", nil, &resp)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var result struct {
		AuthURL string `json:"auth_url"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to unmarshal response: %s", err.Error()), nil)
	}
	return result.AuthURL, nil
}

// DisconnectGitHub disconnects GitHub account
func DisconnectGitHub() error {
	return makeRequest("POST", "/github/disconnect", nil, nil)
}

// GetGitHubRepositories gets list of GitHub repositories
func GetGitHubRepositories(page, limit int) ([]GitHubRepository, error) {
	path := "/github/repositories"
	if page > 0 || limit > 0 {
		path = fmt.Sprintf("%s?page=%d&limit=%d", path, page, limit)
	}

	var resp apiResponse
	err := makeRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var repos []GitHubRepository
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal repositories: %s", err.Error()), nil)
	}
	return repos, nil
}

// SyncGitHubRepositories syncs repositories from GitHub
func SyncGitHubRepositories() error {
	return makeRequest("POST", "/github/repositories/sync", nil, nil)
}

// GetGitHubRepository gets a specific repository by ID
func GetGitHubRepository(repoID string) (*GitHubRepository, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/github/repositories/%s", repoID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var repo GitHubRepository
	if err := json.Unmarshal(data, &repo); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal repository: %s", err.Error()), nil)
	}
	return &repo, nil
}

// GetGitHubInstallationInfo gets GitHub App installation info
func GetGitHubInstallationInfo() (*GitHubInstallationInfo, error) {
	var resp apiResponse
	err := makeRequest("GET", "/github/installation/info", nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var info GitHubInstallationInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal installation info: %s", err.Error()), nil)
	}
	return &info, nil
}

// RevokeGitHubInstallation revokes GitHub App installation
func RevokeGitHubInstallation() error {
	return makeRequest("DELETE", "/github/installation", nil, nil)
}

// DeployFromGitHub creates a deployment from a GitHub repository
func DeployFromGitHub(req GitHubDeployRequest) (*Deployment, error) {
	var resp apiResponse
	err := makeRequest("POST", "/github/deployments", req, &resp)
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
