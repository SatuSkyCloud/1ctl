package deploy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"1ctl/internal/api"
	"1ctl/internal/config"
	satuskyctx "1ctl/internal/context"
	deploypkg "1ctl/internal/deploy"
	"1ctl/internal/utils"
	"1ctl/internal/validator"

	"github.com/google/uuid"
)

// --- Handlers -----------------------------------------------------------

func handleDeploy(ctx context.Context, in DeployInput) error {
	if err := satuskyctx.CheckTokenExpiry(); err != nil {
		return err
	}

	cfg, err := config.FindConfig(in.Config)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to load config: %s", err.Error()), nil)
	}

	merged := mergeConfig(in, cfg)

	if shouldShowDeployHelp(merged, cfg) {
		return utils.NewError("insufficient configuration — use flags or satusky.toml to specify CPU/memory or --image", nil)
	}

	if err := validateInputs(merged); err != nil {
		return utils.NewError(fmt.Sprintf("validation failed: %s", err.Error()), nil)
	}

	opts, err := prepareDeploymentOptions(merged, cfg)
	if err != nil {
		return utils.NewError(fmt.Sprintf("deployment preparation failed: %s", err.Error()), nil)
	}

	resp, err := deploypkg.Deploy(opts)
	if err != nil {
		if _, ok := err.(*utils.ResourceExhaustedCLIError); ok {
			return err
		}
		return utils.NewError(fmt.Sprintf("deployment failed: %s", err.Error()), nil)
	}

	ingressID := ""
	if resp != nil && resp.IngressID != uuid.Nil {
		ingressID = resp.IngressID.String()
	}
	publicURL := deploypkg.WaitForPublicURL(ingressID, resp.Domain)
	return deploypkg.ReportDeployResult(resp.AppLabel, resp.DeploymentID.String(), resp.Domain, publicURL, merged.HealthPath, merged.StrictSmoke)
}

type mergedInput struct {
	DeployInput
	Fast         bool
	StrictSmoke  bool
	AppName      string
	Organization string
	MachineTag   string
	UserSetFlags map[string]bool
}

func mergeConfig(in DeployInput, cfg *config.ProjectConfig) mergedInput {
	m := mergedInput{DeployInput: in}
	m.UserSetFlags = make(map[string]bool)

	trackSet := func(name string, isSet bool) {
		m.UserSetFlags[name] = isSet
	}

	trackSet("cpu", in.CPU != "")
	trackSet("cpu-request", in.CPURequest != "" && in.CPURequest != "250m")
	trackSet("cpu-limit", in.CPULimit != "" && in.CPULimit != "1")
	trackSet("memory", in.Memory != "" && in.Memory != "256Mi")
	trackSet("domain", in.Domain != "")
	trackSet("health-path", in.HealthPath != "")
	trackSet("rolling-max-surge", in.RollingMaxSurge != "" && in.RollingMaxSurge != "25%")
	trackSet("rolling-max-unavailable", in.RollingMaxUnavail != "" && in.RollingMaxUnavail != "25%")
	trackSet("multi-cluster", in.Multicluster)
	trackSet("fast", in.Fast)

	if cfg != nil {
		if m.CPURequest == "" || m.CPURequest == "250m" {
			if cfg.App.CPURequest != "" {
				m.CPURequest = cfg.App.CPURequest
			}
		}
		if m.CPULimit == "" || m.CPULimit == "1" {
			if cfg.App.CPULimit != "" {
				m.CPULimit = cfg.App.CPULimit
			} else if cfg.App.CPU != "" {
				m.CPULimit = cfg.App.CPU
			}
		}
		if m.Memory == "" || m.Memory == "256Mi" {
			applyIf(&m.Memory, cfg.App.Memory)
		}
		if m.Port == 0 || m.Port == 8080 {
			if cfg.App.Port != 0 {
				m.Port = cfg.App.Port
			}
		}
		applyIf(&m.Domain, cfg.App.Domain)
		applyIf(&m.Dockerfile, cfg.Build.Dockerfile)
		if m.Replicas == 0 {
			if cfg.App.Replicas > 0 {
				m.Replicas = cfg.App.Replicas
			}
		}
		applyIf(&m.Zone, cfg.App.Zone)
		applyIf(&m.Organization, cfg.App.Organization)
		applyIf(&m.HealthPath, cfg.Checks.HealthPath)
		applyIf(&m.Strategy, cfg.Deploy.Strategy)
		applyIf(&m.RollingMaxSurge, cfg.Deploy.RollingMaxSurge)
		applyIf(&m.RollingMaxUnavail, cfg.Deploy.RollingMaxUnavailable)
		applyIf(&m.VolumeSize, cfg.Volume.Size)
		applyIf(&m.VolumeMount, cfg.Volume.Mount)
		applyIf(&m.MachineTag, cfg.Deploy.MachineTag)
		if !m.Multicluster && cfg.Multicluster.Enabled {
			m.Multicluster = true
			if m.MulticlusterMode == "" || m.MulticlusterMode == "active-passive" {
				m.MulticlusterMode = cfg.Multicluster.Mode
			}
		}
		if !m.Fast && cfg.Build.FastBuild {
			m.Fast = true
		}
		if len(m.WaitFor) == 0 && len(cfg.Deploy.WaitFor) > 0 {
			m.WaitFor = cfg.Deploy.WaitFor
		}
	}

	if in.Name != "" {
		m.AppName = in.Name
	} else if cfg != nil && cfg.App.Name != "" {
		m.AppName = cfg.App.Name
	}

	if in.Organization != "" {
		m.Organization = in.Organization
	} else {
		m.Organization = satuskyctx.GetCurrentNamespace()
	}

	m.StrictSmoke = in.HealthPath != ""
	return m
}

