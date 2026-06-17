package deploy

import (
	"1ctl/internal/api"
	"1ctl/internal/cleanup"
	"1ctl/internal/context"
	"1ctl/internal/docker"
	"1ctl/internal/utils"
	"1ctl/internal/validator"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

type deploymentProgress struct {
	step     int
	total    int
	message  string
	resource string
	done     bool
}

func (dp *deploymentProgress) print() {
	utils.PrintLoadingStep(dp.step, dp.total, dp.message, dp.resource, dp.done)
}

func (dp *deploymentProgress) complete() {
	dp.done = true
	dp.print()
}

// Deploy handles the sequential deployment process
func Deploy(opts DeploymentOptions) (*api.CreateDeploymentResponse, error) {
	progress := &deploymentProgress{total: 5}
	cmgr := cleanup.NewCleanupManager()

	userID := context.GetUserID()
	if userID == "" {
		return nil, utils.NewError("Failed to get user ID", nil)
	}

	// BYOA targeting is now explicit: the user must pass --machine or
	// --machine-tag to deploy to owned hardware. Default behaviour deploys
	// to managed cloud — issue #24 retires the implicit owner-machine
	// auto-selection that bypassed quota enforcement and confused new users.
	if len(opts.Hostnames) == 0 {
		utils.PrintInfo("Deploying to managed cloud — backend will select the cheapest suitable machine.")
	}

	var projectName string
	if opts.Name != "" {
		projectName = opts.Name
	} else {
		var err2 error
		projectName, err2 = docker.GetProjectName()
		if err2 != nil {
			return nil, utils.NewError("Failed to determine project name", err2)
		}
		utils.PrintInfo("App name: %s (auto-detected — use --name to override)", projectName)
	}

	// K8s Services use DNS-1035: must start with a letter, only [a-z0-9-], end with alphanumeric.
	if err := validateAppName(projectName); err != nil {
		return nil, err
	}

	// Step 1: Build and push image (skipped when a pre-built image is provided)
	var (
		image string
		err   error
	)
	if opts.PrebuiltImage != "" {
		image = opts.PrebuiltImage
		utils.PrintInfo("Using pre-built image: %s", image)
	} else {
		progress.step = 1
		progress.message = "Building image (cloud)"
		if opts.FastBuild {
			progress.message = "Building image (fast cloud)"
		}
		progress.print()

		var imageArch string
		image, imageArch, err = submitRemoteBuild(opts.DockerfilePath, projectName, opts.FastBuild)
		if err != nil {
			return nil, utils.NewError("Failed to build image", err)
		}
		opts.TargetArch = normalizeTargetArch(imageArch)
		progress.complete()
	}

	// Step 2: Create deployment
	progress.step = 2
	progress.message = "Creating/updating deployment"
	progress.resource = projectName
	progress.done = false
	progress.print()

	deploymentID, err := mainDeploy(opts, image, projectName, userID, opts.Organization, opts.Hostnames)
	if err != nil {
		return nil, err
	}
	cmgr.AddResource(cleanup.ResourceDeployment, deploymentID, projectName)
	progress.complete()

	// Step 3: Configure services
	progress.step = 3
	progress.message = "Configuring services"
	progress.resource = projectName
	progress.done = false
	progress.print()

	serviceID, err := upsertService(deploymentID, opts, projectName, opts.Organization)
	if err != nil {
		deployCleanup(cmgr)
		return nil, utils.NewError(fmt.Sprintf("failed to create service: %s", err.Error()), nil)
	}
	cmgr.AddResource(cleanup.ResourceService, serviceID, projectName)
	progress.complete()

	// Step 4: Handle environment and volumes
	progress.step = 4
	progress.message = "Setting up environment and storage"
	progress.resource = projectName
	progress.done = false
	progress.print()

	envID, volumeName, err := handleEnvironmentAndVolumes(opts, deploymentID, projectName, opts.Organization)
	if err != nil {
		deployCleanup(cmgr)
		return nil, utils.NewError(fmt.Sprintf("failed to setup environment and volumes: %s", err.Error()), nil)
	}
	if envID != "" {
		cmgr.AddResource(cleanup.ResourceEnv, envID, projectName)
	}
	if volumeName != "" {
		cmgr.AddResource(cleanup.ResourceVolume, volumeName, projectName)
	}
	progress.complete()

	// Step 5: Handle ingress and dependencies
	progress.step = 5
	progress.message = "Configuring ingress and dependencies"
	progress.resource = projectName
	progress.done = false
	progress.print()

	domainName, ingressID, err := handleIngressAndDependencies(opts, deploymentID, serviceID, userID, opts.Organization, projectName, opts.Hostnames)
	if err != nil {
		deployCleanup(cmgr)
		return nil, utils.NewError(fmt.Sprintf("failed to configure ingress and dependencies: %s", err.Error()), nil)
	}
	if ingressID != "" {
		cmgr.AddResource(cleanup.ResourceIngress, ingressID, projectName)
	}
	progress.complete()

	return &api.CreateDeploymentResponse{
		DeploymentID: api.ToUUID(deploymentID),
		IngressID:    api.ToUUID(ingressID),
		AppLabel:     projectName,
		Domain:       domainName,
	}, nil
}

// deployCleanup runs best-effort cleanup on partial deployment failure.
func deployCleanup(cmgr *cleanup.CleanupManager) {
	utils.PrintWarning("Deployment failed, attempting cleanup of created resources...")
	if errs := cmgr.Cleanup(); len(errs) > 0 {
		utils.PrintWarning("Cleanup encountered errors:\n%s", cleanup.FormatCleanupErrors(errs))
	} else {
		utils.PrintSuccess("Successfully cleaned up partial deployment resources")
	}
}

// submitRemoteBuild packages the local build context, uploads it to the backend,
// and waits for the cloud build to complete. No local Docker daemon is required.
// Returns the image reference, image architecture, and any error.
func submitRemoteBuild(dockerfilePath, projectName string, fastBuild bool) (imageRef, imageArch string, err error) {
	// Validate that the Dockerfile exists and is well-formed before shipping anything.
	if err = validator.ValidateDockerfile(dockerfilePath); err != nil {
		return "", "", utils.NewError(fmt.Sprintf("invalid Dockerfile: %s", err.Error()), nil)
	}

	// Package the build context into a gzipped tar, respecting .dockerignore.
	utils.PrintInfo("Packaging build context...")
	contextPath, err := docker.PackageContext(".")
	if err != nil {
		return "", "", utils.NewError(fmt.Sprintf("failed to package build context: %s", err.Error()), nil)
	}
	defer func() { _ = os.Remove(contextPath) }() //nolint:errcheck

	builder := api.BuildBackendDefault
	if fastBuild {
		builder = api.BuildBackendDepot
	}

	// Submit the context to the backend; it returns a build ID immediately.
	if fastBuild {
		utils.PrintInfo("Submitting fast build to cloud...")
	} else {
		utils.PrintInfo("Submitting build to cloud...")
	}
	buildID, err := api.SubmitBuild(contextPath, projectName, dockerfilePath, builder, nil)
	if err != nil {
		return "", "", utils.NewError(fmt.Sprintf("failed to submit build: %s", err.Error()), nil)
	}
	utils.PrintInfo("Build queued (ID: %s)", buildID)

	// Poll until the cloud build finishes, streaming log output as it arrives.
	// TODO: Should we be polling? is there a better way other than polling?
	result, err := api.WaitForBuildResult(buildID, os.Stdout)
	if err != nil {
		return "", "", err
	}

	utils.PrintSuccess("Cloud build complete: %s", result.ImageRef)
	if result.ImageArch != "" {
		utils.PrintInfo("Image architecture: %s", result.ImageArch)
	}
	return result.ImageRef, result.ImageArch, nil
}

// normalizeTargetArch converts a build result into a single Kubernetes arch label.
// Multi-arch platform lists are intentionally collapsed to empty so the backend
// does not apply an invalid nodeSelector like "linux/amd64,linux/arm64".
func normalizeTargetArch(imageArch string) string {
	imageArch = strings.TrimSpace(imageArch)
	if imageArch == "" || strings.Contains(imageArch, ",") {
		return ""
	}

	imageArch = strings.TrimPrefix(imageArch, "linux/")

	switch imageArch {
	case "amd64", "arm64":
		return imageArch
	default:
		return ""
	}
}

func mainDeploy(opts DeploymentOptions, image, name, userID, organization string, hostnames []string) (string, error) {
	port, err := api.SafeInt32(opts.Port)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("invalid port: %s", err.Error()), nil)
	}
	cpuRequest := opts.CPURequest
	if cpuRequest == "" {
		cpuRequest = "250m"
	}
	cpuLimit := opts.CPULimit
	if cpuLimit == "" {
		cpuLimit = opts.CPU
	}
	if cpuLimit == "" {
		cpuLimit = "1"
	}

	// Replica count: use manual override if set, otherwise derive from hostnames
	var replicas int32
	if opts.Replicas > 0 {
		replicas, err = api.SafeInt32(opts.Replicas)
		if err != nil {
			return "", utils.NewError(fmt.Sprintf("invalid replicas count: %s", err.Error()), nil)
		}
	} else {
		replicas, err = api.SafeInt32(len(hostnames))
		if err != nil {
			return "", utils.NewError(fmt.Sprintf("invalid replicas count: %s", err.Error()), nil)
		}
	}

	// if replicas is 0, set it to 1
	// this is to avoid creating a deployment with 0 replicas (mainly for user with no hostnames)
	if replicas == 0 {
		replicas = 1
	}

	// TODO: Should we be hardcoding any of the values here? Make it strict and require to pass in params.
	deployment := api.Deployment{
		UserID:        api.ToUUID(userID),
		Type:          "production", // Default to production (cluster env)
		Environment:   "production", // Default to production (app env - can switch between development (preview) and production in future)
		Hostnames:     hostnames,
		CpuRequest:    cpuRequest,
		CPULimit:      cpuLimit,
		MemoryRequest: opts.Memory,
		MemoryLimit:   opts.Memory,
		Namespace:     organization,
		Port:          port,
		Image:         image,
		Zone:          opts.Zone, // Target zone for cluster routing
		SSD:           "true",
		GPU:           "false",
		AppLabel:      name,
		Replicas:      replicas,
		EnvEnabled:    opts.EnvEnabled,
		VolumeEnabled: opts.VolumeEnabled,
	}

	// Add multicluster configuration if enabled
	if opts.MulticlusterEnabled {
		scheduleMap := map[string]string{
			"hourly": "0 * * * *",
			"daily":  "0 0 * * *",
			"weekly": "0 18 * * 6", // 2 AM MYT Sunday
		}

		isActivePassive := opts.MulticlusterMode == "active-passive"

		// For active-passive: backup is always enabled
		// For active-active: backup is optional (user can toggle via --backup-enabled)
		backupEnabled := isActivePassive || opts.BackupEnabled

		// Priority cluster defaults to 1 (primary) if not set
		priorityCluster := opts.BackupPriorityCluster
		if priorityCluster <= 0 {
			priorityCluster = 1
		}

		deployment.MulticlusterConfig = &api.MulticlusterConfig{
			Enabled:               true,
			Mode:                  opts.MulticlusterMode,
			BackupEnabled:         backupEnabled,
			BackupSchedule:        scheduleMap[opts.BackupSchedule],
			BackupRetention:       opts.BackupRetention,
			BackupPriorityCluster: priorityCluster,
			FailoverEnabled:       isActivePassive,
			RestoreOnFailover:     isActivePassive,
		}
	}

	// Add PDB configuration
	if opts.PDBConfig != nil && opts.PDBConfig.Enabled {
		deployment.PDBConfig = &api.PDBConfig{
			Enabled:      opts.PDBConfig.Enabled,
			Type:         string(opts.PDBConfig.Type),
			MinAvailable: opts.PDBConfig.MinAvailable,
			Percent:      opts.PDBConfig.Percent,
		}
	} else if replicas > 1 {
		// Auto-enable PDB when replicas > 1
		deployment.PDBConfig = &api.PDBConfig{
			Enabled: true,
			Type:    "auto",
		}
	}

	// Add HPA configuration
	if opts.HPAConfig != nil {
		deployment.HPAConfig = opts.HPAConfig
	}

	// Add VPA configuration
	if opts.VPAConfig != nil {
		deployment.VPAConfig = opts.VPAConfig
	}

	// Add wait-for dependencies (platform injects init containers)
	if len(opts.WaitFor) > 0 {
		deployment.WaitFor = opts.WaitFor
	}

	// Add deployment strategy configuration
	deployment.StrategyConfig = buildStrategyConfig(opts)

	// Pass image architecture so the backend sets the kubernetes.io/arch nodeSelector.
	deployment.TargetArch = opts.TargetArch

	var deploymentID string
	if err := api.UpsertDeployment(deployment, &deploymentID); err != nil {
		// Check if this is a resource exhausted error and handle it specially
		if resourceErr, ok := err.(*utils.ResourceExhaustedCLIError); ok {
			utils.PrintResourceExhaustedError(resourceErr.ResourceError)
			return "", resourceErr
		}
		return "", utils.NewError(fmt.Sprintf("failed to upsert deployment: %s", err.Error()), nil)
	}

	return deploymentID, nil
}

