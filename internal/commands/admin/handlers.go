package admin

import (
	"context"
	"fmt"

	"1ctl/internal/api"
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
