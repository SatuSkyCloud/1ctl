package admin

import (
	"context"
	"strings"
	"testing"

	"1ctl/internal/api"

	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func TestDeploymentAdoptRequiredFlags(t *testing.T) {
	assertRequiredAdoptionFlags(t, findDeploymentCommand(t, "adopt"), false)
}

func TestDeploymentRoutingAdoptRequiredFlags(t *testing.T) {
	assertRequiredAdoptionFlags(t, findDeploymentCommand(t, "routing-adopt"), true)
}

func assertRequiredAdoptionFlags(t *testing.T, command *cli.Command, requireYes bool) {
	t.Helper()
	required := map[string]bool{
		flagReason: true, flagExpectedUID: true, flagExpectedResourceVersion: true,
		flagExpectedGeneration: true, flagRequestID: true,
	}
	if requireYes {
		required[flagYes] = true
	}
	for _, flag := range command.Flags {
		switch typed := flag.(type) {
		case *cli.StringFlag:
			if required[typed.Name] {
				if !typed.Required || typed.Destination == nil {
					t.Errorf("flag %q must be required and wired to a destination", typed.Name)
				}
				delete(required, typed.Name)
			}
		case *cli.Int64Flag:
			if required[typed.Name] {
				if !typed.Required || typed.Destination == nil {
					t.Errorf("flag %q must be required and wired to a destination", typed.Name)
				}
				delete(required, typed.Name)
			}
		case *cli.BoolFlag:
			if required[typed.Name] {
				if !typed.Required || typed.Destination == nil {
					t.Errorf("flag %q must be required and wired to a destination", typed.Name)
				}
				delete(required, typed.Name)
			}
		}
	}
	if len(required) != 0 {
		t.Fatalf("missing required flags: %v", required)
	}
}

func TestDeploymentAdoptCommandValidatesInput(t *testing.T) {
	validID := uuid.NewString()
	base := []string{
		"admin", "deployment", "adopt", validID,
		"--reason", "canary",
		"--expected-uid", uuid.NewString(),
		"--expected-resource-version", "12",
		"--expected-generation", "3",
		"--request-id", uuid.NewString(),
		"--yes",
	}

	tests := []struct {
		name    string
		mutate  func([]string) []string
		wantErr string
	}{
		{
			name: "invalid deployment ID",
			mutate: func(args []string) []string {
				args[3] = "not-a-uuid"
				return args
			},
			wantErr: "invalid deployment ID",
		},
		{
			name: "invalid expected UID",
			mutate: func(args []string) []string {
				return replaceFlagValue(args, "--expected-uid", "not-a-uuid")
			},
			wantErr: "non-zero UUID",
		},
		{
			name: "invalid generation",
			mutate: func(args []string) []string {
				return replaceFlagValue(args, "--expected-generation", "0")
			},
			wantErr: "at least 1",
		},
		{
			name: "missing reason",
			mutate: func(args []string) []string {
				return removeFlag(args, "--reason")
			},
			wantErr: "Required flag",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := test.mutate(append([]string(nil), base...))
			err := Command().Run(context.Background(), args)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Run() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestDeploymentAdoptCommandWiresRequest(t *testing.T) {
	deploymentID := uuid.NewString()
	expectedUID := uuid.NewString()
	requestID := uuid.NewString()
	original := adoptDeployment
	t.Cleanup(func() { adoptDeployment = original })

	called := false
	adoptDeployment = func(gotDeploymentID, gotRequestID string, request api.DeploymentAdoptionRequest) (*api.DeploymentAdoptionResult, error) {
		called = true
		if gotDeploymentID != deploymentID || gotRequestID != requestID {
			t.Errorf("IDs = %q, %q", gotDeploymentID, gotRequestID)
		}
		want := api.DeploymentAdoptionRequest{
			Reason: "legacy canary", ExpectedUID: expectedUID,
			ExpectedResourceVersion: "51", ExpectedGeneration: 4,
		}
		if request != want {
			t.Errorf("request = %+v, want %+v", request, want)
		}
		return &api.DeploymentAdoptionResult{
			DeploymentID: deploymentID, AppLabel: "canary", Namespace: "tenant-a",
			ReconciliationState: "pending", ReconciliationManaged: true,
			FieldManager: "satusky-workload-controller", RequestID: requestID,
		}, nil
	}

	err := Command().Run(context.Background(), []string{
		"admin", "deployment", "adopt", deploymentID,
		"--reason", " legacy canary ",
		"--expected-uid", expectedUID,
		"--expected-resource-version", "51",
		"--expected-generation", "4",
		"--request-id", requestID,
		"--yes",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("adoption API was not called")
	}
}

func TestDeploymentRoutingAdoptCommandValidatesInput(t *testing.T) {
	validID := uuid.NewString()
	base := []string{
		"admin", "deployment", "routing-adopt", validID,
		"--reason", "routing cutover",
		"--expected-uid", uuid.NewString(),
		"--expected-resource-version", "24",
		"--expected-generation", "2",
		"--request-id", uuid.NewString(),
		"--yes",
	}

	tests := []struct {
		name    string
		mutate  func([]string) []string
		wantErr string
	}{
		{
			name: "invalid deployment ID",
			mutate: func(args []string) []string {
				args[3] = "not-a-uuid"
				return args
			},
			wantErr: "invalid deployment ID",
		},
		{
			name: "invalid expected UID",
			mutate: func(args []string) []string {
				return replaceFlagValue(args, "--expected-uid", "not-a-uuid")
			},
			wantErr: "non-zero UUID",
		},
		{
			name: "invalid generation",
			mutate: func(args []string) []string {
				return replaceFlagValue(args, "--expected-generation", "0")
			},
			wantErr: "at least 1",
		},
		{
			name: "missing request ID",
			mutate: func(args []string) []string {
				return removeFlag(args, "--request-id")
			},
			wantErr: "Required flag",
		},
		{
			name: "missing confirmation",
			mutate: func(args []string) []string {
				return removeBoolFlag(args, "--yes")
			},
			wantErr: "Required flag",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := test.mutate(append([]string(nil), base...))
			err := Command().Run(context.Background(), args)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Run() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestDeploymentRoutingAdoptCommandWiresRequest(t *testing.T) {
	deploymentID := uuid.NewString()
	expectedUID := uuid.NewString()
	requestID := uuid.NewString()
	original := adoptDeploymentRouting
	t.Cleanup(func() { adoptDeploymentRouting = original })

	called := false
	adoptDeploymentRouting = func(gotDeploymentID, gotRequestID string, request api.DeploymentAdoptionRequest) (*api.DeploymentRoutingAdoptionResult, error) {
		called = true
		if gotDeploymentID != deploymentID || gotRequestID != requestID {
			t.Errorf("IDs = %q, %q", gotDeploymentID, gotRequestID)
		}
		want := api.DeploymentAdoptionRequest{
			Reason: "routing cutover", ExpectedUID: expectedUID,
			ExpectedResourceVersion: "72", ExpectedGeneration: 6,
		}
		if request != want {
			t.Errorf("request = %+v, want %+v", request, want)
		}
		return &api.DeploymentRoutingAdoptionResult{
			DeploymentID: deploymentID, Name: "canary-route", Namespace: "tenant-a",
			FieldManager: "satusky-routing-controller", Force: true,
			AlreadyManaged: false, RequestID: requestID,
		}, nil
	}

	err := Command().Run(context.Background(), []string{
		"admin", "deployment", "routing-adopt", deploymentID,
		"--reason", " routing cutover ",
		"--expected-uid", expectedUID,
		"--expected-resource-version", "72",
		"--expected-generation", "6",
		"--request-id", requestID,
		"--yes",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("routing adoption API was not called")
	}
}

func findDeploymentCommand(t *testing.T, name string) *cli.Command {
	t.Helper()
	root := Command()
	if len(root.Commands) != 1 || root.Commands[0].Name != "deployment" {
		t.Fatalf("unexpected admin command tree")
	}
	deployment := root.Commands[0]
	for _, command := range deployment.Commands {
		if command.Name == name {
			return command
		}
	}
	t.Fatalf("deployment command %q not found", name)
	return nil
}

func replaceFlagValue(args []string, flag, value string) []string {
	for i := range args {
		if args[i] == flag && i+1 < len(args) {
			args[i+1] = value
			return args
		}
	}
	return args
}

func removeFlag(args []string, flag string) []string {
	for i := range args {
		if args[i] == flag && i+1 < len(args) {
			return append(args[:i], args[i+2:]...)
		}
	}
	return args
}

func removeBoolFlag(args []string, flag string) []string {
	for i := range args {
		if args[i] == flag {
			return append(args[:i], args[i+1:]...)
		}
	}
	return args
}
