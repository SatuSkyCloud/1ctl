package cleanup

import (
	"1ctl/internal/api"
	"1ctl/internal/utils"
	"fmt"
	"strings"
)

type ResourceType string

const (
	ResourceDeployment ResourceType = "deployment"
	ResourceService    ResourceType = "service"
	ResourceIngress    ResourceType = "ingress"
	ResourceVolume     ResourceType = "volume"
	ResourceSecret     ResourceType = "secret"
	ResourceEnv        ResourceType = "environment"
)

type Resource struct {
	Type ResourceType
	ID   string
	Name string
}

type CleanupManager struct {
	resources []Resource
}

func NewCleanupManager() *CleanupManager {
	return &CleanupManager{
		resources: make([]Resource, 0),
	}
}

func (cm *CleanupManager) AddResource(resourceType ResourceType, id, name string) {
	cm.resources = append(cm.resources, Resource{
		Type: resourceType,
		ID:   id,
		Name: name,
	})
}

// TODO: proper cleanup upon error (resources) on orchestrator
func (cm *CleanupManager) Cleanup(payload interface{}) []error {
	var errors []error

	// Cleanup in reverse order to handle dependencies
	for i := len(cm.resources) - 1; i >= 0; i-- {
		resource := cm.resources[i]
		if err := cm.cleanupResource(payload, resource); err != nil {
			errors = append(errors, fmt.Errorf("failed to cleanup %s %s: %w", resource.Type, resource.Name, err))
		}
	}

	return errors
}

func (cm *CleanupManager) cleanupResource(payload interface{}, resource Resource) error {
	utils.PrintWarning("Cleaning up %s: %s...\n", resource.Type, resource.Name)

	switch resource.Type {
	case ResourceDeployment:
		return api.DeleteDeployment(payload, resource.ID)
	case ResourceService:
		return api.DeleteService(payload, resource.ID)
	case ResourceIngress:
		return api.DeleteIngress(payload, resource.ID)
	case ResourceVolume:
		// TODO: Add volume deletion when API supports it
		return nil
	case ResourceSecret:
		// TODO: Add secret deletion when API supports it
		return nil
	case ResourceEnv:
		return api.DeleteEnvironment(payload, resource.ID)
	default:
		return fmt.Errorf("unknown resource type: %s", resource.Type)
	}
}

func FormatCleanupErrors(errors []error) string {
	if len(errors) == 0 {
		return ""
	}

	var messages []string
	for _, err := range errors {
		messages = append(messages, err.Error())
	}

	return fmt.Sprintf("Cleanup errors:\n%s", strings.Join(messages, "\n"))
}

// Make cleanupFuncs package-visible for testing
var cleanupFuncs []func()

// RegisterCleanupFunc adds a cleanup function to be run later
func RegisterCleanupFunc(f func()) {
	if cleanupFuncs == nil {
		cleanupFuncs = make([]func(), 0)
	}
	cleanupFuncs = append(cleanupFuncs, f)
}

// RunCleanup executes all registered cleanup functions
func RunCleanup() {
	if cleanupFuncs == nil {
		return
	}
	for _, f := range cleanupFuncs {
		f()
	}
	cleanupFuncs = nil // Clear the slice after running
}

// ResetCleanup resets the cleanup functions (useful for testing)
func ResetCleanup() {
	cleanupFuncs = nil
}
