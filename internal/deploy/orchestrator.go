package deploy

import (
	"1ctl/internal/api"
	"1ctl/internal/context"
	"1ctl/internal/docker"
	"1ctl/internal/utils"
	"fmt"
	"os"
	"path/filepath"
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

	userID := context.GetUserID()
	if userID == "" {
		return nil, utils.NewError("Failed to get user ID", nil)
	}

	// First try to use owner's machines if no hostnames provided
	if len(opts.Hostnames) == 0 {
		// Try owner's machines first
		machines, err := api.GetMachinesByOwnerID(api.ToUUID(userID))
		if err != nil {
			// Don't return error here, we'll fall back to monetized machines
			// log.Info("Failed to get owner's machines: %v", err)
			utils.PrintWarning("Failed to get owner's machines: %v", err)
		} else {
			// Check if owner has any machines and deduplicate hostnames
			hostnameSet := make(map[string]bool)
			var hostnames []string
			for _, machine := range machines {
				// Only add hostname if we haven't seen it before (using machine ID instead of machine name)
				if !hostnameSet[machine.MachineID] {
					hostnameSet[machine.MachineID] = true
					hostnames = append(hostnames, machine.MachineID)
				}
			}

			if len(hostnames) > 0 {
				opts.Hostnames = hostnames
				utils.PrintInfo("Using owner's machines: %v", hostnames)
			}
		}

		// If still no hostnames (no owner machines or error), let backend handle monetized machine selection
		if len(opts.Hostnames) == 0 {
			utils.PrintWarning("No owner machines available, will use monetized machines")
			// The backend will:
			// 1. Find cheapest machine with sufficient resources
			// 2. Check user's credit balance
			// 3. Create usage records if using monetized machines
		}
	}

	// Step 1: Build and push image
	progress.step = 1
	progress.message = "Building and pushing Docker image"
	progress.print()

	projectName, err := docker.GetProjectName()
	if err != nil {
		return nil, utils.NewError("Failed to determine project name", err)
	}

	image, err := buildAndUploadImage(opts.DockerfilePath, projectName)
	if err != nil {
		return nil, utils.NewError("Failed to build and push image", err)
	}
	progress.complete()

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
	progress.complete()

	// Step 3: Configure services
	progress.step = 3
	progress.message = "Configuring services"
	progress.resource = projectName
	progress.done = false
	progress.print()

	serviceID, err := upsertService(deploymentID, opts, projectName, opts.Organization)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to create service: %s", err.Error()), nil)
	}
	progress.complete()

	// Step 4: Handle environment and volumes
	progress.step = 4
	progress.message = "Setting up environment and storage"
	progress.resource = projectName
	progress.done = false
	progress.print()

	if err := handleEnvironmentAndVolumes(opts, deploymentID, projectName, opts.Organization); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to setup environment and volumes: %s", err.Error()), nil)
	}
	progress.complete()

	// Step 5: Handle ingress and dependencies
	progress.step = 5
	progress.message = "Configuring ingress and dependencies"
	progress.resource = projectName
	progress.done = false
	progress.print()

	domainName, err := handleIngressAndDependencies(opts, deploymentID, serviceID, userID, opts.Organization, projectName, opts.Hostnames)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to configure ingress and dependencies: %s", err.Error()), nil)
	}
	progress.complete()

	return &api.CreateDeploymentResponse{
		DeploymentID: api.ToUUID(deploymentID),
		AppLabel:     projectName,
		Domain:       domainName,
	}, nil
}

func buildAndUploadImage(dockerfilePath, projectName string) (string, error) {
	// Build Docker image locally
	if err := docker.Build(docker.BuildOptions{
		DockerfilePath: dockerfilePath,
		Tag:            projectName,
	}); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to build Docker image: %s", err.Error()), nil)
	}

	// Save image to temporary file
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s.tar", strings.ReplaceAll(projectName, "/", "_")))
	defer func() { _ = os.Remove(tmpFile) }() //nolint:errcheck // Clean up temp file

	if err := docker.SaveImage(projectName, tmpFile); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to save Docker image: %s", err.Error()), nil)
	}

	// Upload image to backend with retry logic
	var version string
	var uploadErr error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		utils.PrintInfo("Uploading Docker image (attempt %d/%d)...", attempt, maxRetries)

		version, uploadErr = api.UploadDockerImage(tmpFile, projectName)
		if uploadErr == nil {
			utils.PrintSuccess("Docker image uploaded successfully")
			break
		}

		if attempt < maxRetries {
			// Exponential backoff: wait 2^attempt seconds
			waitTime := 1 << attempt // 2, 4, 8 seconds
			utils.PrintWarning("Upload failed (attempt %d/%d): %s. Retrying in %d seconds...",
				attempt, maxRetries, uploadErr.Error(), waitTime)
			time.Sleep(time.Duration(waitTime) * time.Second)
		} else {
			utils.PrintError("Upload failed after %d attempts: %s", maxRetries, uploadErr.Error())
		}
	}

	if uploadErr != nil {
		return "", utils.NewError(fmt.Sprintf("failed to deploy Docker image after %d attempts: %s", maxRetries, uploadErr.Error()), nil)
	}

	// generate full image tag
	fullImage := fmt.Sprintf("%s/%s:%s", docker.RegistryURL, projectName, version)

	return fullImage, nil
}