func upsertService(deploymentID string, opts DeploymentOptions, projectName, organization string) (string, error) {
	port, err := api.SafeInt32(opts.Port)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("invalid port: %s", err.Error()), nil)
	}

	service := api.Service{
		DeploymentID: api.ToUUID(deploymentID),
		Namespace:    organization,
		ServiceName:  projectName,
		Port:         port,
	}

	var serviceID string
	if err := api.UpsertService(service, &serviceID); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to upsert service: %s", err.Error()), nil)
	}

	return serviceID, nil
}

func upsertIngress(deploymentID string, serviceID string, opts DeploymentOptions, organization, projectName string) (domainName, ingressID string, err error) {
	// Check if there's an existing ingress for this deployment
	existingIngress, err := api.GetIngressByDeploymentID(deploymentID)
	if err != nil {
		utils.PrintInfo("No existing ingress found for deployment %s, will create new one: %s", deploymentID, err.Error())

		// Generate domain name if not provided and no existing ingress
		if opts.Domain == "" {
			domainName, err = api.GenerateDomainName(projectName)
			if err != nil {
				return "", "", utils.NewError(fmt.Sprintf("failed to generate domain name: %s", err.Error()), nil)
			}
			utils.PrintInfo("Generated new domain: %s", domainName)
		} else {
			domainName = opts.Domain
		}
	} else {
		// Use existing domain name if no explicit domain provided
		if opts.Domain == "" {
			domainName = existingIngress.DomainName
		} else {
			domainName = opts.Domain
		}
	}

	port, err := api.SafeInt32(opts.Port)
	if err != nil {
		return "", "", utils.NewError(fmt.Sprintf("invalid port: %s", err.Error()), nil)
	}

	ingress := api.Ingress{
		DeploymentID: api.ToUUID(deploymentID),
		ServiceID:    api.ToUUID(serviceID),
		Namespace:    organization,
		AppLabel:     projectName,
		DomainName:   domainName,
		DnsConfig:    api.DnsConfigDefault,
		Port:         port,
	}

	ingressResp, err := api.UpsertIngress(ingress)
	if err != nil {
		return "", "", utils.NewError(fmt.Sprintf("failed to upsert ingress: %s", err.Error()), nil)
	}

	return ingressResp.DomainName, ingressResp.IngressID.String(), nil
}

