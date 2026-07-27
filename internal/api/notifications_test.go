package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotificationRequestsUseCanonicalOrganizationPaths(t *testing.T) {
	const (
		orgID   = "62338977-2dc7-4698-86c3-3b83d1c4be80"
		notifID = "c685d7c6-19ad-4b48-a94a-da80255d24c6"
		base    = "/v1/cli/organizations/" + orgID + "/notifications"
	)

	tests := []struct {
		name     string
		method   string
		path     string
		rawQuery string
		response string
		invoke   func() error
	}{
		{
			name: "list", method: http.MethodGet, path: base, rawQuery: "unread=true&limit=25",
			response: `{"error":false,"data":[]}`,
			invoke: func() error {
				_, err := GetNotifications(orgID, true, 25)
				return err
			},
		},
		{
			name: "unread count", method: http.MethodGet, path: base + "/unread-count",
			response: `{"error":false,"data":{"count":3}}`,
			invoke: func() error {
				_, err := GetUnreadCount(orgID)
				return err
			},
		},
		{
			name: "mark read", method: http.MethodPatch, path: base + "/" + notifID + "/read",
			invoke: func() error { return MarkNotificationAsRead(orgID, notifID) },
		},
		{
			name: "mark all read", method: http.MethodPost, path: base + "/mark-all-read",
			invoke: func() error { return MarkAllNotificationsAsRead(orgID) },
		},
		{
			name: "delete", method: http.MethodDelete, path: base + "/" + notifID,
			invoke: func() error { return DeleteNotification(orgID, notifID) },
		},
		{
			name: "get", method: http.MethodGet, path: base + "/" + notifID,
			response: `{"error":false,"data":{"notification_id":"` + notifID + `","organization_id":"` + orgID + `"}}`,
			invoke: func() error {
				_, err := GetNotification(orgID, notifID)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != tt.method {
					t.Errorf("method = %s, want %s", request.Method, tt.method)
				}
				if request.URL.Path != tt.path {
					t.Errorf("path = %s, want %s", request.URL.Path, tt.path)
				}
				if request.URL.RawQuery != tt.rawQuery {
					t.Errorf("query = %q, want %q", request.URL.RawQuery, tt.rawQuery)
				}
				if tt.response != "" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprint(w, tt.response)
				}
			}))
			defer server.Close()

			configureAdminAPITestContext(t, server.URL+"/v1/cli")
			if err := tt.invoke(); err != nil {
				t.Fatalf("request error = %v", err)
			}
		})
	}
}