func applyIf(dst *string, src string) {
	if src != "" && *dst == "" {
		*dst = src
	}
}

func shouldShowDeployHelp(m mergedInput, cfg *config.ProjectConfig) bool {
	if m.Image != "" {
		return false
	}
	if m.CPURequest != "" && m.Memory != "" {
		return false
	}
	return cfg == nil || (cfg.App.CPU == "" && cfg.App.CPURequest == "" && cfg.App.Memory == "")
}

func validateInputs(m mergedInput) error {
	if m.Image == "" {
		if err := validator.ValidateDockerfile(m.Dockerfile); err != nil {
			_, findErr := validator.FindDockerfile(".")
			if findErr != nil {
				return utils.NewError("no valid Dockerfile found: please ensure a Dockerfile exists in your project", err)
			}
		}
	}

	if m.CPU != "" {
		if err := validator.ValidateCPU(m.CPU); err != nil {
			return utils.NewError(fmt.Sprintf("invalid CPU value: %v", err), nil)
		}
	}
	if m.CPURequest != "" {
		if err := validator.ValidateCPU(m.CPURequest); err != nil {
			return utils.NewError(fmt.Sprintf("invalid CPU request value: %v", err), nil)
		}
	}
	if m.CPULimit != "" {
		if err := validator.ValidateCPU(m.CPULimit); err != nil {
			return utils.NewError(fmt.Sprintf("invalid CPU limit value: %v", err), nil)
		}
	}
	if m.Memory != "" {
		if err := validator.ValidateMemory(m.Memory); err != nil {
			return utils.NewError(fmt.Sprintf("invalid memory value: %v", err), nil)
		}
	}
	if m.Domain != "" {
		if err := validator.ValidateDomain(m.Domain); err != nil {
			return utils.NewError(fmt.Sprintf("invalid domain: %v", err), nil)
		}
	}
	if m.HealthPath != "" {
		if err := validator.ValidateURLPath(m.HealthPath); err != nil {
			return utils.NewError(fmt.Sprintf("invalid health path: %v", err), nil)
		}
	}

	if m.VolumeSize != "" || m.VolumeMount != "" {
		if m.VolumeSize == "" {
			return utils.NewError("volume-size is required when volume is enabled", nil)
		}
		if m.VolumeMount == "" {
			return utils.NewError("volume-mount is required when volume is enabled", nil)
		}
		if err := validator.ValidateMemory(m.VolumeSize); err != nil {
			return utils.NewError(fmt.Sprintf("invalid volume size: %v", err), nil)
		}
	}

	if m.HPA && m.VPA {
		if m.VPAMode == "Auto" {
			return utils.NewError("HPA and VPA with mode 'Auto' cannot be used together", nil)
		}
	}
	if m.HPAMinReplicas > m.HPAMaxReplicas {
		return utils.NewError("hpa-min-replicas cannot be greater than hpa-max-replicas", nil)
	}
	if m.PDBPercent < 0 || m.PDBPercent > 100 {
		return utils.NewError("pdb-percent must be between 1 and 100", nil)
	}

	if m.Multicluster {
		domain := strings.TrimSpace(strings.ToLower(m.Domain))
		domain = strings.TrimPrefix(domain, "*.")
		if domain != "" && domain != "satusky.com" && !strings.HasSuffix(domain, ".satusky.com") {
			return utils.NewError(fmt.Sprintf("--multi-cluster is not supported with custom domains yet: %q", m.Domain), nil)
		}
	}

	for _, v := range m.WaitFor {
		if _, _, err := validator.ValidateWaitFor(v); err != nil {
			return err
		}
	}

	return nil
}

