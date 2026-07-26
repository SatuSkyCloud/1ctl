package valkey

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

var valkeyNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

var maxmemoryPolicies = map[string]struct{}{
	"noeviction":      {},
	"allkeys-lru":     {},
	"allkeys-lfu":     {},
	"allkeys-random":  {},
	"volatile-lru":    {},
	"volatile-lfu":    {},
	"volatile-random": {},
	"volatile-ttl":    {},
}

func handleCreate(_ context.Context, in createInput) error {
	in.Name = strings.ToLower(strings.TrimSpace(in.Name))
	in.Topology = strings.ToLower(strings.TrimSpace(in.Topology))
	in.AppendFsync = strings.ToLower(strings.TrimSpace(in.AppendFsync))
	in.MaxmemoryPolicy = strings.ToLower(strings.TrimSpace(in.MaxmemoryPolicy))
	if err := validateCreate(in); err != nil {
		return err
	}
	if in.Instances == 0 {
		if in.Topology == "replicated" {
			in.Instances = 3
		} else {
			in.Instances = 1
		}
	}

	storageClass := in.StorageClass
	if storageClass == "" {
		classes, err := api.ListStorageClasses()
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to list storage classes: %s. Provide --storage-class explicitly.", err.Error()), nil)
		}
		for _, class := range classes {
			if class.IsDefault {
				storageClass = class.Name
				break
			}
		}
		if storageClass == "" {
			return utils.NewError("no default storage class found. Provide --storage-class explicitly.", nil)
		}
	}

	instance, err := api.CreateValkey(api.ValkeyCreateOptions{
		Name:             in.Name,
		Topology:         in.Topology,
		Instances:        in.Instances,
		Persistence:      in.Persistence,
		StorageSize:      in.StorageSize,
		StorageClass:     storageClass,
		CPURequest:       in.CPURequest,
		CPULimit:         in.CPULimit,
		MemoryRequest:    in.MemoryRequest,
		MemoryLimit:      in.MemoryLimit,
		AppendOnly:       in.AppendOnly,
		AppendFsync:      in.AppendFsync,
		MaxmemoryPolicy:  in.MaxmemoryPolicy,
		MaxmemoryPercent: in.MaxmemoryPercent,
		MetricsEnabled:   in.MetricsEnabled,
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create Valkey service: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(instance) {
		return nil
	}
	utils.PrintSuccess("Valkey service creation started")
	printInstance(instance)
	utils.PrintInfo("Use '1ctl valkey status %s' to watch readiness", instance.StorageID.String())
	utils.PrintInfo("Valkey endpoints are private to workloads in this cluster.")
	return nil
}

func handleList(_ context.Context) error {
	instances, err := api.ListValkey("")
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list Valkey services: %s", err.Error()), nil)
	}
	if utils.PrintListOrJSON(instances, "No Valkey services found") {
		return nil
	}
	rows := make([][]string, 0, len(instances))
	for _, instance := range instances {
		rows = append(rows, []string{
			instance.StorageID.String(),
			stringValue(instance.ClusterName),
			instance.Version,
			topology(instance),
			fmt.Sprintf("%d", effectiveInstances(instance)),
			instance.StorageSize,
			instance.StorageClass,
		})
	}
	utils.PrintTable([]string{"ID", "NAME", "VERSION", "TOPOLOGY", "INSTANCES", "SIZE", "CLASS"}, rows)
	return nil
}

func handleGet(_ context.Context, in storageInput) error {
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	instance, err := api.GetValkey(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Valkey service: %s", err.Error()), nil)
	}
	if instance.Engine != api.StorageEngineValkey {
		return utils.NewError(fmt.Sprintf("storage %s is not a Valkey service", storageID), nil)
	}
	if utils.TryPrintJSON(instance) {
		return nil
	}
	printInstance(instance)
	return nil
}