func mainDeploy(opts DeploymentOptions, image, name, userID, organization string, hostnames []string) (string, error) {
	port, err := api.SafeInt32(opts.Port)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("invalid port: %s", err.Error()), nil)
	}

	// number of replicas will be based on the number of hostnames
	replicas, err := api.SafeInt32(len(hostnames))
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("invalid replicas count: %s", err.Error()), nil)
	}

	// if replicas is 0, set it to 1
	// this is to avoid creating a deployment with 0 replicas (mainly for user with no hostnames)
	if replicas == 0 {
		replicas = 1
	}

	deployment := api.Deployment{
		UserID:        api.ToUUID(userID),
		Type:          "production", // Default to production (cluster env)
		Environment:   "production", // Default to production (app env - can switch between development (preview) and production in future)
		Hostnames:     hostnames,
		CpuRequest:    opts.CPU,
		MemoryRequest: opts.Memory,
		MemoryLimit:   opts.Memory,
		Namespace:     organization,
		Port:          port,
		Image:         image,
		Region:        "SG",       // Default to Singapore region
		Zone:          "sg-sgp-1", // Default to Singapore zone
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

	var deploymentID string
	if err := api.UpsertDeployment(deployment, &deploymentID); err != nil {
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

func upsertIngress(deploymentID string, serviceID string, opts DeploymentOptions, organization, projectName string) (string, error) {
	// Check if there's an existing ingress for this deployment
	var domainName string
	existingIngress, err := api.GetIngressByDeploymentID(deploymentID)
	if err != nil {
		utils.PrintInfo("No existing ingress found for deployment %s, will create new one: %s", deploymentID, err.Error())

		// Generate domain name if not provided and no existing ingress
		if opts.Domain == "" {
			domainName, err = api.GenerateDomainName(projectName)
			if err != nil {
				return "", utils.NewError(fmt.Sprintf("failed to generate domain name: %s", err.Error()), nil)
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
		return "", utils.NewError(fmt.Sprintf("invalid port: %s", err.Error()), nil)
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
		return "", utils.NewError(fmt.Sprintf("failed to upsert ingress: %s", err.Error()), nil)
	}

	return ingressResp.DomainName, nil
}

func handleDependencies(deps []api.Dependency, userID, organization string, hostnames []string) error {
	for _, dep := range deps {
		opts := DeploymentOptions{
			CPU:          "100m",  // TODO: change this when --cpu is specified for each dependency
			Memory:       "128Mi", // TODO: change this when --memory is specified for each dependency
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

func handleEnvironmentAndVolumes(opts DeploymentOptions, deploymentID, projectName, organization string) error {
	errChan := make(chan error, 2)

	// Handle environment variables
	go func() {
		if opts.EnvEnabled && opts.Environment != nil {
			opts.Environment.DeploymentID = api.ToUUID(deploymentID)
			opts.Environment.AppLabel = projectName
			opts.Environment.Namespace = organization

			_, err := api.UpsertEnvironment(*opts.Environment)
			if err != nil {
				errChan <- utils.NewError(fmt.Sprintf("failed to create environment: %s", err.Error()), nil)
				return
			}
		}
		errChan <- nil
	}()

	// Handle volume
	go func() {
		if opts.VolumeEnabled && opts.Volume != nil {
			opts.Volume.DeploymentID = api.ToUUID(deploymentID)
			opts.Volume.VolumeName = fmt.Sprintf("%s-volume", projectName)
			opts.Volume.ClaimName = fmt.Sprintf("%s-claim", projectName)
			if err := api.CreateVolume(*opts.Volume); err != nil {
				errChan <- utils.NewError(fmt.Sprintf("failed to create volume: %s", err.Error()), nil)
				return
			}
		}
		errChan <- nil
	}()

	// Wait for both operations
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return err
		}
	}

	return nil
}

func handleIngressAndDependencies(opts DeploymentOptions, deploymentID, serviceID, userID, organization, projectName string, hostnames []string) (string, error) {
	errChan := make(chan error, 2)
	var domain string

	// Handle ingress
	go func() {
		domainName, err := upsertIngress(deploymentID, serviceID, opts, organization, projectName)
		if err != nil {
			errChan <- utils.NewError(fmt.Sprintf("failed to create ingress: %s", err.Error()), nil)
			return
		}
		errChan <- nil
		domain = domainName
	}()

	// Handle dependencies
	go func() {
		if len(opts.Dependencies) > 0 {
			if err := handleDependencies(opts.Dependencies, userID, organization, hostnames); err != nil {
				errChan <- utils.NewError(fmt.Sprintf("failed to handle dependencies: %s", err.Error()), nil)
				return
			}
		}
		errChan <- nil
	}()

	// Wait for both operations
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return "", err
		}
	}

	return domain, nil
}