func prepareDeploymentOptions(m mergedInput, cfg *config.ProjectConfig) (deploypkg.DeploymentOptions, error) {
	dockerfilePath := m.Dockerfile
	if m.Image == "" && dockerfilePath != "" {
		if err := validator.ValidateDockerfile(dockerfilePath); err != nil {
			found, findErr := validator.FindDockerfile(".")
			if findErr != nil {
				return deploypkg.DeploymentOptions{}, utils.NewError("no valid Dockerfile found", err)
			}
			dockerfilePath = found
		}
	}
	if m.Image != "" {
		dockerfilePath = ""
	}

	opts := deploypkg.DeploymentOptions{
		CPU:            m.CPU,
		CPURequest:     m.CPURequest,
		CPULimit:       m.CPULimit,
		Memory:         m.Memory,
		Domain:         m.Domain,
		SmokePath:      m.HealthPath,
		StrictSmoke:    m.StrictSmoke,
		Port:           m.Port,
		DockerfilePath: dockerfilePath,
		PrebuiltImage:  m.Image,
		FastBuild:      m.Fast,
	}

	opts.Name = m.AppName
	opts.Organization = m.Organization

	if len(m.Env) > 0 {
		opts.EnvEnabled = true
		opts.Environment = &api.Environment{
			KeyValues: parseEnvVars(m.Env),
		}
	}

	if len(m.Machine) > 0 {
		hostnameSet := make(map[string]bool)
		for _, machineName := range m.Machine {
			machine, err := api.GetMachineByName(machineName)
			if err != nil {
				return deploypkg.DeploymentOptions{}, utils.NewError(fmt.Sprintf("failed to get machine by name: %s", err.Error()), nil)
			}
			if !machine.Monetized && machine.OwnerID.String() != satuskyctx.GetUserID() {
				return deploypkg.DeploymentOptions{}, utils.NewError(fmt.Sprintf("machine %s is not owned by you", machineName), nil)
			}
			if !hostnameSet[machine.MachineID] {
				hostnameSet[machine.MachineID] = true
				opts.Hostnames = append(opts.Hostnames, machine.MachineID)
			}
		}
	}

	if m.VolumeSize != "" || m.VolumeMount != "" {
		opts.VolumeEnabled = true
		storageClass := m.VolumeStorageClass
		if storageClass == "" {
			storageClass = "ceph-block"
		}
		opts.Volume = &api.Volume{
			StorageSize:  m.VolumeSize,
			MountPath:    m.VolumeMount,
			StorageClass: storageClass,
		}
	}

	if m.Zone != "" {
		opts.Zone = m.Zone
	}

	if m.MachineTag != "" && len(m.Machine) == 0 {
		hostnames, err := resolveMachineTag(m.MachineTag)
		if err != nil {
			return deploypkg.DeploymentOptions{}, err
		}
		opts.Hostnames = hostnames
		utils.PrintInfo("Resolved --machine-tag %q to %d owned machine(s)", m.MachineTag, len(hostnames))
	}

	if m.Multicluster {
		opts.MulticlusterEnabled = true
		opts.MulticlusterMode = m.MulticlusterMode
		opts.BackupEnabled = m.BackupEnabled
		opts.BackupSchedule = m.BackupSchedule
		opts.BackupRetention = m.BackupRetention
		opts.BackupPriorityCluster = m.BackupPriority
	}

	if m.Replicas > 0 {
		opts.Replicas = m.Replicas
	}

	if m.PDB {
		pdbCfg := &deploypkg.PDBConfig{
			Enabled: true,
			Type:    deploypkg.PDBConfigType(m.PDBType),
		}
		if m.PDBMinAvailable > 0 {
			val, err := api.SafeInt32(m.PDBMinAvailable)
			if err != nil {
				return deploypkg.DeploymentOptions{}, utils.NewError("invalid pdb-min-available value", err)
			}
			pdbCfg.MinAvailable = &val
		}
		if m.PDBPercent > 0 {
			val, err := api.SafeInt32(m.PDBPercent)
			if err != nil {
				return deploypkg.DeploymentOptions{}, utils.NewError("invalid pdb-percent value", err)
			}
			pdbCfg.Percent = &val
		}
		opts.PDBConfig = pdbCfg
	}

	if m.HPA {
		cpuTarget, err := api.SafeInt32(m.HPACPUCoreTarget)
		if err != nil {
			return deploypkg.DeploymentOptions{}, utils.NewError("invalid hpa-cpu-target value", err)
		}
		minReplicas, err := api.SafeInt32(m.HPAMinReplicas)
		if err != nil {
			return deploypkg.DeploymentOptions{}, utils.NewError("invalid hpa-min-replicas value", err)
		}
		maxReplicas, err := api.SafeInt32(m.HPAMaxReplicas)
		if err != nil {
			return deploypkg.DeploymentOptions{}, utils.NewError("invalid hpa-max-replicas value", err)
		}
		hpaCfg := &api.HPAConfig{
			Enabled:     true,
			MinReplicas: minReplicas,
			MaxReplicas: maxReplicas,
			CPUTarget:   &cpuTarget,
		}
		if m.HPAMemoryTarget > 0 {
			memTarget, err := api.SafeInt32(m.HPAMemoryTarget)
			if err != nil {
				return deploypkg.DeploymentOptions{}, utils.NewError("invalid hpa-memory-target value", err)
			}
			hpaCfg.MemoryTarget = &memTarget
		}
		opts.HPAConfig = hpaCfg
	}

	if m.VPA {
		opts.VPAConfig = &api.VPAConfig{
			Enabled:    true,
			UpdateMode: m.VPAMode,
			MinCPU:     m.VPAMinCPU,
			MaxCPU:     m.VPAMaxCPU,
			MinMemory:  m.VPAMinMemory,
			MaxMemory:  m.VPAMaxMemory,
		}
	}

	for _, v := range m.WaitFor {
		host, port, err := validator.ValidateWaitFor(v)
		if err != nil {
			return deploypkg.DeploymentOptions{}, err
		}
		opts.WaitFor = append(opts.WaitFor, api.WaitFor{Host: host, Port: port})
	}

	opts.Strategy = m.Strategy
	opts.RollingMaxSurge = m.RollingMaxSurge
	opts.RollingMaxUnavailable = m.RollingMaxUnavail
	opts.RollingFlagsExplicit = m.UserSetFlags["rolling-max-surge"] || m.UserSetFlags["rolling-max-unavailable"]
	switch opts.Strategy {
	case "rolling", "recreate":
	default:
		return deploypkg.DeploymentOptions{}, utils.NewError(fmt.Sprintf("invalid --strategy %q: must be 'rolling' or 'recreate'", opts.Strategy), nil)
	}

	if cfg != nil {
		if !m.HPA && cfg.HPA.Enabled {
			applyConfigHPA(&opts, cfg.HPA)
		}
		if !m.VPA && cfg.VPA.Enabled {
			applyConfigVPA(&opts, cfg.VPA)
		}
		if !m.PDB && cfg.PDB.Enabled {
			applyConfigPDB(&opts, cfg.PDB)
		}
		if !m.Multicluster && cfg.Multicluster.Enabled {
			applyConfigMulticluster(&opts, cfg.Multicluster)
		}
	}

	return opts, nil
}

