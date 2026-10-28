package admin

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	"1ctl/internal/deploy"
	"1ctl/internal/utils"
)

var adoptDeployment = api.AdoptDeployment
var adoptDeploymentRouting = api.AdoptDeploymentRouting

// deploymentAdoptionRemoved explains why these commands no longer call the API.
//
// The adoption endpoints were removed deliberately -- domains/admin/http keeps a
// static test asserting they stay unregistered -- but the commands went on calling
// them, so every use failed with a bare "Cannot POST /admin/deployments/:id/adopt"
// while `1ctl admin --help` still advertised the operation
// (satusky-core_backend#633).
//
// The commands are kept rather than deleted so a script gets a reason instead of
// "unknown command", and so the seam the tests use to prove no call is made stays
// visible.
const deploymentAdoptionRemoved = "deployment adoption is no longer served by the API: " +
	"the adopt endpoints were deliberately unregistered, so this command cannot transfer " +
	"ownership of anything. Migrate or retire the deployment through durable reconciliation " +
	"instead -- see satusky-core_backend#633 for the current state of that path."

func handleDeploymentAdopt(_ context.Context, _ deploymentAdoptInput) error {
	return utils.NewError(deploymentAdoptionRemoved, nil)
}

func handleDeploymentRoutingAdopt(_ context.Context, _ deploymentAdoptInput) error {
	return utils.NewError(deploymentAdoptionRemoved, nil)
}

// RetireMachineStateDeadLetters is the injectable seam, so a test can prove the
// command reports what the API returned without a server.
var retireMachineStateDeadLetters = api.RetireMachineStateDeadLetters

// handleMachineStateRetire retires a dead-lettered machine-state event.
//
// EnsureReconciliationEvent refuses to queue while a dead-letter exists for the same
// (machine, generation), and a generation never advances on its own — so without this
// the machine can never be reconciled again. In production that kept two of five nodes
// out of the placement pool (satusky-core_backend#660).
func handleMachineStateRetire(_ context.Context, in machineStateRetireInput) error {
	result, err := retireMachineStateDeadLetters(in.MachineID, in.Generation)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to retire dead-lettered machine state events: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(result) {
		return nil
	}
	// Constant format: the count travels as a status line so the printf checker can
	// verify this call site.
	utils.PrintSuccess("Machine state dead-letters retired")
	utils.PrintStatusLine("Retired", fmt.Sprintf("%d", result.Retired))
	utils.PrintStatusLine("Machine", result.MachineID)
	utils.PrintStatusLine("Generation", fmt.Sprintf("%d", result.Generation))
	if result.Retired == 0 {
		utils.PrintStatusLine("Note", "nothing was dead-lettered at that generation; reconciliation was already free to queue")
	}
	return nil
}

// Injectable seams so tests can prove what the move command sends.
var (
	moveDeploymentPlacement = api.MoveDeploymentPlacement
	resolveMoveDeploymentID = deploy.ResolveDeploymentID
	resolveMachineID        = func(name string) (string, error) {
		machine, err := api.GetMachineByName(name)
		if err != nil {
			return "", err
		}
		if machine == nil || machine.MachineID == "" {
			return "", fmt.Errorf("machine %q has no machine ID", name)
		}
		return machine.MachineID, nil
	}
	confirmMove = utils.Confirm
)

func handleDeploymentMove(_ context.Context, in deploymentMoveInput) error {
	deploymentID, err := resolveMoveDeploymentID(in.DeploymentID, in.App, "")
	if err != nil {
		return utils.NewError(err.Error(), nil)
	}
	fromID, err := resolveMachineID(in.From)
	if err != nil {
		return utils.NewError(fmt.Sprintf("resolve machine %q: %s", in.From, err.Error()), nil)
	}
	toID, err := resolveMachineID(in.To)
	if err != nil {
		return utils.NewError(fmt.Sprintf("resolve machine %q: %s", in.To, err.Error()), nil)
	}
	label := in.App
	if label == "" {
		label = deploymentID
	}
	if !confirmMove(fmt.Sprintf("Move %s from %s to %s? Its pods will roll to the new machine", label, in.From, in.To), in.Yes) {
		return utils.NewError("move cancelled", nil)
	}
	result, err := moveDeploymentPlacement(deploymentID, fromID, toID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to move deployment: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(result) {
		return nil
	}
	utils.PrintSuccess("Move accepted; the rolling update runs in the background")
	utils.PrintStatusLine("Deployment", deploymentID)
	utils.PrintStatusLine("From", fmt.Sprintf("%s (%s)", in.From, fromID))
	utils.PrintStatusLine("To", fmt.Sprintf("%s (%s)", in.To, toID))
	utils.PrintStatusLine("Generation", fmt.Sprintf("%d", result.DesiredGeneration))
	utils.PrintStatusLine("Next", "watch it with: 1ctl app status "+label)
	return nil
}
