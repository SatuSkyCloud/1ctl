package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Notification represents a notification
type Notification struct {
	ID             uuid.UUID              `json:"id"`
	OrganizationID uuid.UUID              `json:"organization_id"`
	UserID         uuid.UUID              `json:"user_id"`
	Type           string                 `json:"type"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Data           map[string]interface{} `json:"data,omitempty"`
	Read           bool                   `json:"read"`
	CreatedAt      time.Time              `json:"created_at"`
}

// UnreadCount represents unread notification count
type UnreadCount struct {
	Count int `json:"count"`
}

// GetNotifications gets notifications for an organization
func GetNotifications(orgID string, unreadOnly bool, limit int) ([]Notification, error) {
	path := fmt.Sprintf("/notifications/organizations/%s", orgID)
	params := []string{}
	if unreadOnly {
		params = append(params, "unread=true")
	}
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if len(params) > 0 {
		path = path + "?"
		for i, p := range params {
			if i > 0 {
				path += "&"
			}
			path += p
		}
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

	var notifications []Notification
	if err := json.Unmarshal(data, &notifications); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal notifications: %s", err.Error()), nil)
	}
	return notifications, nil
}

// GetUnreadCount gets unread notification count
func GetUnreadCount(orgID string) (int, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/notifications/organizations/%s/unread-count", orgID), nil, &resp)
	if err != nil {
		return 0, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return 0, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var result UnreadCount
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, utils.NewError(fmt.Sprintf("failed to unmarshal count: %s", err.Error()), nil)
	}
	return result.Count, nil
}

// MarkNotificationAsRead marks a notification as read
func MarkNotificationAsRead(orgID, notifID string) error {
	return makeRequest("PATCH", fmt.Sprintf("/notifications/organizations/%s/%s/read", orgID, notifID), nil, nil)
}

// MarkAllNotificationsAsRead marks all notifications as read
func MarkAllNotificationsAsRead(orgID string) error {
	return makeRequest("POST", fmt.Sprintf("/notifications/organizations/%s/mark-all-read", orgID), nil, nil)
}

// DeleteNotification deletes a notification
func DeleteNotification(orgID, notifID string) error {
	return makeRequest("DELETE", fmt.Sprintf("/notifications/organizations/%s/%s", orgID, notifID), nil, nil)
}

// GetNotification gets a single notification
func GetNotification(orgID, notifID string) (*Notification, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/notifications/organizations/%s/%s", orgID, notifID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var notification Notification
	if err := json.Unmarshal(data, &notification); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal notification: %s", err.Error()), nil)
	}
	return &notification, nil
}
