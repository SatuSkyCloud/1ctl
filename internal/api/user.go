package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"
)

// CLIUserProfile represents user profile information for CLI operations
type CLIUserProfile struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	Name         *string   `json:"name,omitempty"`
	Organization string    `json:"organization,omitempty"`
	Role         string    `json:"role,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserPermission represents a single permission entry returned by the backend.
type UserPermission struct {
	Name        string `json:"permission_name"`
	Description string `json:"permission_description"`
	Resource    string `json:"resource_type"`
	Action      string `json:"action"`
}

// UpdateUserRequest represents request to update user profile
type UpdateUserRequest struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// ChangePasswordRequest represents request to change password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// GetCurrentUser gets the current user's profile
func GetCurrentUser() (*CLIUserProfile, error) {
	// This endpoint returns fields at top level (not wrapped in data)
	var user CLIUserProfile
	err := makeRequest("GET", "/users/profile", nil, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates user profile
func UpdateUser(userID string, req UpdateUserRequest) (*CLIUserProfile, error) {
	var resp apiResponse
	err := makeRequest("PUT", fmt.Sprintf("/users/%s", userID), req, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var user CLIUserProfile
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal user profile: %s", err.Error()), nil)
	}
	return &user, nil
}

// ChangePassword changes user password
func ChangePassword(currentPassword, newPassword string) error {
	req := ChangePasswordRequest{
		CurrentPassword: currentPassword,
		NewPassword:     newPassword,
	}
	return makeRequest("POST", "/auth/change-password", req, nil)
}

// GetUserPermissions gets user permissions for an organization
func GetUserPermissions(orgID string) ([]UserPermission, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/users/permissions/%s", orgID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var perms []UserPermission
	if err := json.Unmarshal(data, &perms); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal permissions: %s", err.Error()), nil)
	}
	return perms, nil
}

// RevokeAllSessions revokes all user sessions
func RevokeAllSessions() error {
	return makeRequest("POST", "/auth/revoke-all", nil, nil)
}