func applyConfigHPA(opts *deploypkg.DeploymentOptions, hpa config.HPAConfig) {
	cfg := &api.HPAConfig{
		Enabled:     true,
		MinReplicas: defaultInt32(hpa.MinReplicas, 1),
		MaxReplicas: defaultInt32(hpa.MaxReplicas, 10),
	}
	cpu := defaultInt32(hpa.CPUTarget, 80)
	cfg.CPUTarget = &cpu
	if hpa.MemoryTarget > 0 {
		mem := hpa.MemoryTarget
		cfg.MemoryTarget = &mem
	}
	opts.HPAConfig = cfg
}

func applyConfigVPA(opts *deploypkg.DeploymentOptions, vpa config.VPAConfig) {
	mode := vpa.Mode
	if mode == "" {
		mode = "Off"
	}
	opts.VPAConfig = &api.VPAConfig{
		Enabled:    true,
		UpdateMode: mode,
		MinCPU:     vpa.MinCPU,
		MaxCPU:     vpa.MaxCPU,
		MinMemory:  vpa.MinMemory,
		MaxMemory:  vpa.MaxMemory,
	}
}

func applyConfigPDB(opts *deploypkg.DeploymentOptions, pdb config.PDBConfig) {
	typ := pdb.Type
	if typ == "" {
		typ = "auto"
	}
	cfg := &deploypkg.PDBConfig{Enabled: true, Type: deploypkg.PDBConfigType(typ)}
	if pdb.MinAvailable > 0 {
		v := pdb.MinAvailable
		cfg.MinAvailable = &v
	}
	if pdb.Percent > 0 {
		v := pdb.Percent
		cfg.Percent = &v
	}
	opts.PDBConfig = cfg
}

