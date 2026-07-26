package nats

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	deploypkg "1ctl/internal/deploy"
	"1ctl/internal/utils"

	"github.com/google/uuid"
)

const natsMarketplaceName = "nats"

func handleCreate(ctx context.Context, in createInput) error {
	if err := validateCreateInput(in); err != nil {
		return err
	}
	namespace, err := satuskyctx.GetCurrentNamespaceOrError()
	if err != nil {
		return err
	}
	app, err := api.ResolveMarketplaceApp(natsMarketplaceName)
	if err != nil {
		return err
	}
	if !app.Deployable {
		return utils.NewError("NATS is not currently deployable", nil)
	}

	replicas := int32(1)
	config := natsProfileConfig(false, in.StorageSize, in.StorageClass)
	req := api.MarketplaceDeployRequest{
		DeploymentName: in.Name,
		Replicas:       replicas,
		CPURequest:     in.CPU,
		MemoryRequest:  in.Memory,
		Values:         map[string]interface{}{"config": config},
	}
	if in.JetStream {
		replicas = 3
		req.Replicas = replicas
		req.StorageSize = in.StorageSize
		req.Values["config"] = natsProfileConfig(true, in.StorageSize, in.StorageClass)
	}

	response, err := api.DeployMarketplaceApp(namespace, app.MarketplaceID.String(), req)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create NATS deployment: %s", err.Error()), nil)
	}
	return deploypkg.ReportDeployResult(
		response.AppLabel,
		response.DeploymentID.String(),
		response.Domain,
		deploypkg.PublicURLReadiness{Ready: false, Reason: "deployment accepted; readiness was not verified"},
		"",
		true,
	)
}

func validateCreateInput(in createInput) error {
	if len(in.Name) > 63 || !natsNamePattern.MatchString(in.Name) {
		return utils.NewError("name must be 1-63 lowercase letters, numbers, or hyphens, and start and end with a letter or number", nil)
	}
	if !natsCPUPattern.MatchString(in.CPU) {
		return utils.NewError("--cpu must be a positive millicore value such as 250m", nil)
	}
	if !natsMemoryPattern.MatchString(in.Memory) {
		return utils.NewError("--memory must be a positive Mi or Gi value such as 256Mi", nil)
	}
	if !natsStoragePattern.MatchString(in.StorageSize) {
		return utils.NewError("--storage-size must be a positive Mi, Gi, or Ti value such as 10Gi", nil)
	}
	if len(in.StorageClass) > 253 || !natsStorageClassPattern.MatchString(in.StorageClass) {
		return utils.NewError("--storage-class must be a valid Kubernetes storage class name", nil)
	}
	if !in.JetStream && (in.StorageClass != "" || in.StorageSizeSet) {
		return utils.NewError("--storage-size and --storage-class require --jetstream", nil)
	}
	return nil
}

func natsProfileConfig(jetStream bool, storageSize, storageClass string) map[string]interface{} {
	replicas := 1
	if jetStream {
		replicas = 3
	}
	return map[string]interface{}{
		"cluster": map[string]interface{}{
			"enabled":  jetStream,
			"replicas": replicas,
		},
		"jetstream": map[string]interface{}{
			"enabled": jetStream,
			"fileStore": map[string]interface{}{
				"enabled": true,
				"pvc": map[string]interface{}{
					"enabled":          jetStream,
					"size":             storageSize,
					"storageClassName": storageClass,
				},
			},
		},
		"merge": map[string]interface{}{
			"authorization": map[string]interface{}{"token": "<< $NATS_TOKEN >>"},
		},
	}
}

