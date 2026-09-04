package deploy

import (
	"1ctl/internal/api"
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

	"github.com/google/uuid"
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
func Deploy(opts DeploymentOptions, requestID string) (*api.CreateDeploymentResponse, error) {
	progress := &deploymentProgress{total: 5}

	if fallbackReason := atomicIntentFallbackReason(opts); fallbackReason != "" {
		return nil, utils.NewError(fmt.Sprintf("deployment cannot be submitted safely: %s is not supported by the atomic deployment API", fallbackReason), nil)
	}

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
		setSourceBuildTargetArch(&opts, imageArch)
		progress.complete()
	}

	return deployAtomicIntent(opts, image, projectName, userID, requestID)
}

// atomicIntentFallbackReason returns the first setting the durable intent
// endpoint cannot faithfully express. Unsupported settings fail closed so no
// setting is silently omitted and the retired legacy endpoint is never called.
func atomicIntentFallbackReason(opts DeploymentOptions) string {
	switch {
	case len(opts.Dependencies) > 0:
		return "dependent workload creation"
	case len(opts.WaitFor) > 0:
		return "dependency readiness declarations (--wait-for)"
	default:
		return ""
	}
}

func deployAtomicIntent(opts DeploymentOptions, image, projectName, userID, requestID string) (*api.CreateDeploymentResponse, error) {
	return deployAtomicIntentWithClient(opts, image, projectName, userID, requestID, context.GetCurrentOrgID(), atomicDeployClient{
		createIntent: api.CreateDeploymentIntent,
		getIngress:   api.GetIngressByDeploymentID,
		attachDomain: api.AttachDomain,
		sleep:        time.Sleep,
	})
}

type atomicDeployClient struct {
	createIntent func(api.DeploymentIntent, string) (*api.DeploymentIntentAccepted, error)
	getIngress   func(string) (*api.Ingress, error)
	attachDomain func(string, api.AttachDomainRequest) (*api.IngressAlias, error)
	sleep        func(time.Duration)
}

func deployAtomicIntentWithClient(opts DeploymentOptions, image, projectName, userID, requestID, currentOrgID string, client atomicDeployClient) (*api.CreateDeploymentResponse, error) {
	intent, err := buildAtomicDeploymentIntent(opts, image, projectName, userID)
	if err != nil {
		return nil, err
	}
	var orgID uuid.UUID
	if opts.Domain != "" {
		orgID, err = api.ParseUUID(strings.TrimSpace(currentOrgID))
		if err != nil {
			return nil, fmt.Errorf("cannot attach custom domain %q without a valid active organization ID: %w", opts.Domain, err)
		}
	}
	utils.PrintInfo("Deployment path: atomic intent")
	accepted, err := client.createIntent(intent, requestID)
	if err != nil {
		return nil, err
	}
	response := &api.CreateDeploymentResponse{
		DeploymentID: api.ToUUID(accepted.DeploymentID),
		AppLabel:     accepted.AppLabel,
		Intent:       accepted,
	}
	if opts.Domain == "" {
		return response, nil
	}

	ingress, err := waitForAtomicIngress(accepted.DeploymentID, client.getIngress, client.sleep)
	if err != nil {
		return nil, fmt.Errorf("deployment intent was accepted, but custom domain %q could not be attached: %w", opts.Domain, err)
	}
	alias, err := client.attachDomain(ingress.IngressID.String(), api.AttachDomainRequest{
		OrgID:      orgID,
		DomainName: opts.Domain,
	})
	if err != nil {
		return nil, fmt.Errorf("deployment intent was accepted, but attaching custom domain %q failed: %w", opts.Domain, err)
	}
	response.IngressID = ingress.IngressID
	response.Domain = alias.DomainName
	return response, nil
}

func waitForAtomicIngress(deploymentID string, lookup func(string) (*api.Ingress, error), sleep func(time.Duration)) (*api.Ingress, error) {
	const (
		attempts = 60
		interval = time.Second
	)
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		ingress, err := lookup(deploymentID)
		if err == nil && ingress != nil && ingress.IngressID != uuid.Nil {
			return ingress, nil
		}
		if err != nil {
			var statusErr *api.HTTPStatusError
			notFound := (errors.As(err, &statusErr) && statusErr.StatusCode == 404) ||
				strings.Contains(strings.ToLower(err.Error()), "not found")
			if !notFound {
				return nil, err
			}
			lastErr = err
		}
		if attempt+1 < attempts {
			sleep(interval)
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("timed out waiting for the default ingress: %w", lastErr)
	}
	return nil, errors.New("timed out waiting for the default ingress")
}

