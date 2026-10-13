package admin

import (
	"context"

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