func handleDependencies(deps []api.Dependency, userID, organization string, hostnames []string) error {
	for _, dep := range deps {
		opts := DeploymentOptions{
			CPURequest:   "125m",  // TODO: change this when CPU is specified for each dependency
			CPULimit:     "1000m", // TODO: change this when CPU is specified for each dependency
			Memory:       "128Mi", // TODO: change this when memory is specified for each dependency
			Organization: organization,
			Port:         int(dep.Service.Port), // Convert int32 to int
		}

		// Create deployment for dependency
		deploymentID, err := mainDeploy(opts, dep.Image, dep.Name, userID, organization, hostnames)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to create dependency deployment: %s", err.Error()), nil)
		}

		// Create service for dependency
		if dep.Service != nil {
			dep.Service.DeploymentID = api.ToUUID(deploymentID)
			if err := api.UpsertService(*dep.Service, nil); err != nil {
				return utils.NewError(fmt.Sprintf("failed to upsert dependency service: %s", err.Error()), nil)
			}
		}

		// Create volume for dependency if specified
		if dep.Volume != nil {
			dep.Volume.DeploymentID = api.ToUUID(deploymentID)
			if err := api.CreateVolume(*dep.Volume); err != nil {
				return utils.NewError(fmt.Sprintf("failed to create dependency volume: %s", err.Error()), nil)
			}
		}
	}

	return nil
}