func handleList(ctx context.Context) error {
	namespace, err := satuskyctx.GetCurrentNamespaceOrError()
	if err != nil {
		return err
	}
	deployments, err := api.ListDeploymentsByNamespace(namespace)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list NATS deployments: %s", err.Error()), nil)
	}
	natsDeployments := make([]api.Deployment, 0)
	for _, deployment := range deployments {
		if isNATSDeployment(deployment) {
			natsDeployments = append(natsDeployments, deployment)
		}
	}
	if utils.PrintListOrJSON(natsDeployments, "No NATS deployments found") {
		return nil
	}
	rows := make([][]string, 0, len(natsDeployments))
	for _, deployment := range natsDeployments {
		rows = append(rows, []string{
			deployment.AppLabel,
			deployment.DeploymentID.String(),
			fmt.Sprintf("%d", deployment.Replicas),
			deployment.Status,
			api.FormatTimeAgo(deployment.CreatedAt),
		})
	}
	utils.PrintTable([]string{"NAME", "DEPLOYMENT ID", "REPLICAS", "STATUS", "CREATED"}, rows)
	return nil
}

func handleGet(ctx context.Context, in deploymentInput) error {
	deployment, err := resolveNATSDeployment(in.Deployment)
	if err != nil {
		return err
	}
	if utils.TryPrintJSON(deployment) {
		return nil
	}
	utils.PrintHeader("NATS Deployment")
	printNATSDeployment(deployment)
	return nil
}

func handleStatus(ctx context.Context, in deploymentInput) error {
	deployment, err := resolveNATSDeployment(in.Deployment)
	if err != nil {
		return err
	}
	status, err := api.GetDeploymentStatus(deployment.DeploymentID.String())
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get NATS deployment status: %s", err.Error()), nil)
	}
	details := struct {
		Deployment *api.Deployment       `json:"deployment"`
		Status     *api.DeploymentStatus `json:"status"`
	}{deployment, status}
	if utils.TryPrintJSON(details) {
		return nil
	}
	utils.PrintHeader("NATS Status")
	utils.PrintStatusLine("Name", deployment.AppLabel)
	utils.PrintStatusLine("Deployment ID", deployment.DeploymentID.String())
	utils.PrintStatusLine("Workload", status.Status)
	if status.Message != "" {
		utils.PrintStatusLine("Message", status.Message)
	}
	utils.PrintStatusLine("Progress", fmt.Sprintf("%d%%", status.Progress))
	utils.PrintStatusLine("Replicas", fmt.Sprintf("%d desired", deployment.Replicas))
	return nil
}

func handleCredentials(ctx context.Context, in credentialsInput) error {
	if (in.OutputDir == "" && !in.Stdout) || (in.OutputDir != "" && in.Stdout) {
		return utils.NewError("choose exactly one of --output-dir or --stdout", nil)
	}
	deployment, err := resolveNATSDeployment(in.Deployment)
	if err != nil {
		return err
	}
	outputs := make(map[string][]byte, 2)
	for _, name := range []string{"client-url", "client-token"} {
		value, downloadErr := api.DownloadMarketplaceDeploymentOutput(deployment.DeploymentID.String(), name)
		if downloadErr != nil {
			return utils.NewError(fmt.Sprintf("failed to download NATS %s: %s", name, downloadErr.Error()), nil)
		}
		outputs[name] = value
	}
	if in.Stdout {
		fmt.Printf("client-url: %s\nclient-token: %s\n", outputs["client-url"], outputs["client-token"])
		return nil
	}
	if err := writeCredentialFiles(in.OutputDir, outputs); err != nil {
		return err
	}
	utils.PrintSuccess("NATS credentials written to %s", in.OutputDir)
	utils.PrintWarning("Credential files contain secrets; keep them private.")
	return nil
}