func applyConfigMulticluster(opts *deploypkg.DeploymentOptions, mc config.MulticlusterConfig) {
	opts.MulticlusterEnabled = true
	opts.MulticlusterMode = mc.Mode
	if opts.MulticlusterMode == "" {
		opts.MulticlusterMode = "active-passive"
	}
	opts.BackupEnabled = mc.BackupEnabled
	opts.BackupSchedule = mc.BackupSchedule
	opts.BackupRetention = mc.BackupRetention
	opts.BackupPriorityCluster = mc.BackupPriorityCluster
	if opts.BackupPriorityCluster == 0 {
		opts.BackupPriorityCluster = 1
	}
}

func defaultInt32(v, fallback int32) int32 {
	if v == 0 {
		return fallback
	}
	return v
}

func resolveMachineTag(tag string) ([]string, error) {
	userID := satuskyctx.GetUserID()
	if userID == "" {
		return nil, utils.NewError("not authenticated — run '1ctl auth login' first", nil)
	}
	userUUID, err := api.ParseUUID(userID)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("invalid user ID in context: %s", err.Error()), nil)
	}
	machines, err := api.GetMachinesByOwnerID(userUUID)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to list owned machines: %s", err.Error()), nil)
	}
	if len(machines) == 0 {
		return nil, utils.NewError("no owned machines found — register a machine before using --machine-tag", nil)
	}
	var hostnames []string
	for _, m := range machines {
		if m.Status != "online" {
			continue
		}
		labels, err := api.GetMachineLabels(m.MachineID)
		if err != nil {
			utils.PrintWarning("Could not read labels for machine %s: %s", m.MachineID, err.Error())
			continue
		}
		if api.MachineHasTag(labels, tag) {
			hostnames = append(hostnames, m.MachineID)
		}
	}
	if len(hostnames) == 0 {
		return nil, utils.NewError(fmt.Sprintf("no machines tagged %q are online", tag), nil)
	}
	return hostnames, nil
}



// --- List / Get ---------------------------------------------------------

