package valkey

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"1ctl/internal/api"
	satuskyctx "1ctl/internal/context"
	"1ctl/internal/utils"
)

var valkeyNamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
var valkeyUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$`)

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
		MachineID:        strings.TrimSpace(in.MachineID),
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

func handleUpdate(_ context.Context, in updateInput) error {
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if in.AppendOnly && in.NoAppendOnly {
		return utils.NewError("append-only and no-append-only cannot be used together", nil)
	}
	if in.MetricsEnabled && in.NoMetrics {
		return utils.NewError("metrics and no-metrics cannot be used together", nil)
	}
	in.AppendFsync = strings.ToLower(strings.TrimSpace(in.AppendFsync))
	in.MaxmemoryPolicy = strings.ToLower(strings.TrimSpace(in.MaxmemoryPolicy))
	if err := validateMutableSettings(in); err != nil {
		return err
	}
	if !hasUpdate(in) {
		return utils.NewError("provide at least one mutable setting to update", nil)
	}

	instance, err := api.GetValkey(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Valkey service: %s", err.Error()), nil)
	}
	if instance.Engine != api.StorageEngineValkey || instance.Valkey == nil {
		return utils.NewError(fmt.Sprintf("storage %s is not a Valkey service", storageID), nil)
	}
	if in.Instances > 0 {
		if instance.Valkey.Topology == "standalone" && in.Instances != 1 {
			return utils.NewError("standalone topology supports exactly one instance", nil)
		}
		if instance.Valkey.Topology == "replicated" && in.Instances < 2 {
			return utils.NewError("replicated topology requires at least two instances", nil)
		}
		if instance.Valkey.Topology == "replicated" && in.Instances > 10 {
			return utils.NewError("replicated topology supports at most ten instances", nil)
		}
		instance.Replicas = in.Instances
		instance.Instances = &in.Instances
	}
	if in.CPURequest != "" {
		instance.CPURequest = in.CPURequest
	}
	if in.CPULimit != "" {
		instance.CPULimit = in.CPULimit
	}
	if in.MemoryRequest != "" {
		instance.MemoryRequest = in.MemoryRequest
	}
	if in.MemoryLimit != "" {
		instance.MemoryLimit = in.MemoryLimit
	}
	if in.AppendOnly || in.NoAppendOnly {
		appendOnly := in.AppendOnly
		instance.Valkey.AppendOnly = &appendOnly
	}
	if in.AppendFsync != "" {
		instance.Valkey.AppendFsync = in.AppendFsync
	}
	if in.MaxmemoryPolicy != "" {
		instance.Valkey.MaxmemoryPolicy = in.MaxmemoryPolicy
	}
	if in.MaxmemoryPercent > 0 {
		instance.Valkey.MaxmemoryPercent = in.MaxmemoryPercent
	}
	if in.MetricsEnabled || in.NoMetrics {
		metricsEnabled := in.MetricsEnabled
		instance.Valkey.MetricsEnabled = &metricsEnabled
	}

	updated, err := api.UpdateValkey(storageID, instance)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to update Valkey service: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(updated) {
		return nil
	}
	utils.PrintSuccess("Valkey settings updated and reconciled")
	printInstance(updated)
	return nil
}

func handleUsersList(_ context.Context, in storageInput) error {
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	users, err := api.ListValkeyUsers(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list Valkey users: %s", err.Error()), nil)
	}
	if utils.PrintListOrJSON(users, "No custom Valkey users found") {
		return nil
	}
	rows := make([][]string, 0, len(users))
	for _, user := range users {
		rows = append(rows, []string{
			user.Username,
			user.AccessPreset,
			patternsValue(user.KeyPatterns),
			patternsValue(user.ChannelPatterns),
		})
	}
	utils.PrintTable([]string{"USERNAME", "PRESET", "KEY PATTERNS", "CHANNEL PATTERNS"}, rows)
	utils.PrintInfo("The default and replication users are protected system users.")
	return nil
}

func handleUsersCreate(_ context.Context, in userMutationInput) error {
	if err := validateUserMutation(in, true); err != nil {
		return err
	}
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	created, err := api.CreateValkeyUser(storageID, api.ValkeyCreateUserRequest{
		Username:        in.Username,
		AccessPreset:    strings.ToLower(strings.TrimSpace(in.AccessPreset)),
		KeyPatterns:     in.KeyPatterns,
		ChannelPatterns: in.ChannelPatterns,
	})
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to create Valkey user: %s", err.Error()), nil)
	}
	created.Notice = "Save this generated password now; it will not be shown again."
	if utils.TryPrintJSON(created) {
		return nil
	}
	utils.PrintSuccess("Valkey user created")
	printUser(created.User)
	printGeneratedPassword(created.User.Username, created.Password)
	printRestartPending(created.RestartPending)
	return nil
}

func handleUsersUpdate(_ context.Context, in userMutationInput) error {
	if err := validateUserMutation(in, false); err != nil {
		return err
	}
	if in.ClearKeyPatterns && len(in.KeyPatterns) > 0 {
		return utils.NewError("key-pattern and clear-key-patterns cannot be used together", nil)
	}
	if in.ClearChannelPatterns && len(in.ChannelPatterns) > 0 {
		return utils.NewError("channel-pattern and clear-channel-patterns cannot be used together", nil)
	}
	if in.AccessPreset == "" && in.KeyPatterns == nil && in.ChannelPatterns == nil &&
		!in.ClearKeyPatterns && !in.ClearChannelPatterns {
		return utils.NewError("provide a preset or pattern change to update", nil)
	}
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	request := api.ValkeyUpdateUserRequest{}
	if in.AccessPreset != "" {
		preset := strings.ToLower(strings.TrimSpace(in.AccessPreset))
		request.AccessPreset = &preset
	}
	if in.KeyPatterns != nil || in.ClearKeyPatterns {
		patterns := in.KeyPatterns
		if in.ClearKeyPatterns {
			patterns = []string{}
		}
		request.KeyPatterns = &patterns
	}
	if in.ChannelPatterns != nil || in.ClearChannelPatterns {
		patterns := in.ChannelPatterns
		if in.ClearChannelPatterns {
			patterns = []string{}
		}
		request.ChannelPatterns = &patterns
	}
	user, err := api.UpdateValkeyUser(storageID, in.Username, request)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to update Valkey user: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(user) {
		return nil
	}
	utils.PrintSuccess("Valkey user updated")
	printUser(*user)
	return nil
}

func handleUsersDelete(_ context.Context, in confirmedUserInput) error {
	if err := validateMutableUsername(in.Username); err != nil {
		return err
	}
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Delete Valkey user %s?", in.Username), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := api.DeleteValkeyUser(storageID, in.Username); err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete Valkey user: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Valkey user deleted")
	return nil
}

func handleUsersRotatePassword(_ context.Context, in confirmedUserInput) error {
	if err := validateMutableUsername(in.Username); err != nil {
		return err
	}
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if !utils.Confirm(fmt.Sprintf("Rotate the password for Valkey user %s? Existing clients will lose access.", in.Username), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	credential, err := api.RotateValkeyUserPassword(storageID, in.Username)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to rotate Valkey user password: %s", err.Error()), nil)
	}
	credential.Notice = "Save this generated password now; it will not be shown again."
	if utils.TryPrintJSON(credential) {
		return nil
	}
	utils.PrintSuccess("Valkey user password rotated")
	printGeneratedPassword(credential.Username, credential.Password)
	printRestartPending(credential.RestartPending)
	return nil
}

func handleRotateCredentials(_ context.Context, in confirmedStorageInput) error {
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	if !utils.Confirm("Rotate the default Valkey credential? Existing clients will lose access.", in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}
	credential, err := api.RotateValkeyCredentials(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to rotate Valkey credentials: %s", err.Error()), nil)
	}
	credential.Notice = "Save this generated password now; it will not be shown again."
	if utils.TryPrintJSON(credential) {
		return nil
	}
	utils.PrintSuccess("Default Valkey credential rotated")
	printGeneratedPassword(credential.Username, credential.Password)
	printRestartPending(credential.RestartPending)
	utils.PrintWarning("Update every client that uses the previous credential.")
	return nil
}

func handleMetrics(_ context.Context, in storageInput) error {
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	metrics, err := api.GetValkeyMetrics(storageID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Valkey metrics: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(metrics) {
		return nil
	}
	if len(metrics) == 0 {
		utils.PrintInfo("No Valkey metrics are available yet")
		return nil
	}
	keys := make([]string, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		value := metrics[key]
		if value == "" {
			value = "unavailable"
		}
		rows = append(rows, []string{key, value})
	}
	utils.PrintTable([]string{"METRIC", "VALUE"}, rows)
	return nil
}

func handleLogs(_ context.Context, in logsInput) error {
	if in.Tail < 1 || in.Tail > 2000 {
		return utils.NewError("tail must be between 1 and 2000", nil)
	}
	storageID, err := resolveStorageID(in.StorageID)
	if err != nil {
		return err
	}
	logs, err := api.GetValkeyLogs(storageID, in.Tail)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Valkey logs: %s", err.Error()), nil)
	}
	if utils.TryPrintJSON(logs) {
		return nil
	}
	if logs.Pod != "" {
		utils.PrintInfo("Pod: %s", logs.Pod)
	}
	if len(logs.Lines) == 0 {
		utils.PrintInfo("No Valkey log lines are available yet")
		return nil
	}
	for _, line := range logs.Lines {
		fmt.Println(line)
	}
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
	if in.Topology == "replicated" && in.Instances > 10 {
		return utils.NewError("replicated topology supports at most ten instances", nil)
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

func validateMutableSettings(in updateInput) error {
	if in.Instances < 0 {
		return utils.NewError("instances cannot be negative", nil)
	}
	if in.AppendFsync != "" && in.AppendFsync != "always" && in.AppendFsync != "everysec" && in.AppendFsync != "no" {
		return utils.NewError("append-fsync must be always, everysec, or no", nil)
	}
	if in.MaxmemoryPolicy != "" {
		if _, ok := maxmemoryPolicies[in.MaxmemoryPolicy]; !ok {
			return utils.NewError("unsupported maxmemory-policy", nil)
		}
	}
	if in.MaxmemoryPercent != 0 && (in.MaxmemoryPercent < 50 || in.MaxmemoryPercent > 90) {
		return utils.NewError("maxmemory-percent must be between 50 and 90", nil)
	}
	return nil
}

func hasUpdate(in updateInput) bool {
	return in.Instances > 0 || in.CPURequest != "" || in.CPULimit != "" ||
		in.MemoryRequest != "" || in.MemoryLimit != "" || in.AppendOnly ||
		in.NoAppendOnly || in.AppendFsync != "" || in.MaxmemoryPolicy != "" ||
		in.MaxmemoryPercent > 0 || in.MetricsEnabled || in.NoMetrics
}

func validateUserMutation(in userMutationInput, requirePreset bool) error {
	if err := validateMutableUsername(in.Username); err != nil {
		return err
	}
	preset := strings.ToLower(strings.TrimSpace(in.AccessPreset))
	if requirePreset || preset != "" {
		if preset != "admin" && preset != "read_write" && preset != "read_only" {
			return utils.NewError("preset must be admin, read_write, or read_only", nil)
		}
	}
	if len(in.KeyPatterns) > 16 {
		return utils.NewError("at most 16 key patterns are supported", nil)
	}
	if len(in.ChannelPatterns) > 16 {
		return utils.NewError("at most 16 channel patterns are supported", nil)
	}
	for _, pattern := range in.KeyPatterns {
		if err := validateACLPattern(pattern, "~"); err != nil {
			return utils.NewError(fmt.Sprintf("invalid key pattern %q: %s", pattern, err.Error()), nil)
		}
	}
	for _, pattern := range in.ChannelPatterns {
		if err := validateACLPattern(pattern, "&"); err != nil {
			return utils.NewError(fmt.Sprintf("invalid channel pattern %q: %s", pattern, err.Error()), nil)
		}
	}
	return nil
}

func validateMutableUsername(username string) error {
	if !valkeyUsernamePattern.MatchString(username) {
		return utils.NewError("username must be 1-32 characters using only letters, numbers, dots, underscores, and hyphens", nil)
	}
	if username == "default" || username == "replication" {
		return utils.NewError(fmt.Sprintf("%s is a protected system user", username), nil)
	}
	return nil
}

func validateACLPattern(pattern, aclPrefix string) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}
	if len(pattern) > 256 {
		return fmt.Errorf("pattern must be 256 characters or fewer")
	}
	if strings.HasPrefix(pattern, aclPrefix) {
		return fmt.Errorf("omit the ACL %q prefix", aclPrefix)
	}
	if strings.ContainsAny(pattern, " \t\r\n\x00") {
		return fmt.Errorf("pattern cannot contain whitespace or NUL")
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
		utils.PrintStatusLine("Append only", boolPointerStatus(instance.Valkey.AppendOnly))
		utils.PrintStatusLine("Append fsync", instance.Valkey.AppendFsync)
		utils.PrintStatusLine("Maxmemory", fmt.Sprintf("%d%%, %s", instance.Valkey.MaxmemoryPercent, instance.Valkey.MaxmemoryPolicy))
		utils.PrintStatusLine("Metrics", boolPointerStatus(instance.Valkey.MetricsEnabled))
		utils.PrintStatusLine("Chart", instance.Valkey.ChartVersion)
		utils.PrintStatusLine("Image", instance.Valkey.ImageVersion)
	}
}

func printUser(user api.ValkeyUserConfig) {
	utils.PrintStatusLine("Username", user.Username)
	utils.PrintStatusLine("Preset", user.AccessPreset)
	utils.PrintStatusLine("Key patterns", patternsValue(user.KeyPatterns))
	utils.PrintStatusLine("Channel patterns", patternsValue(user.ChannelPatterns))
}

func printGeneratedPassword(username, password string) {
	utils.PrintStatusLine("Username", username)
	utils.PrintStatusLine("Password", password)
	utils.PrintWarning("Save this generated password now; it will not be shown again.")
}

func printRestartPending(pending bool) {
	if pending {
		utils.PrintWarning("The password was saved, but the rolling restart is still pending. Check service status before reconnecting.")
	}
}

func patternsValue(patterns []string) string {
	if len(patterns) == 0 {
		return "preset default"
	}
	return strings.Join(patterns, ",")
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
