package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CLIToken represents an API token for CLI operations
type CLIToken struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	OrgID      uuid.UUID  `json:"org_id"`
	Name       string     `json:"name"`
	Token      string     `json:"token,omitempty"` // Only returned on creation
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Enabled    bool       `json:"enabled"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateTokenRequest represents request to create a token
type CreateTokenRequest struct {
	Name      string `json:"name"`
	ExpiresIn int    `json:"expires_in,omitempty"` // in days
}

// TokenStateRequest represents request to enable/disable token
type TokenStateRequest struct {
	Enabled bool `json:"enabled"`
}

// DeleteTokenRequest represents request to delete a token
type DeleteTokenRequest struct {
	TokenID string `json:"token_id"`
}

// GetCLITokens gets all API tokens for user
func GetCLITokens(userID, orgID string) ([]CLIToken, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/api-tokens/list/%s/%s", userID, orgID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var tokens []CLIToken
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal tokens: %s", err.Error()), nil)
	}
	return tokens, nil
}

// CreateCLIToken creates a new API token
func CreateCLIToken(userID, orgID string, req CreateTokenRequest) (*CLIToken, error) {
	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/api-tokens/create/%s/%s", userID, orgID), req, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var token CLIToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal token: %s", err.Error()), nil)
	}
	return &token, nil
}

// GetCLIToken gets a specific API token
func GetCLIToken(userID, tokenID string) (*CLIToken, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/api-tokens/%s/%s", userID, tokenID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var token CLIToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal token: %s", err.Error()), nil)
	}
	return &token, nil
}

// SetCLITokenState enables or disables an API token
func SetCLITokenState(userID, tokenID string, enabled bool) error {
	req := TokenStateRequest{Enabled: enabled}
	return makeRequest("POST", fmt.Sprintf("/api-tokens/state/%s/%s", userID, tokenID), req, nil)
}

// DeleteCLIToken deletes an API token
func DeleteCLIToken(userID, orgID, tokenID string) error {
	req := DeleteTokenRequest{TokenID: tokenID}
	return makeRequest("POST", fmt.Sprintf("/api-tokens/delete/%s/%s", userID, orgID), req, nil)
}
