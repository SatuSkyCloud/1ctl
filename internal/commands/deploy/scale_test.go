package deploy

import (
	"errors"
	"testing"

	"1ctl/internal/api"
)

func TestSubmitScaleDeploymentUsesReplicaOnlyMutation(t *testing.T) {
	const requestID = "scale-request"
	var gotDeploymentID, gotRequestID string
	var gotReplicas int32
	accepted, err := submitScaleDeployment("deployment-1", 3, 7, requestID,
		func(deploymentID string, replicas int32, requestID string) (*api.DeploymentScaleAccepted, error) {
			gotDeploymentID, gotReplicas, gotRequestID = deploymentID, replicas, requestID
			return &api.DeploymentScaleAccepted{Data: api.DeploymentScaleResult{DeploymentID: deploymentID, Replicas: replicas}, DesiredGeneration: 8}, nil
		})
	if err != nil {
		t.Fatalf("submitScaleDeployment: %v", err)
	}
	if gotDeploymentID != "deployment-1" || gotReplicas != 3 || gotRequestID != requestID {
		t.Fatalf("scale mutation = deployment %q replicas %d request %q", gotDeploymentID, gotReplicas, gotRequestID)
	}
	if accepted.DesiredGeneration != 8 {
		t.Fatalf("desired generation = %d, want 8", accepted.DesiredGeneration)
	}
}

func TestSubmitScaleDeploymentRejectsInvalidAcceptance(t *testing.T) {
	tests := []struct {
		name     string
		accepted *api.DeploymentScaleAccepted
	}{
		{"nil", nil},
		{"deployment mismatch", &api.DeploymentScaleAccepted{Data: api.DeploymentScaleResult{DeploymentID: "other", Replicas: 3}, DesiredGeneration: 8}},
		{"replicas mismatch", &api.DeploymentScaleAccepted{Data: api.DeploymentScaleResult{DeploymentID: "deployment-1", Replicas: 2}, DesiredGeneration: 8}},
		{"stale generation", &api.DeploymentScaleAccepted{Data: api.DeploymentScaleResult{DeploymentID: "deployment-1", Replicas: 3}, DesiredGeneration: 7}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := submitScaleDeployment("deployment-1", 3, 7, "request-id",
				func(string, int32, string) (*api.DeploymentScaleAccepted, error) { return tt.accepted, nil })
			if err == nil {
				t.Fatal("submitScaleDeployment error = nil")
			}
		})
	}

	wantErr := errors.New("rejected")
	_, err := submitScaleDeployment("deployment-1", 3, 7, "request-id",
		func(string, int32, string) (*api.DeploymentScaleAccepted, error) { return nil, wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("submitScaleDeployment error = %v, want source error", err)
	}
}
