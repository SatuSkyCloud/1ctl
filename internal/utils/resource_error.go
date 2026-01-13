package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ResourceExhaustedType represents the type of resource exhaustion
type ResourceExhaustedType string

const (
	QuotaCPU     ResourceExhaustedType = "quota_cpu"
	QuotaMemory  ResourceExhaustedType = "quota_memory"
	QuotaPods    ResourceExhaustedType = "quota_pods"
	QuotaStorage ResourceExhaustedType = "quota_storage"
	NodeCapacity ResourceExhaustedType = "node_capacity"
	LimitRange   ResourceExhaustedType = "limit_range"
)

// ResourceType represents the type of resource affected
type ResourceType string

const (
	ResourceCPU     ResourceType = "cpu"
	ResourceMemory  ResourceType = "memory"
	ResourcePods    ResourceType = "pods"
	ResourceStorage ResourceType = "storage"
)

// QuotaTier represents the organization's quota tier
type QuotaTier string

const (
	TierFree       QuotaTier = "free"
	TierStarter    QuotaTier = "starter"
	TierPro        QuotaTier = "pro"
	TierEnterprise QuotaTier = "enterprise"
)

// ResourceExhaustedError represents a resource quota/capacity failure from the API
type ResourceExhaustedError struct {
	Code        string                `json:"code"`
	Message     string                `json:"message"`
	Type        ResourceExhaustedType `json:"type"`
	Resource    ResourceType          `json:"resource"`
	Requested   string                `json:"requested"`
	Available   string                `json:"available"`
	Limit       string                `json:"limit"`
	Suggestion  string                `json:"suggestion"`
	CanUpgrade  bool                  `json:"can_upgrade"`
	CurrentTier QuotaTier             `json:"current_tier"`
	NextTier    QuotaTier             `json:"next_tier"`
}

// ResourceExhaustedAPIResponse wraps the API error response
type ResourceExhaustedAPIResponse struct {
	Success bool                   `json:"success"`
	Error   ResourceExhaustedError `json:"error"`
}

// TierLimits holds the limits for each tier
type TierLimits struct {
	CPU    string
	Memory string
	Pods   int
}

// TierLimitsMap maps quota tiers to their resource limits
var TierLimitsMap = map[QuotaTier]TierLimits{
	TierFree:       {CPU: "2 cores", Memory: "4Gi", Pods: 10},
	TierStarter:    {CPU: "4 cores", Memory: "8Gi", Pods: 25},
	TierPro:        {CPU: "8 cores", Memory: "16Gi", Pods: 50},
	TierEnterprise: {CPU: "16 cores", Memory: "32Gi", Pods: 100},
}

// IsResourceExhaustedStatus checks if an HTTP status code indicates resource exhaustion (422)
func IsResourceExhaustedStatus(statusCode int) bool {
	return statusCode == http.StatusUnprocessableEntity
}

// ParseResourceExhaustedError attempts to parse a resource exhausted error from an HTTP response
func ParseResourceExhaustedError(resp *http.Response) (*ResourceExhaustedError, error) {
	if !IsResourceExhaustedStatus(resp.StatusCode) {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp ResourceExhaustedAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, nil // Not a resource exhausted error format
	}

	if apiResp.Error.Code != "RESOURCE_EXHAUSTED" {
		return nil, nil
	}

	return &apiResp.Error, nil
}

// ParseResourceExhaustedFromBytes attempts to parse a resource exhausted error from response bytes
func ParseResourceExhaustedFromBytes(body []byte, statusCode int) (*ResourceExhaustedError, error) {
	if !IsResourceExhaustedStatus(statusCode) {
		return nil, nil
	}

	var apiResp ResourceExhaustedAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, nil // Not a resource exhausted error format
	}

	if apiResp.Error.Code != "RESOURCE_EXHAUSTED" {
		return nil, nil
	}

	return &apiResp.Error, nil
}

// GetResourceDisplayName returns a human-readable name for a resource type
func GetResourceDisplayName(resource ResourceType) string {
	switch resource {
	case ResourceCPU:
		return "CPU"
	case ResourceMemory:
		return "Memory"
	case ResourcePods:
		return "Pods"
	case ResourceStorage:
		return "Storage"
	default:
		return string(resource)
	}
}

// GetExhaustionTypeDisplayName returns a human-readable name for an exhaustion type
func GetExhaustionTypeDisplayName(exhaustionType ResourceExhaustedType) string {
	switch exhaustionType {
	case QuotaCPU:
		return "CPU Quota Exceeded"
	case QuotaMemory:
		return "Memory Quota Exceeded"
	case QuotaPods:
		return "Pod Limit Exceeded"
	case QuotaStorage:
		return "Storage Quota Exceeded"
	case NodeCapacity:
		return "Node Capacity Full"
	case LimitRange:
		return "Container Limit Exceeded"
	default:
		return "Resource Limit Exceeded"
	}
}

// PrintResourceExhaustedError prints a formatted resource exhausted error to the console
func PrintResourceExhaustedError(err *ResourceExhaustedError) {
	// Header
	fmt.Println()
	fmt.Println(ErrorColor("Deployment Failed: Resource Limit Reached"))
	fmt.Println(DividerColor("────────────────────────────────────────────"))
	fmt.Println()

	// Error type
	fmt.Printf("%s: %s\n", BoldColor("Error Type"), GetExhaustionTypeDisplayName(err.Type))
	fmt.Println()

	// Resource comparison table
	fmt.Println(BoldColor("Resource Details:"))
	fmt.Println()

	PrintTable(
		[]string{"", "Value"},
		[][]string{
			{"Resource", GetResourceDisplayName(err.Resource)},
			{"Requested", err.Requested},
			{"Available", WarnColor(err.Available)},
			{"Limit", fmt.Sprintf("%s (%s tier)", err.Limit, err.CurrentTier)},
		},
	)

	fmt.Println()

	// Suggestion
	fmt.Printf("%s %s\n", InfoColor("Suggestion:"), err.Suggestion)

	// Upgrade prompt if available
	if err.CanUpgrade && err.NextTier != "" {
		fmt.Println()
		fmt.Println(SuccessColor("Upgrade Available:"))

		nextLimits, ok := TierLimitsMap[err.NextTier]
		if ok {
			fmt.Printf("  Upgrade to %s tier for:\n", BoldColor(string(err.NextTier)))
			fmt.Printf("    - CPU: %s\n", nextLimits.CPU)
			fmt.Printf("    - Memory: %s\n", nextLimits.Memory)
			fmt.Printf("    - Pods: %d\n", nextLimits.Pods)
		}

		fmt.Println()
		fmt.Printf("  %s Visit https://cloud.satusky.com/billing to upgrade\n", InfoColor("->"))
	}

	fmt.Println()
}

// ResourceExhaustedCLIError wraps ResourceExhaustedError as a CLIError
type ResourceExhaustedCLIError struct {
	ResourceError *ResourceExhaustedError
}

func (e *ResourceExhaustedCLIError) Error() string {
	return fmt.Sprintf("resource exhausted: %s - %s", e.ResourceError.Type, e.ResourceError.Suggestion)
}

// NewResourceExhaustedCLIError creates a new ResourceExhaustedCLIError
func NewResourceExhaustedCLIError(err *ResourceExhaustedError) *ResourceExhaustedCLIError {
	return &ResourceExhaustedCLIError{
		ResourceError: err,
	}
}

// HandleResourceExhaustedError prints the error and returns it wrapped
func HandleResourceExhaustedError(err *ResourceExhaustedError) error {
	PrintResourceExhaustedError(err)
	return NewResourceExhaustedCLIError(err)
}