func handleStatus(_ context.Context, in storageInput) error {
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	status, err := api.GetValkeyStatus(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Valkey status: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(status) {
		return nil
	}
	utils.PrintHeader("Valkey Status")
	utils.PrintStatusLine("Status", status.Status)
	utils.PrintStatusLine("Workload", status.WorkloadKind)
	utils.PrintStatusLine("Instances", fmt.Sprintf("%d/%d ready", status.ReadyInstances, status.DesiredInstances))
	if status.Primary != "" {
		utils.PrintStatusLine("Primary", status.Primary)
	}
	utils.PrintStatusLine("Private host", status.PrivateHost)
	if status.ReadOnlyHost != "" {
		utils.PrintStatusLine("Private read-only host", status.ReadOnlyHost)
	}
	utils.PrintStatusLine("Persistence", boolStatus(status.PersistenceEnabled))
	utils.PrintStatusLine("Metrics", boolStatus(status.MetricsEnabled))
	utils.PrintInfo("These endpoints are reachable only by workloads in this cluster.")
	return nil
}

func handleCredentials(_ context.Context, in storageInput) error {
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	credentials, err := api.GetValkeyCredentials(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Valkey credentials: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(credentials) {
		return nil
	}
	utils.PrintHeader("Valkey Credentials")
	utils.PrintStatusLine("Username", credentials.Username)
	utils.PrintStatusLine("Password", credentials.Password)
	utils.PrintStatusLine("Private host", credentials.Host)
	utils.PrintStatusLine("Port", credentials.Port)
	utils.PrintStatusLine("Private URI", credentials.URI)
	if credentials.ReadOnlyURI != "" {
		utils.PrintStatusLine("Private read-only URI", credentials.ReadOnlyURI)
	}
	utils.PrintStatusLine("TLS", boolStatus(credentials.TLS))
	utils.PrintWarning("Credentials are sensitive. Valkey endpoints are reachable only by workloads in this cluster.")
	return nil
}

func handleRedeploy(_ context.Context, in storageInput) error {
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if err := api.RedeployValkey(storageID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to reconcile Valkey service: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Valkey reconciliation completed")
	return nil
}

func handleRestart(_ context.Context, in storageInput) error {
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if err := api.RestartValkey(storageID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to restart Valkey service: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Valkey rolling restart started")
	return nil
}

func handleDestroy(_ context.Context, in destroyInput) error {
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Destroy Valkey service %s and delete its data?", storageID), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteValkey(storageID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to destroy Valkey service: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Valkey service destroy started")
	return nil
}

func validateCreate(in createInput) error {
	if in.Name == "" {
		return utils.NewError("service name cannot be empty", nil)
	}
	if len(in.Name) > 40 {
		return utils.NewError("service name must be 40 characters or fewer", nil)
	}
	if !valkeyNamePattern.MatchString(in.Name) {
		return utils.NewError("service name must contain only lowercase letters, numbers, and hyphens, and must start and end with a letter or number", nil)
	}
	if in.Topology != "standalone" && in.Topology != "replicated" {
		return utils.NewError("topology must be standalone or replicated", nil)
	}
	if in.Instances < 0 {
		return utils.NewError("instances cannot be negative", nil)
	}
	if in.Topology == "standalone" && in.Instances > 1 {
		return utils.NewError("standalone topology supports exactly one instance", nil)
	}
	if in.Topology == "replicated" && in.Instances > 0 && in.Instances < 2 {
		return utils.NewError("replicated topology requires at least two instances", nil)
	}
	if in.AppendFsync != "always" && in.AppendFsync != "everysec" && in.AppendFsync != "no" {
		return utils.NewError("append-fsync must be always, everysec, or no", nil)
	}
	if _, ok := maxmemoryPolicies[in.MaxmemoryPolicy]; !ok {
		return utils.NewError("unsupported maxmemory-policy", nil)
	}
	if in.MaxmemoryPercent < 50 || in.MaxmemoryPercent > 90 {
		return utils.NewError("maxmemory-percent must be between 50 and 90", nil)
	}
	return nil
}

func resolveStorageID(arg string) (string, error) {
	if arg == "" {
		return "", utils.NewError("storage ID is required", nil)
	}
	if _, err := api.ParseUUID(arg); err == nil {
		return arg, nil
	}
	namespace := satuskyctx.GetCurrentNamespace()
	if namespace == "" {
		return "", utils.NewError("not authenticated — run '1ctl auth login' first", nil)
	}
	instances, err := api.ListValkey(namespace)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to list Valkey services: %s", err.Error()), nil)
	}
	for _, instance := range instances {
		if instance.ClusterName != nil && strings.EqualFold(*instance.ClusterName, arg) {
			return instance.StorageID.String(), nil
		}
	}
	return "", utils.NewError(fmt.Sprintf("Valkey service %q not found — pass the storage ID from '1ctl valkey list'", arg), nil)
}

func printInstance(instance *api.StorageConfig) {
	if instance == nil {
		return
	}
	utils.PrintHeader("Valkey Service")
	utils.PrintStatusLine("Storage ID", instance.StorageID.String())
	utils.PrintStatusLine("Name", stringValue(instance.ClusterName))
	utils.PrintStatusLine("Namespace", instance.Namespace)
	utils.PrintStatusLine("Version", instance.Version)
	utils.PrintStatusLine("Topology", topology(*instance))
	utils.PrintStatusLine("Instances", fmt.Sprintf("%d", effectiveInstances(*instance)))
	utils.PrintStatusLine("Persistence", boolPointerStatus(instance.PersistenceEnabled))
	utils.PrintStatusLine("Storage", fmt.Sprintf("%s (%s)", instance.StorageSize, instance.StorageClass))
	utils.PrintStatusLine("Resources", fmt.Sprintf("%s-%s CPU, %s-%s memory", instance.CPURequest, instance.CPULimit, instance.MemoryRequest, instance.MemoryLimit))
	if instance.Valkey != nil {
		utils.PrintStatusLine("Append only", boolStatus(instance.Valkey.AppendOnly))
		utils.PrintStatusLine("Append fsync", instance.Valkey.AppendFsync)
		utils.PrintStatusLine("Maxmemory", fmt.Sprintf("%d%%, %s", instance.Valkey.MaxmemoryPercent, instance.Valkey.MaxmemoryPolicy))
		utils.PrintStatusLine("Metrics", boolStatus(instance.Valkey.MetricsEnabled))
		utils.PrintStatusLine("Chart", instance.Valkey.ChartVersion)
		utils.PrintStatusLine("Image", instance.Valkey.ImageVersion)
	}
}

func topology(instance api.StorageConfig) string {
	if instance.Valkey == nil {
		return ""
	}
	return instance.Valkey.Topology
}

func effectiveInstances(instance api.StorageConfig) int {
	if instance.Instances != nil && *instance.Instances > 0 {
		return *instance.Instances
	}
	return instance.Replicas
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolStatus(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

func boolPointerStatus(value *bool) string {
	return boolStatus(value != nil && *value)
}
