package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CLIUserProfile represents user profile information for CLI operations
type CLIUserProfile struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserPermissions represents user permissions for an organization
type UserPermissions struct {
	UserID         uuid.UUID `json:"user_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Role           string    `json:"role"`
	Permissions    []string  `json:"permissions"`
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
	var resp apiResponse
	err := makeRequest("GET", "/auth/me", nil, &resp)
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
func GetUserPermissions(orgID string) (*UserPermissions, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/users/permissions/%s", orgID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var perms UserPermissions
	if err := json.Unmarshal(data, &perms); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal permissions: %s", err.Error()), nil)
	}
	return &perms, nil
}

// RevokeAllSessions revokes all user sessions
func RevokeAllSessions() error {
	return makeRequest("POST", "/auth/revoke-all", nil, nil)
}
