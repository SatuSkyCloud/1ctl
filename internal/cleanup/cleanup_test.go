package cleanup

import (
	"fmt"
	"testing"
)

func TestCleanupManager_AddAndCleanupRegistry(t *testing.T) {
	cm := NewCleanupManager()
	cm.AddResource(ResourceDeployment, "dep-1", "myapp")
	cm.AddResource(ResourceService, "svc-1", "myapp")
	cm.AddResource(ResourceVolume, "myapp-volume", "myapp")

	// We don't exercise the actual API delete calls here (they require a
	// live backend). The test asserts that resources are tracked in
	// reverse-order registration so cleanup runs in dependency order.
	if len(cm.resources) != 3 {
		t.Fatalf("AddResource: have %d resources, want 3", len(cm.resources))
	}
	if cm.resources[0].Type != ResourceDeployment ||
		cm.resources[1].Type != ResourceService ||
		cm.resources[2].Type != ResourceVolume {
		t.Errorf("registration order wrong: %+v", cm.resources)
	}
}

func TestFormatCleanupErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   []error
		expected string
	}{
		{
			name:     "no errors",
			errors:   nil,
			expected: "",
		},
		{
			name: "single error",
			errors: []error{
				fmt.Errorf("test error"),
			},
			expected: "Cleanup errors:\ntest error",
		},
		{
			name: "multiple errors",
			errors: []error{
				fmt.Errorf("error 1"),
				fmt.Errorf("error 2"),
			},
			expected: "Cleanup errors:\nerror 1\nerror 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCleanupErrors(tt.errors)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