func writeCredentialFiles(outputDir string, outputs map[string][]byte) error {
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		return utils.NewError(fmt.Sprintf("failed to create credential directory: %s", err.Error()), nil)
	}
	for name, value := range outputs {
		path := filepath.Join(outputDir, name+".txt")
		if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return utils.NewError(fmt.Sprintf("refusing to overwrite symlink %s", path), nil)
		} else if err != nil && !os.IsNotExist(err) {
			return utils.NewError(fmt.Sprintf("failed to inspect credential file: %s", err.Error()), nil)
		}
		if err := os.WriteFile(path, value, 0600); err != nil {
			return utils.NewError(fmt.Sprintf("failed to write credential file: %s", err.Error()), nil)
		}
		if err := os.Chmod(path, 0600); err != nil {
			return utils.NewError(fmt.Sprintf("failed to secure credential file: %s", err.Error()), nil)
		}
	}
	return nil
}

func handleDelete(ctx context.Context, in deleteInput) error {
	deployment, err := resolveNATSDeployment(in.Deployment)
	if err != nil {
		return err
	}
	if in.Purge {
		utils.PrintWarning("Persistent NATS data will be permanently deleted.")
	} else {
		utils.PrintInfo("Persistent volumes are retained by default and may continue to incur storage charges.")
		utils.PrintInfo("Use --purge-retained to permanently remove retained NATS data.")
	}
	prompt := fmt.Sprintf("Delete NATS deployment %s?", deployment.AppLabel)
	if !utils.Confirm(prompt, in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	operation, err := api.DeleteDeployment(deployment.DeploymentID.String(), in.Purge)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete NATS deployment: %s", err.Error()), nil)
	}
	if !in.NoWait && !operation.IsTerminal() {
		operation, err = api.WaitForDeploymentDeletion(operation, 5*time.Minute)
		if err != nil {
			return utils.NewError(fmt.Sprintf("NATS deletion did not complete: %s", err.Error()), nil)
		}
	}
	if operation.IsTerminal() && !operation.IsSuccessful() {
		return utils.NewError(fmt.Sprintf("NATS deletion failed: %s", operation.Lifecycle.ErrorText()), nil)
	}
	if utils.TryPrintJSON(operation) {
		return nil
	}
	utils.PrintSuccess("NATS deletion %s", operation.Status)
	utils.PrintStatusLine("Deployment ID", operation.DeploymentID)
	utils.PrintStatusLine("Purge retained", fmt.Sprintf("%t", operation.PurgeRetained))
	return nil
}

func resolveNATSDeployment(reference string) (*api.Deployment, error) {
	var (
		deployment *api.Deployment
		err        error
	)
	if _, parseErr := uuid.Parse(reference); parseErr == nil {
		deployment, err = api.GetDeployment(reference)
	} else {
		namespace, namespaceErr := satuskyctx.GetCurrentNamespaceOrError()
		if namespaceErr != nil {
			return nil, namespaceErr
		}
		deployment, err = api.GetDeploymentByAppLabel(namespace, reference)
	}
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("NATS deployment %q not found: %s", reference, err.Error()), nil)
	}
	if !isNATSDeployment(*deployment) {
		return nil, utils.NewError(fmt.Sprintf("deployment %q is not a NATS marketplace deployment", reference), nil)
	}
	return deployment, nil
}

func isNATSDeployment(deployment api.Deployment) bool {
	return strings.EqualFold(strings.TrimSpace(deployment.MarketplaceAppName), natsMarketplaceName)
}

func printNATSDeployment(deployment *api.Deployment) {
	utils.PrintStatusLine("Name", deployment.AppLabel)
	utils.PrintStatusLine("Deployment ID", deployment.DeploymentID.String())
	utils.PrintStatusLine("Namespace", deployment.Namespace)
	utils.PrintStatusLine("Status", deployment.Status)
	utils.PrintStatusLine("Profile", map[bool]string{true: "JetStream HA", false: "Core"}[deployment.Replicas == 3])
	utils.PrintStatusLine("Replicas", fmt.Sprintf("%d", deployment.Replicas))
	utils.PrintStatusLine("CPU", deployment.CpuRequest)
	utils.PrintStatusLine("Memory", deployment.MemoryRequest)
	if deployment.StorageSize != "" {
		utils.PrintStatusLine("Storage", deployment.StorageSize)
	}
}