func handleListDeployments(ctx context.Context) error {
	namespace, err := satuskyctx.GetCurrentNamespaceOrError()
	if err != nil {
		return err
	}
	deployments, err := api.ListDeploymentsByNamespace(namespace)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list deployments: %s", err.Error()), nil)
	}

	if utils.PrintListOrJSON(deployments, "No deployments found") {
		return nil
	}

	headers := []string{"NAME", "DEPLOYMENT ID", "HOSTNAMES", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(deployments))
	for _, d := range deployments {
		name := d.AppLabel
		if name == "" {
			name = "-"
		}
		rows = append(rows, []string{
			name,
			d.DeploymentID.String(),
			strings.Join(d.Hostnames, ", "),
			d.Status,
			api.FormatTimeAgo(d.CreatedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleGetDeployment(ctx context.Context, in GetDeploymentInput) error {
	deploymentID, err := deploypkg.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}
	deployment, err := api.GetDeployment(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get deployment: %s", err.Error()), nil)
	}

	if ingress, iErr := api.GetIngressByDeploymentID(deploymentID); iErr == nil && ingress != nil && ingress.DomainName != "" {
		deployment.Domain = "https://" + ingress.DomainName
	}

	if utils.TryPrintJSON(deployment) {
		return nil
	}

	utils.PrintHeader("Deployment Details")
	if deployment.MarketplaceAppName != "" {
		utils.PrintStatusLine("Application (from marketplace)", deployment.MarketplaceAppName)
	}
	utils.PrintStatusLine("Deployment ID", deployment.DeploymentID.String())
	utils.PrintStatusLine("Status", deployment.Status)
	if deployment.Domain != "" {
		utils.PrintStatusLine("URL", deployment.Domain)
	}
	utils.PrintStatusLine("Deployed to machines", strings.Join(deployment.Hostnames, ", "))
	utils.PrintStatusLine("Type", deployment.Type)
	utils.PrintStatusLine("Zone", deployment.Zone)
	version := "untagged"
	if parts := strings.SplitN(deployment.Image, ":", 2); len(parts) == 2 {
		version = parts[1]
	}
	utils.PrintStatusLine("Version", version)
	utils.PrintStatusLine("Port", fmt.Sprintf("%d", deployment.Port))
	utils.PrintStatusLine("CPU Request", deployment.CpuRequest)
	utils.PrintStatusLine("Memory Request", deployment.MemoryRequest)
	utils.PrintStatusLine("Memory Limit", deployment.MemoryLimit)
	utils.PrintStatusLine("Created", api.FormatTimeAgo(deployment.CreatedAt))
	utils.PrintStatusLine("Last Updated", api.FormatTimeAgo(deployment.UpdatedAt))
	return nil
}

// --- Status -------------------------------------------------------------

func handleDeploymentStatus(ctx context.Context, in StatusInput) error {
	deploymentID, err := deploypkg.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}

	if in.Watch {
		status, err := api.WaitForDeployment(deploymentID, 5*time.Minute)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to watch deployment: %s", err.Error()), nil)
		}
		utils.PrintStatusLine("Final status", status.Status)
		if status.Message != "" {
			utils.PrintStatusLine("Message", status.Message)
		}
		return nil
	}

	status, err := api.GetDeploymentStatus(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get deployment status: %s", err.Error()), nil)
	}

	deployment, err := api.GetDeployment(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get deployment details: %s", err.Error()), nil)
	}

	var ingress *api.Ingress
	var domainStatus *api.DomainStatusResponse
	if ing, ingErr := api.GetIngressByDeploymentID(deploymentID); ingErr == nil {
		ingress = ing
		if ing.DomainName != "" {
			if ds, dsErr := api.GetDomainStatus(ing.IngressID.String(), ing.DomainName, false); dsErr == nil {
				domainStatus = ds
			}
		}
	}

	details := struct {
		Deployment   *api.Deployment           `json:"deployment"`
		Status       *api.DeploymentStatus     `json:"status"`
		Ingress      *api.Ingress              `json:"ingress,omitempty"`
		DomainStatus *api.DomainStatusResponse `json:"domain_status,omitempty"`
	}{
		Deployment:   deployment,
		Status:       status,
		Ingress:      ingress,
		DomainStatus: domainStatus,
	}
	if utils.TryPrintJSON(details) {
		return nil
	}

	utils.PrintHeader("Deployment Status")
	utils.PrintStatusLine("App", deployment.AppLabel)
	utils.PrintStatusLine("Deployment ID", deployment.DeploymentID.String())
	utils.PrintStatusLine("Namespace", deployment.Namespace)
	if ingress != nil && ingress.DomainName != "" {
		utils.PrintStatusLine("URL", "https://"+ingress.DomainName)
	}
	utils.PrintStatusLine("Workload", status.Status)
	if status.Message != "" {
		utils.PrintStatusLine("Message", status.Message)
	}
	utils.PrintStatusLine("Progress", fmt.Sprintf("%d%%", status.Progress))
	utils.PrintStatusLine("Image", deployment.Image)
	utils.PrintStatusLine("Replicas", fmt.Sprintf("%d desired", deployment.Replicas))
	utils.PrintStatusLine("Strategy", deploymentStrategyText(deployment.StrategyConfig))
	utils.PrintStatusLine("Environment", enabledText(deployment.EnvEnabled))
	utils.PrintStatusLine("Secrets", enabledText(deployment.SecretEnabled))
	utils.PrintStatusLine("Volume", enabledText(deployment.VolumeEnabled))
	if domainStatus != nil {
		utils.PrintStatusLine("Route", domainRouteText(domainStatus.Route))
		utils.PrintStatusLine("DNS", domainDNSText(domainStatus.DNS))
		utils.PrintStatusLine("TLS", domainTLSText(domainStatus.TLS))
	}
	utils.PrintStatusLine("Created", api.FormatTimeAgo(deployment.CreatedAt))
	utils.PrintStatusLine("Last Updated", api.FormatTimeAgo(deployment.UpdatedAt))
	return nil
}

// --- Destroy ------------------------------------------------------------

func handleDestroyDeployment(ctx context.Context, in DestroyInput) error {
	deploymentID, err := deploypkg.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}

	preview, pErr := previewDeletion(deploymentID)
	if pErr == nil {
		fmt.Println(strings.Join(preview, "\n"))
		fmt.Println()
	} else {
		utils.PrintWarning("Could not preview resources: %s", pErr.Error())
	}

	if !utils.Confirm("This will permanently destroy the deployment and all its associated resources.", in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}

	utils.PrintInfo("Destroying deployment %s...", deploymentID)
	statuses, volErr := api.GetDeploymentVolumeLifecycleStatuses(deploymentID)
	if volErr != nil {
		utils.PrintWarning("Could not list volumes for destruction: %s", volErr.Error())
	} else {
		for _, v := range statuses {
			utils.PrintInfo("Destroying volume %s (PVC: %s)...", v.Volume.VolumeName, v.PVC.Name)
			if _, delErr := api.DeleteVolumePVC(v.Volume.VolumeID.String()); delErr != nil {
				utils.PrintWarning("Failed to destroy volume %s: %s", v.Volume.VolumeName, delErr.Error())
			}
		}
	}

	result, err := api.DeleteDeployment(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to delete deployment: %s", err.Error()), nil)
	}

	if utils.TryPrintJSON(result) {
		return nil
	}
	printDeletionResult(deploymentID, result)
	return nil
}

