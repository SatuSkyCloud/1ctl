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
func (cm *CleanupManager) Cleanup() []error {
	var errors []error

	// Cleanup in reverse order to handle dependencies
	for i := len(cm.resources) - 1; i >= 0; i-- {
		resource := cm.resources[i]
		if err := cm.cleanupResource(resource); err != nil {
			errors = append(errors, utils.NewError(fmt.Sprintf("failed to cleanup %s %s: %s", resource.Type, resource.Name, err.Error()), nil))
		}
	}

	return errors
}

func (cm *CleanupManager) cleanupResource(resource Resource) error {
	utils.PrintWarning("Cleaning up %s: %s...\n", resource.Type, resource.Name)

	switch resource.Type {
	case ResourceDeployment:
		return api.DeleteDeployment(resource.ID)
	case ResourceService:
		return api.DeleteService(resource.ID)
	case ResourceIngress:
		return api.DeleteIngress(resource.ID)
	case ResourceVolume:
		// The backend's POST /volumes/create has no DELETE counterpart yet.
		// We still register volumes with the manager so they appear in cleanup
		// logs (operators can manually clean up the PVC), but the actual delete
		// is a no-op until the backend ships volume deletion.
		utils.PrintWarning("Volume %s (%s) cannot be deleted via CLI — manual PVC cleanup required.", resource.Name, resource.ID)
		return nil
	case ResourceSecret:
		// TODO: Add secret deletion when API supports it
		return nil
	case ResourceEnv:
		return api.DeleteEnvironment(resource.ID)
	default:
		return utils.NewError(fmt.Sprintf("unknown resource type: %s", resource.Type), nil)
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
