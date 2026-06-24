package volumes

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	"1ctl/internal/deploy"
	"1ctl/internal/utils"
)

// --- Handlers -----------------------------------------------------------

func handleVolumesList(ctx context.Context, in volumesListInput) error {
	deploymentID, err := deploy.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}

	statuses, err := api.GetDeploymentVolumeLifecycleStatuses(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list volumes: %s", err.Error()), nil)
	}
	if utils.PrintListOrJSON(statuses, "No persistent volumes found for this deployment") {
		return nil
	}

	headers := []string{"VOLUME ID", "NAME", "PVC", "PVC STATUS", "MOUNT", "DESTROY POLICY"}
	rows := make([][]string, 0, len(statuses))
	for _, status := range statuses {
		rows = append(rows, []string{
			status.Volume.VolumeID.String(),
			status.Volume.VolumeName,
			status.PVC.Name,
			pvcStatusText(status.PVC),
			mountStatusText(status.Mount),
			status.DestroyPolicy,
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleVolumesInspect(ctx context.Context, in volumesActionInput) error {
	volumeID := in.VolumeID

	// --- Path 1: Volume name + app/deployment scope ---
	if in.VolumeName != "" {
		app := in.App
		depID := in.DeploymentID
		if app == "" && depID == "" {
			// Try positional arg as app name and volume name as the remaining
			// but we already have in.VolumeName from positional. We need --app or --deployment-id.
			return utils.NewError("use --app to specify which app when inspecting by volume name", nil)
		}
		deploymentID, err := deploy.ResolveDeploymentID(depID, app, in.Config)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to resolve deployment: %s", err.Error()), nil)
		}
		statuses, err := api.GetDeploymentVolumeLifecycleStatuses(deploymentID)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to list volumes: %s", err.Error()), nil)
		}
		found := false
		for _, s := range statuses {
			if s.Volume.VolumeName == in.VolumeName || s.Volume.ClaimName == in.VolumeName {
				volumeID = s.Volume.VolumeID.String()
				found = true
				break
			}
		}
		if !found {
			return utils.NewError(fmt.Sprintf("volume %q not found for this deployment", in.VolumeName), nil)
		}
	}

	// --- Path 2: App/deployment only (no volume name) ---
	if volumeID == "" && (in.App != "" || in.DeploymentID != "") {
		deploymentID, err := deploy.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
		if err != nil {
			return err
		}
		statuses, err := api.GetDeploymentVolumeLifecycleStatuses(deploymentID)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to list volumes: %s", err.Error()), nil)
		}
		switch len(statuses) {
		case 0:
			return utils.NewError("no volumes found for this deployment", nil)
		case 1:
			volumeID = statuses[0].Volume.VolumeID.String()
		default:
			return utils.NewError(fmt.Sprintf("deployment has %d volumes — specify volume name or --volume-id", len(statuses)), nil)
		}
	}

	if volumeID == "" {
		return utils.NewError("provide --volume-id, --app, or --deployment-id", nil)
	}

	status, err := api.GetVolumeLifecycleStatus(volumeID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to inspect volume: %s", err.Error()), nil)
	}
	printVolumeLifecycle(status)
	return nil
}

func handleVolumesDetach(ctx context.Context, in volumesActionInput) error {
	if !utils.Confirm(fmt.Sprintf("Detach volume %s? The PVC will be retained.", in.VolumeID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}

	status, err := api.DetachVolume(in.VolumeID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to detach volume: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Volume %s detached; PVC retained", in.VolumeID)
	printVolumeLifecycle(status)
	return nil
}

func handleVolumesDestroy(ctx context.Context, in volumesActionInput) error {
	if !utils.Confirm(fmt.Sprintf("Destroy volume %s? This detaches the volume and deletes its PVC.", in.VolumeID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}

	status, err := api.DeleteVolumePVC(in.VolumeID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to destroy volume: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Volume %s destroyed", in.VolumeID)
	printVolumeLifecycle(status)
	return nil
}

// --- Shared helpers -----------------------------------------------------

func printVolumeLifecycle(status *api.VolumeLifecycleStatus) {
	if status == nil {
		return
	}
	if utils.TryPrintJSON(status) {
		return
	}

	utils.PrintHeader("Volume %s", status.Volume.VolumeName)
	utils.PrintStatusLine("Volume ID", status.Volume.VolumeID.String())
	utils.PrintStatusLine("Deployment ID", status.Volume.DeploymentID.String())
	utils.PrintStatusLine("Claim", status.Volume.ClaimName)
	utils.PrintStatusLine("Size", status.Volume.StorageSize)
	utils.PrintStatusLine("Mount path", status.Volume.MountPath)
	utils.PrintStatusLine("PVC", pvcStatusText(status.PVC))
	if status.PVC.Message != "" {
		utils.PrintStatusLine("PVC message", status.PVC.Message)
	}
	utils.PrintStatusLine("Mount", mountStatusText(status.Mount))
	if status.Mount.Message != "" {
		utils.PrintStatusLine("Mount message", status.Mount.Message)
	}
	utils.PrintStatusLine("Destroy policy", status.DestroyPolicy)
}

func pvcStatusText(status api.VolumePVCStatus) string {
	if !status.Exists {
		if status.Message != "" {
			return "missing: " + status.Message
		}
		return "missing"
	}
	if status.Phase != "" {
		return status.Phase
	}
	return "exists"
}

func mountStatusText(status api.VolumeMountStatus) string {
	switch {
	case status.Attached && status.Mounted:
		if status.Path != "" {
			return "mounted at " + status.Path
		}
		return "mounted"
	case status.Attached:
		return "attached"
	default:
		return "detached"
	}
}