func previewDeletion(deploymentID string) ([]string, error) {
	var lines []string
	dep, err := api.GetDeployment(deploymentID)
	if err != nil {
		return nil, err
	}
	lines = append(lines, fmt.Sprintf("App: %s (ID: %s)", dep.AppLabel, deploymentID))
	lines = append(lines, "")
	lines = append(lines, "Resources that will be deleted:")

	ing, err := api.GetIngressByDeploymentID(deploymentID)
	if err == nil && ing != nil {
		domainDisplay := ing.DomainName
		if domainDisplay == "" {
			domainDisplay = "(no domain attached)"
		}
		lines = append(lines, fmt.Sprintf("  • Ingress  — %s", domainDisplay))
	}

	volumes, err := api.GetDeploymentVolumeLifecycleStatuses(deploymentID)
	if err == nil {
		for _, v := range volumes {
			policy := v.DestroyPolicy
			if policy == "" {
				policy = "default"
			}
			lines = append(lines, fmt.Sprintf("  • Volume   — %s (%s, destroy: %s)", v.Volume.ClaimName, v.Volume.StorageSize, policy))
		}
	}

	services, err := api.ListServices()
	if err == nil {
		for _, s := range services {
			if s.DeploymentID.String() == deploymentID {
				lines = append(lines, fmt.Sprintf("  • Service  — %s (port: %d)", s.ServiceName, s.Port))
			}
		}
	}

	if len(lines) == 2 {
		lines = append(lines, "  (no additional resources found)")
	}
	return lines, nil
}

func printDeletionResult(deploymentID string, result *api.DeletionResult) {
	utils.PrintSuccess("Deployment %s delete completed", deploymentID)
	if result == nil {
		return
	}
	utils.PrintHeader("Deleted Resources")
	if result.AppLabel != "" {
		utils.PrintStatusLine("App", result.AppLabel)
	}
	if result.Namespace != "" {
		utils.PrintStatusLine("Namespace", result.Namespace)
	}
	if len(result.DeletedDeployments) > 0 {
		utils.PrintStatusLine("Deployments", strings.Join(result.DeletedDeployments, ", "))
	} else {
		utils.PrintStatusLine("Deployments", "none reported")
	}
	if result.IsCNPGDeployment {
		utils.PrintStatusLine("CNPG", "database deployment cleanup applied")
	}
	if len(result.Volumes) == 0 {
		utils.PrintStatusLine("PVCs", "none reported")
		return
	}
	headers := []string{"PVC", "VOLUME", "STATUS", "POLICY", "MESSAGE"}
	rows := make([][]string, 0, len(result.Volumes))
	for _, volume := range result.Volumes {
		rows = append(rows, []string{
			volume.ClaimName,
			volume.VolumeName,
			volume.Status,
			volume.DestroyPolicy,
			volume.Message,
		})
	}
	utils.PrintTable(headers, rows)
}

// --- Restart / Releases / Rollback / Open / Scale -----------------------

func handleRestartDeployment(ctx context.Context, in DeployRefInput) error {
	deploymentID, err := deploypkg.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}
	utils.PrintInfo("Initiating rolling restart for deployment %s...", deploymentID)
	if err := api.RestartDeployment(deploymentID); err != nil {
		return utils.NewError(fmt.Sprintf("failed to restart: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Rolling restart initiated.")
	utils.PrintInfo("Use '1ctl deploy status --deployment-id %s' to monitor progress.", deploymentID)
	return nil
}

func handleListReleases(ctx context.Context, in DeployRefInput) error {
	deploymentID, err := deploypkg.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}
	versions, err := api.ListDeploymentVersions(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to list releases: %s", err.Error()), nil)
	}
	if utils.PrintListOrJSON(versions, "No releases found") {
		return nil
	}
	headers := []string{"VERSION", "IMAGE", "STATUS", "DEPLOYED"}
	rows := make([][]string, 0, len(versions))
	for _, v := range versions {
		rows = append(rows, []string{
			fmt.Sprintf("%d", v.VersionNumber),
			v.Image,
			v.Status,
			api.FormatTimeAgo(v.DeployedAt),
		})
	}
	utils.PrintTable(headers, rows)
	return nil
}

