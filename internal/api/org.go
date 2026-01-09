package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TeamMember represents a member of an organization team
type TeamMember struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	Role           string    `json:"role"`
	JoinedAt       time.Time `json:"joined_at"`
}

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AddTeamMemberRequest represents a request to add a team member
type AddTeamMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// UpdateTeamMemberRoleRequest represents a request to update team member role
type UpdateTeamMemberRoleRequest struct {
	Role string `json:"role"`
}

// GetUserOrganizations gets all organizations for the current user
func GetUserOrganizations() ([]Organization, error) {
	var resp apiResponse
	err := makeRequest("GET", "/organizations/user", nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var orgs []Organization
	if err := json.Unmarshal(data, &orgs); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal organizations: %s", err.Error()), nil)
	}
	return orgs, nil
}

// CreateOrganization creates a new organization
func CreateOrganization(req CreateOrganizationRequest) (*Organization, error) {
	var resp apiResponse
	err := makeRequest("POST", "/organizations/create", req, &resp)
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

// DeleteOrganization deletes an organization
func DeleteOrganization(orgID string) error {
	return makeRequest("POST", fmt.Sprintf("/organizations/delete/%s", orgID), nil, nil)
}

// GetOrganizationTeam gets team members for an organization
func GetOrganizationTeam(orgID string) ([]TeamMember, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/organizations/id/%s/team", orgID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var members []TeamMember
	if err := json.Unmarshal(data, &members); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal team members: %s", err.Error()), nil)
	}
	return members, nil
}

// AddTeamMember adds a team member to an organization
func AddTeamMember(orgID string, req AddTeamMemberRequest) (*TeamMember, error) {
	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/organizations/id/%s/team/add", orgID), req, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var member TeamMember
	if err := json.Unmarshal(data, &member); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal team member: %s", err.Error()), nil)
	}
	return &member, nil
}

// UpdateTeamMemberRole updates a team member's role
func UpdateTeamMemberRole(orgID, orgUserID, role string) error {
	req := UpdateTeamMemberRoleRequest{Role: role}
	return makeRequest("PUT", fmt.Sprintf("/organizations/id/%s/team/%s/role", orgID, orgUserID), req, nil)
}

// RemoveTeamMember removes a team member from an organization
func RemoveTeamMember(orgID, orgUserID string) error {
	return makeRequest("DELETE", fmt.Sprintf("/organizations/id/%s/team/%s", orgID, orgUserID), nil, nil)
}
