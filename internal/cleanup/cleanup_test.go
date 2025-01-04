package cleanup

import (
	"fmt"
	"testing"
)

func TestRegisterCleanupFunc(t *testing.T) {
	t.Skip("Skipping cleanup tests for now")
}

func TestRunCleanup(t *testing.T) {
	t.Skip("Skipping cleanup tests for now")
}

func TestCleanupManager(t *testing.T) {
	t.Skip("Skipping cleanup tests for now")
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