func handleRollback(ctx context.Context, in RollbackInput) error {
	deploymentID, err := deploypkg.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}
	version := in.Version
	if version == 0 {
		versions, err := api.ListDeploymentVersions(deploymentID)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to fetch releases: %s", err.Error()), nil)
		}
		if len(versions) < 2 {
			return utils.NewError("no previous release to roll back to", nil)
		}
		version = versions[1].VersionNumber
	}

	if !utils.Confirm(fmt.Sprintf("Roll back deployment %s to version %d?", deploymentID, version), in.Yes) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := api.RollbackDeployment(deploymentID, version); err != nil {
		return utils.NewError(fmt.Sprintf("rollback failed: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Rollback to version %d initiated", version)
	utils.PrintInfo("Use '1ctl deploy status --deployment-id %s' to monitor progress.", deploymentID)
	return nil
}

func handleOpenDeployment(ctx context.Context, in DeployRefInput) error {
	deploymentID, err := deploypkg.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}
	ing, err := api.GetIngressByDeploymentID(deploymentID)
	if err != nil || ing == nil || ing.DomainName == "" {
		return utils.NewError(fmt.Sprintf("no domain attached to deployment %s — use '1ctl domains add' first", deploymentID), nil)
	}
	url := "https://" + ing.DomainName
	utils.PrintInfo("Opening %s", url)
	if err := deploypkg.OpenBrowser(url); err != nil {
		utils.PrintWarning("Could not open browser: %s", err.Error())
		utils.PrintInfo("URL: %s", url)
	}
	return nil
}

func handleScaleDeployment(ctx context.Context, in ScaleInput) error {
	deploymentID, err := deploypkg.ResolveDeploymentID(in.DeploymentID, in.App, in.Config)
	if err != nil {
		return err
	}
	replicas, err := api.SafeInt32(in.Replicas)
	if err != nil {
		return utils.NewError("invalid --replicas value", err)
	}
	if replicas < 1 {
		return utils.NewError("--replicas must be >= 1", nil)
	}

	current, err := api.GetDeployment(deploymentID)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to fetch deployment: %s", err.Error()), nil)
	}
	if current.HPAConfig != nil && current.HPAConfig.Enabled {
		return utils.NewError(fmt.Sprintf("deployment %s is managed by HPA — adjust --hpa-min-replicas / --hpa-max-replicas instead", deploymentID), nil)
	}
	if current.VPAConfig != nil && current.VPAConfig.Enabled {
		return utils.NewError(fmt.Sprintf("deployment %s is managed by VPA", deploymentID), nil)
	}
	if current.Replicas == replicas {
		utils.PrintInfo("Deployment %s already at %d replicas — no change.", deploymentID, replicas)
		return nil
	}
	current.Replicas = replicas

	var resp string
	if err := api.UpsertDeployment(*current, &resp); err != nil {
		return utils.NewError(fmt.Sprintf("failed to scale deployment: %s", err.Error()), nil)
	}
	utils.PrintSuccess("Scaled deployment %s to %d replicas", deploymentID, replicas)
	return nil
}

// --- Display helpers ----------------------------------------------------

func deploymentStrategyText(strategy *api.DeploymentStrategyConfig) string {
	if strategy == nil {
		return "default"
	}
	if strategy.Rolling == nil {
		return string(strategy.Type)
	}
	return fmt.Sprintf("%s (maxSurge=%s, maxUnavailable=%s)",
		strategy.Type,
		strategy.Rolling.MaxSurge,
		strategy.Rolling.MaxUnavailable,
	)
}

func enabledText(enabled bool) string {
	if enabled {
		return "attached"
	}
	return "not attached"
}

func parseEnvVars(envVars []string) []api.KeyValuePair {
	var keyValues []api.KeyValuePair
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			keyValues = append(keyValues, api.KeyValuePair{
				Key:   parts[0],
				Value: parts[1],
			})
		}
	}
	return keyValues
}

func domainRouteText(status api.DomainRouteStatus) string {
	if !status.Attached {
		if status.Message != "" {
			return "not attached: " + status.Message
		}
		return "not attached"
	}
	if status.ResourceKind == "" && status.ResourceName == "" {
		return "attached"
	}
	return fmt.Sprintf("attached to %s/%s", status.ResourceKind, status.ResourceName)
}

func domainDNSText(status api.DNSStatusResponse) string {
	parts := []string{string(status.Status)}
	if len(status.ResolvedIPs) > 0 {
		parts = append(parts, "resolved "+strings.Join(status.ResolvedIPs, ", "))
	}
	if status.ExpectedIP != "" {
		parts = append(parts, "expected "+status.ExpectedIP)
	}
	if status.Message != "" {
		parts = append(parts, status.Message)
	}
	return strings.Join(parts, " - ")
}

func domainTLSText(status api.TLSStatusResponse) string {
	if status.Status == "" {
		return "unknown"
	}
	if status.ExpiresAt != nil {
		return fmt.Sprintf("%s, expires %s", status.Status, status.ExpiresAt.Format("2006-01-02"))
	}
	if status.Message != "" {
		return fmt.Sprintf("%s - %s", status.Message, status.Status)
	}
	return string(status.Status)
}