// handleEnvironmentAndVolumes returns the envID and volumeName of any resources
// it created so the caller can register them with the cleanup manager.
// Either may be "" when the corresponding feature wasn't enabled.
func handleEnvironmentAndVolumes(opts DeploymentOptions, deploymentID, projectName, organization string) (envID, volumeName string, err error) {
	type envResult struct {
		id  string
		err error
	}
	type volResult struct {
		name string
		err  error
	}
	envChan := make(chan envResult, 1)
	volChan := make(chan volResult, 1)

	go func() {
		if opts.EnvEnabled && opts.Environment != nil {
			opts.Environment.DeploymentID = api.ToUUID(deploymentID)
			opts.Environment.AppLabel = projectName
			opts.Environment.Namespace = organization

			created, e := api.UpsertEnvironment(*opts.Environment)
			if e != nil {
				envChan <- envResult{err: utils.NewError(fmt.Sprintf("failed to create environment: %s", e.Error()), nil)}
				return
			}
			envChan <- envResult{id: created.EnvironmentID.String()}
			return
		}
		envChan <- envResult{}
	}()

	go func() {
		if opts.VolumeEnabled && opts.Volume != nil {
			opts.Volume.DeploymentID = api.ToUUID(deploymentID)
			opts.Volume.VolumeName = fmt.Sprintf("%s-volume", projectName)
			opts.Volume.ClaimName = fmt.Sprintf("%s-claim", projectName)
			if e := api.CreateVolume(*opts.Volume); e != nil {
				volChan <- volResult{err: utils.NewError(fmt.Sprintf("failed to create volume: %s", e.Error()), nil)}
				return
			}
			// Poll for PVC readiness. The storage provisioner (Ceph RBD) typically
			// binds PVCs in under 30 seconds. We poll for up to 60s with a fast
			// initial interval then exponential backoff so the deploy doesn't
			// block unnecessarily while the PVC is provisioning.
			//
			// Check both PVC.Exists AND PVC.Phase == "Bound" — a PVC can exist
			// (created by the backend) but still be Pending while the provisioner
			// creates the backing volume.
			bound := false
			for i := 0; i < 30; i++ {
				statuses, sErr := api.GetDeploymentVolumeLifecycleStatuses(deploymentID)
				if sErr == nil {
					for _, s := range statuses {
						if s.PVC.Exists && s.PVC.Phase == "Bound" {
							bound = true
							break
						}
					}
					if bound {
						break
					}
				}
				// 2s for first 10 attempts, then 5s — provisions usually
				// complete quickly and we don't want to hold up deploy.
				delay := 2 * time.Second
				if i >= 10 {
					delay = 5 * time.Second
				}
				time.Sleep(delay)
			}
			if !bound {
				utils.PrintWarning("PVC %s is still provisioning after 60s — storage will be available shortly", opts.Volume.ClaimName)
			}
			volChan <- volResult{name: opts.Volume.VolumeName}
			return
		}
		volChan <- volResult{}
	}()

	envRes := <-envChan
	volRes := <-volChan
	// Both errors are surfaced — previously only errs[0] was returned, hiding
	// the second when both goroutines failed simultaneously.
	if joined := errors.Join(envRes.err, volRes.err); joined != nil {
		return envRes.id, volRes.name, joined
	}
	return envRes.id, volRes.name, nil
}

