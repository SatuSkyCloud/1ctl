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
			// Check if owner has any machines
			var hostnames []string
			for _, machine := range machines {
				hostnames = append(hostnames, machine.MachineName)
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
	progress.message = "Creating deployment"
	progress.resource = projectName
	progress.done = false
	progress.print()

	deploymentID, err := createMainDeployment(opts, image, projectName, userID, opts.Organization, opts.Hostnames)
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

	serviceID, err := createService(deploymentID, opts, projectName, opts.Organization)
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
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s.tar", strings.Replace(projectName, "/", "_", -1)))
	defer os.Remove(tmpFile) // Clean up temp file

	if err := docker.SaveImage(projectName, tmpFile); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to save Docker image: %s", err.Error()), nil)
	}

	// Upload image to backend
	version, err := api.UploadDockerImage(tmpFile, projectName)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to deploy Docker image: %s", err.Error()), nil)
	}

	// generate full image tag
	fullImage := fmt.Sprintf("%s/%s:%s", docker.RegistryURL, projectName, version)

	return fullImage, nil
}

func createMainDeployment(opts DeploymentOptions, image, name, userID, organization string, hostnames []string) (string, error) {
	port, err := api.SafeInt32(opts.Port)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("invalid port: %s", err.Error()), nil)
	}

	// number of replicas will be based on the number of hostnames
	replicas, err := api.SafeInt32(len(hostnames))
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("invalid replicas count: %s", err.Error()), nil)
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

	var deploymentID string
	if err := api.CreateDeployment(deployment, &deploymentID); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to create deployment: %s", err.Error()), nil)
	}

	return deploymentID, nil
}

func createService(deploymentID string, opts DeploymentOptions, projectName, organization string) (string, error) {
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
	if err := api.CreateService(service, &serviceID); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to create service: %s", err.Error()), nil)
	}

	return serviceID, nil
}

func createIngress(deploymentID string, serviceID string, opts DeploymentOptions, organization, projectName string) (string, error) {
	// Generate domain name if not provided
	domainName := opts.Domain
	if domainName == "" {
		var err error
		domainName, err = api.GenerateDomainName(projectName)
		if err != nil {
			return "", utils.NewError(fmt.Sprintf("failed to generate domain name: %s", err.Error()), nil)
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

	_, err = api.CreateIngress(ingress)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to create ingress: %s", err.Error()), nil)
	}

	return domainName, nil
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
		deploymentID, err := createMainDeployment(opts, dep.Image, dep.Name, userID, organization, hostnames)
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to create dependency deployment: %s", err.Error()), nil)
		}

		// Create service for dependency
		if dep.Service != nil {
			dep.Service.DeploymentID = api.ToUUID(deploymentID)
			if err := api.CreateService(*dep.Service, nil); err != nil {
				return utils.NewError(fmt.Sprintf("failed to create dependency service: %s", err.Error()), nil)
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
			_, err := api.CreateEnvironment(*opts.Environment)
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
		domainName, err := createIngress(deploymentID, serviceID, opts, organization, projectName)
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
