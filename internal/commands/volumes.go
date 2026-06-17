package commands

import (
	"context"
	"fmt"

	"1ctl/internal/api"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v3"
)

func VolumesCommand() *cli.Command {
	return &cli.Command{
		Name:    "volumes",
		Aliases: []string{"volume"},
		Usage:   "Inspect, detach, and destroy persistent volumes",
		Commands: []*cli.Command{
			volumesListCommand(),
			volumesInspectCommand(),
			volumesDetachCommand(),
			volumesDestroyCommand(),
		},
	}
}

func volumesListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List persistent volumes for a deployment",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "deployment-id", Usage: "Deployment ID"},
			&cli.StringFlag{Name: "config", Usage: "Config name or path. Default: satusky.toml"},
		},
		Action: handleVolumesList,
	}
}

func volumesInspectCommand() *cli.Command {
	return &cli.Command{
		Name:      "inspect",
		Aliases:   []string{"get"},
		Usage:     "Inspect PVC and mount state for a volume",
		ArgsUsage: "<volume-id>",
		Action:    handleVolumesInspect,
	}
}

func volumesDetachCommand() *cli.Command {
	return &cli.Command{
		Name:      "detach",
		Usage:     "Detach a volume from its deployment without deleting the PVC",
		ArgsUsage: "<volume-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"},
		},
		Action: handleVolumesDetach,
	}
}

func volumesDestroyCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"destroy", "rm"},
		Usage:     "Detach and delete a persistent volume claim",
		ArgsUsage: "<volume-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"},
		},
		Action: handleVolumesDestroy,
	}
}

func handleVolumesList(ctx context.Context, cmd *cli.Command) error {
	deploymentID, err := resolveDeploymentID(cmd.String("deployment-id"), "", cmd.String("config"))
	if err != nil {
		return err
	}

	statuses, err := api.GetDeploymentVolumeLifecycleStatuses(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list volumes: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(statuses) {
		return nil
	}
	if len(statuses) == 0 {
		utils.PrintInfo("No persistent volumes found for deployment %s", deploymentID)
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

func handleVolumesInspect(ctx context.Context, cmd *cli.Command) error {
	volumeID, err := requiredVolumeID(cmd, "inspect")
	if err != nil {
		return err
	}

	status, err := api.GetVolumeLifecycleStatus(volumeID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to inspect volume: %s", err.Error()), nil)
	}
	printVolumeLifecycle(status)
	return nil
}

func handleVolumesDetach(ctx context.Context, cmd *cli.Command) error {
	volumeID, err := requiredVolumeID(cmd, "detach")
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Detach volume %s? The PVC will be retained.", volumeID), cmd.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}

	status, err := api.DetachVolume(volumeID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to detach volume: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Volume %s detached; PVC retained", volumeID)
	printVolumeLifecycle(status)
	return nil
}

func handleVolumesDestroy(ctx context.Context, cmd *cli.Command) error {
	volumeID, err := requiredVolumeID(cmd, "destroy")
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Destroy volume %s? This detaches the volume and deletes its PVC.", volumeID), cmd.Bool("yes")) {
		fmt.Println("Aborted.")
		return nil
	}

	status, err := api.DeleteVolumePVC(volumeID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to destroy volume: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Volume %s destroyed", volumeID)
	printVolumeLifecycle(status)
	return nil
}

func requiredVolumeID(cmd *cli.Command, action string) (string, error) {
	if cmd.NArg() < 1 {
		return "", utils.NewError(fmt.Sprintf("volume ID is required. Usage: 1ctl volumes %s <volume-id>", action), nil)
	}
	return cmd.Args().First(), nil
}

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