// buildStrategyConfig converts DeploymentOptions strategy fields into the API struct.
//
// Optimisation: when the user didn't touch any strategy flag, we omit the
// strategy config from the request to reduce noise. When the user explicitly
// passed --rolling-max-surge or --rolling-max-unavailable — even with the
// default values — the config is sent through so audit logs / version history
// capture the user's intent.
func buildStrategyConfig(opts DeploymentOptions) *api.DeploymentStrategyConfig {
	strategy := opts.Strategy
	if strategy == "" || strategy == "rolling" {
		if opts.RollingMaxSurge == "25%" && opts.RollingMaxUnavailable == "25%" && !opts.RollingFlagsExplicit {
			// User-untouched defaults — omit config.
			return nil
		}
		return &api.DeploymentStrategyConfig{
			Type: api.StrategyRolling,
			Rolling: &api.RollingUpdateConfig{
				MaxSurge:       opts.RollingMaxSurge,
				MaxUnavailable: opts.RollingMaxUnavailable,
			},
		}
	}

	if api.DeploymentStrategyType(strategy) == api.StrategyRecreate {
		return &api.DeploymentStrategyConfig{Type: api.StrategyRecreate}
	}
	return nil
}

// handleIngressAndDependencies returns the resolved domain name and the
// ingressID of any ingress it created so the caller can register the resource
// with the cleanup manager.
func handleIngressAndDependencies(opts DeploymentOptions, deploymentID, serviceID, userID, organization, projectName string, hostnames []string) (domainName, ingressID string, err error) {
	type ingressResult struct {
		domain string
		id     string
		err    error
	}
	ingressChan := make(chan ingressResult, 1)
	depErrChan := make(chan error, 1)

	go func() {
		domain, id, e := upsertIngress(deploymentID, serviceID, opts, organization, projectName)
		if e != nil {
			ingressChan <- ingressResult{err: utils.NewError(fmt.Sprintf("failed to create ingress: %s", e.Error()), nil)}
			return
		}
		ingressChan <- ingressResult{domain: domain, id: id}
	}()

	go func() {
		if len(opts.Dependencies) > 0 {
			if e := handleDependencies(opts.Dependencies, userID, organization, hostnames); e != nil {
				depErrChan <- utils.NewError(fmt.Sprintf("failed to handle dependencies: %s", e.Error()), nil)
				return
			}
		}
		depErrChan <- nil
	}()

	ing := <-ingressChan
	depErr := <-depErrChan
	if joined := errors.Join(ing.err, depErr); joined != nil {
		return "", ing.id, joined
	}
	return ing.domain, ing.id, nil
}

// dns1035 matches valid K8s Service names (DNS-1035): starts with a letter,
// only lowercase alphanumeric and hyphens, ends with alphanumeric, max 63 chars.
var dns1035 = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

// validateAppName checks the name against DNS-1035 before the deploy pipeline
// starts, so users get an actionable error before any K8s resources are created.
func validateAppName(name string) error {
	if len(name) > 63 {
		return utils.NewError(fmt.Sprintf(
			"app name %q is too long (%d chars, max 63). Use --name <short-name> to set a shorter name, or update [app] name in satusky.toml.",
			name, len(name)), nil)
	}
	if !dns1035.MatchString(name) {
		hint := ""
		if len(name) > 0 && (name[0] >= '0' && name[0] <= '9') {
			hint = fmt.Sprintf(" (starts with a digit — try --name %s)", "app-"+name)
		}
		return utils.NewError(fmt.Sprintf(
			"app name %q is not a valid K8s service name%s.\n"+
				"  Names must start with a letter, contain only [a-z0-9-], and end with [a-z0-9].\n"+
				"  Source: --name flag, [app] name in satusky.toml, or git remote auto-detect.",
			name, hint), nil)
	}
	return nil
}