func buildAtomicDeploymentIntent(opts DeploymentOptions, image, projectName, userID string) (api.DeploymentIntent, error) {
	port, err := api.SafeInt32(opts.Port)
	if err != nil {
		return api.DeploymentIntent{}, utils.NewError(fmt.Sprintf("invalid port: %s", err.Error()), nil)
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
	replicas := opts.Replicas
	if replicas == 0 {
		replicas = 1
	}
	replicaCount, err := api.SafeInt32(replicas)
	if err != nil {
		return api.DeploymentIntent{}, utils.NewError(fmt.Sprintf("invalid replicas count: %s", err.Error()), nil)
	}

	deployment := api.Deployment{
		UserID:         api.ToUUID(userID),
		Type:           "production",
		Environment:    "production",
		Hostnames:      append([]string(nil), opts.Hostnames...),
		CpuRequest:     cpuRequest,
		CPULimit:       cpuLimit,
		MemoryRequest:  opts.Memory,
		MemoryLimit:    opts.Memory,
		Namespace:      opts.Organization,
		Port:           port,
		Image:          image,
		Zone:           opts.Zone,
		SSD:            "true",
		GPU:            "false",
		AppLabel:       projectName,
		Replicas:       replicaCount,
		EnvEnabled:     opts.EnvEnabled,
		VolumeEnabled:  opts.VolumeEnabled || len(opts.IntentVolumes) > 0,
		WaitFor:        opts.WaitFor,
		StrategyConfig: buildStrategyConfig(opts),
		TargetArch:     opts.TargetArch,
	}
	if opts.MulticlusterEnabled {
		schedule := map[string]string{
			"hourly": "0 * * * *",
			"daily":  "0 0 * * *",
			"weekly": "0 18 * * 6",
		}[opts.BackupSchedule]
		activePassive := opts.MulticlusterMode == "active-passive"
		priority := opts.BackupPriorityCluster
		if priority <= 0 {
			priority = 1
		}
		deployment.MulticlusterConfig = &api.MulticlusterConfig{
			Enabled: true, Mode: opts.MulticlusterMode,
			BackupEnabled: activePassive || opts.BackupEnabled, BackupSchedule: schedule,
			BackupRetention: opts.BackupRetention, BackupPriorityCluster: priority,
			FailoverEnabled: activePassive, RestoreOnFailover: activePassive,
		}
	}
	if opts.PDBConfig != nil && opts.PDBConfig.Enabled {
		deployment.PDBConfig = &api.PDBConfig{Enabled: true, Type: string(opts.PDBConfig.Type), MinAvailable: opts.PDBConfig.MinAvailable, Percent: opts.PDBConfig.Percent}
	} else if replicaCount > 1 {
		deployment.PDBConfig = &api.PDBConfig{Enabled: true, Type: "auto"}
	}
	if opts.HPAConfig != nil {
		deployment.HPAConfig = opts.HPAConfig
	}
	if opts.VPAConfig != nil {
		deployment.VPAConfig = opts.VPAConfig
	}

	volumes := append([]api.DeploymentIntentVolume(nil), opts.IntentVolumes...)
	if len(volumes) == 0 && opts.VolumeEnabled && opts.Volume != nil {
		volumes = append(volumes, api.DeploymentIntentVolume{
			VolumeName: projectName + "-volume", ClaimName: projectName + "-claim",
			StorageClass: opts.Volume.StorageClass, StorageSize: opts.Volume.StorageSize, MountPath: opts.Volume.MountPath,
		})
	}
	var environment []api.KeyValuePair
	if opts.Environment != nil {
		environment = append(environment, opts.Environment.KeyValues...)
	}
	return api.DeploymentIntent{
		Deployment:  deployment,
		Environment: environment,
		Config:      opts.DesiredStateConfig,
		Volumes:     volumes,
		Service:     &api.DeploymentIntentService{Name: projectName, Port: port},
		PublicRoute: &api.DeploymentIntentPublicRoute{Kind: "default_dns"},
	}, nil
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

// setSourceBuildTargetArch keeps build detection authoritative over any
// configured architecture that is only meaningful for pre-built images.
func setSourceBuildTargetArch(opts *DeploymentOptions, imageArch string) {
	opts.TargetArch = normalizeTargetArch(imageArch)
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
