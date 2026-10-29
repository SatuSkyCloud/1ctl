package admin

import (
	"context"
	"errors"
	"testing"

	"1ctl/internal/api"
)

func stubMoveSeams(t *testing.T, confirmed bool) *[]string {
	t.Helper()
	calls := []string{}
	oldMove, oldResolveDep, oldResolveMachine, oldConfirm := moveDeploymentPlacement, resolveMoveDeploymentID, resolveMachineID, confirmMove
	t.Cleanup(func() {
		moveDeploymentPlacement, resolveMoveDeploymentID, resolveMachineID, confirmMove = oldMove, oldResolveDep, oldResolveMachine, oldConfirm
	})
	resolveMoveDeploymentID = func(depID, app, _ string) (string, error) {
		if depID != "" {
			return depID, nil
		}
		return "dep-for-" + app, nil
	}
	resolveMachineID = func(name string) (string, error) { return "id-" + name, nil }
	confirmMove = func(string, bool) bool { return confirmed }
	moveDeploymentPlacement = func(deploymentID, from, to string) (*api.PlacementMoveResult, error) {
		calls = append(calls, deploymentID+" "+from+" "+to)
		return &api.PlacementMoveResult{DesiredGeneration: 9, Changed: true}, nil
	}
	return &calls
}

func TestDeploymentMoveSendsResolvedIDs(t *testing.T) {
	calls := stubMoveSeams(t, true)
	err := Command().Run(context.Background(), []string{"admin", "deployment", "move", "ims-dev", "--from", "compute-main-04", "--to", "compute-main-01", "--yes"})
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}
	if len(*calls) != 1 || (*calls)[0] != "dep-for-ims-dev id-compute-main-04 id-compute-main-01" {
		t.Fatalf("unexpected API calls: %v", *calls)
	}
}

func TestDeploymentMoveRejectsBadArgumentsWithoutCallingTheAPI(t *testing.T) {
	calls := stubMoveSeams(t, true)
	for _, args := range [][]string{
		{"admin", "deployment", "move", "--from", "a", "--to", "b"},                            // no app or ID
		{"admin", "deployment", "move", "x", "y", "--from", "a", "--to", "b"},                  // two apps
		{"admin", "deployment", "move", "x", "--from", "a", "--to", "a"},                       // same machine
		{"admin", "deployment", "move", "x", "--to", "b"},                                      // missing --from
		{"admin", "deployment", "move", "--deployment-id", "nope", "--from", "a", "--to", "b"}, // bad UUID
	} {
		if err := Command().Run(context.Background(), args); err == nil {
			t.Fatalf("%v returned nil error", args)
		}
	}
	if len(*calls) != 0 {
		t.Fatalf("API was called for invalid input: %v", *calls)
	}
}

func TestDeploymentMoveStopsWhenNotConfirmed(t *testing.T) {
	calls := stubMoveSeams(t, false)
	err := Command().Run(context.Background(), []string{"admin", "deployment", "move", "ims-dev", "--from", "a", "--to", "b"})
	if err == nil || len(*calls) != 0 {
		t.Fatalf("an unconfirmed move must not call the API (err=%v calls=%v)", err, *calls)
	}
}

func TestDeploymentMoveReportsMachineLookupFailure(t *testing.T) {
	calls := stubMoveSeams(t, true)
	resolveMachineID = func(string) (string, error) { return "", errors.New("not found") }
	if err := Command().Run(context.Background(), []string{"admin", "deployment", "move", "x", "--from", "a", "--to", "b", "--yes"}); err == nil {
		t.Fatal("an unknown machine must fail")
	}
	if len(*calls) != 0 {
		t.Fatalf("API was called after a failed lookup: %v", *calls)
	}
}
