package api

import (
	"encoding/json"
	"testing"
)

func TestCreateDatabaseUserResponseDecodesReadinessFields(t *testing.T) {
	payload := []byte(`{
		"data": {
			"username": "reporter",
			"secret_name": "cnpg-user-test-reporter"
		},
		"password": "secret",
		"ready": false,
		"reconciliation_status": "pending",
		"readiness_message": "CNPG reconciles managed database roles asynchronously"
	}`)

	var resp CreateDatabaseUserResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if resp.User.Username != "reporter" {
		t.Fatalf("User.Username = %q, want reporter", resp.User.Username)
	}
	if resp.Password != "secret" {
		t.Fatalf("Password = %q, want secret", resp.Password)
	}
	if resp.Ready {
		t.Fatal("Ready = true, want false")
	}
	if resp.ReconciliationStatus != "pending" {
		t.Fatalf("ReconciliationStatus = %q, want pending", resp.ReconciliationStatus)
	}
	if resp.ReadinessMessage == "" {
		t.Fatal("ReadinessMessage is empty")
	}
}
