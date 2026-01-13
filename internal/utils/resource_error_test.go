package utils

import (
	"testing"
)

func TestIsResourceExhaustedStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"422 Unprocessable Entity", 422, true},
		{"200 OK", 200, false},
		{"400 Bad Request", 400, false},
		{"500 Internal Server Error", 500, false},
		{"403 Forbidden", 403, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsResourceExhaustedStatus(tt.statusCode)
			if result != tt.expected {
				t.Errorf("IsResourceExhaustedStatus(%d) = %v, want %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}

func TestParseResourceExhaustedFromBytes(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		statusCode int
		wantError  bool
		wantNil    bool
	}{
		{
			name: "Valid resource exhausted error",
			body: `{
				"success": false,
				"error": {
					"code": "RESOURCE_EXHAUSTED",
					"message": "CPU quota exceeded",
					"type": "quota_cpu",
					"resource": "cpu",
					"requested": "4",
					"available": "1",
					"limit": "2",
					"suggestion": "Reduce CPU requests or upgrade your plan",
					"can_upgrade": true,
					"current_tier": "starter",
					"next_tier": "pro"
				}
			}`,
			statusCode: 422,
			wantError:  false,
			wantNil:    false,
		},
		{
			name:       "Non-422 status code",
			body:       `{"success": false, "error": {"code": "RESOURCE_EXHAUSTED"}}`,
			statusCode: 400,
			wantError:  false,
			wantNil:    true,
		},
		{
			name: "422 but different error code",
			body: `{
				"success": false,
				"error": {
					"code": "VALIDATION_ERROR",
					"message": "Invalid input"
				}
			}`,
			statusCode: 422,
			wantError:  false,
			wantNil:    true,
		},
		{
			name:       "Invalid JSON",
			body:       `{invalid json}`,
			statusCode: 422,
			wantError:  false,
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseResourceExhaustedFromBytes([]byte(tt.body), tt.statusCode)

			if tt.wantError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.wantNil && result != nil {
				t.Error("Expected nil result but got non-nil")
			}
			if !tt.wantNil && result == nil {
				t.Error("Expected non-nil result but got nil")
			}
		})
	}
}

func TestParseResourceExhaustedFromBytes_Fields(t *testing.T) {
	body := `{
		"success": false,
		"error": {
			"code": "RESOURCE_EXHAUSTED",
			"message": "CPU quota exceeded",
			"type": "quota_cpu",
			"resource": "cpu",
			"requested": "4",
			"available": "1",
			"limit": "2",
			"suggestion": "Reduce CPU requests or upgrade your plan",
			"can_upgrade": true,
			"current_tier": "starter",
			"next_tier": "pro"
		}
	}`

	result, err := ParseResourceExhaustedFromBytes([]byte(body), 422)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Validate fields
	if result.Code != "RESOURCE_EXHAUSTED" {
		t.Errorf("Code = %v, want RESOURCE_EXHAUSTED", result.Code)
	}
	if result.Type != QuotaCPU {
		t.Errorf("Type = %v, want quota_cpu", result.Type)
	}
	if result.Resource != ResourceCPU {
		t.Errorf("Resource = %v, want cpu", result.Resource)
	}
	if result.Requested != "4" {
		t.Errorf("Requested = %v, want 4", result.Requested)
	}
	if result.Available != "1" {
		t.Errorf("Available = %v, want 1", result.Available)
	}
	if result.Limit != "2" {
		t.Errorf("Limit = %v, want 2", result.Limit)
	}
	if !result.CanUpgrade {
		t.Error("CanUpgrade = false, want true")
	}
	if result.CurrentTier != TierStarter {
		t.Errorf("CurrentTier = %v, want starter", result.CurrentTier)
	}
	if result.NextTier != TierPro {
		t.Errorf("NextTier = %v, want pro", result.NextTier)
	}
}

func TestGetResourceDisplayName(t *testing.T) {
	tests := []struct {
		resource ResourceType
		expected string
	}{
		{ResourceCPU, "CPU"},
		{ResourceMemory, "Memory"},
		{ResourcePods, "Pods"},
		{ResourceStorage, "Storage"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.resource), func(t *testing.T) {
			result := GetResourceDisplayName(tt.resource)
			if result != tt.expected {
				t.Errorf("GetResourceDisplayName(%s) = %v, want %v", tt.resource, result, tt.expected)
			}
		})
	}
}

func TestGetExhaustionTypeDisplayName(t *testing.T) {
	tests := []struct {
		exhaustionType ResourceExhaustedType
		expected       string
	}{
		{QuotaCPU, "CPU Quota Exceeded"},
		{QuotaMemory, "Memory Quota Exceeded"},
		{QuotaPods, "Pod Limit Exceeded"},
		{QuotaStorage, "Storage Quota Exceeded"},
		{NodeCapacity, "Node Capacity Full"},
		{LimitRange, "Container Limit Exceeded"},
		{"unknown", "Resource Limit Exceeded"},
	}

	for _, tt := range tests {
		t.Run(string(tt.exhaustionType), func(t *testing.T) {
			result := GetExhaustionTypeDisplayName(tt.exhaustionType)
			if result != tt.expected {
				t.Errorf("GetExhaustionTypeDisplayName(%s) = %v, want %v", tt.exhaustionType, result, tt.expected)
			}
		})
	}
}

func TestResourceExhaustedCLIError(t *testing.T) {
	resourceErr := &ResourceExhaustedError{
		Code:        "RESOURCE_EXHAUSTED",
		Type:        QuotaCPU,
		Suggestion:  "Reduce CPU requests",
		CurrentTier: TierStarter,
	}

	cliErr := NewResourceExhaustedCLIError(resourceErr)

	if cliErr.ResourceError != resourceErr {
		t.Error("ResourceError not set correctly")
	}

	errorMsg := cliErr.Error()
	if errorMsg == "" {
		t.Error("Error() returned empty string")
	}
}

func TestTierLimitsMap(t *testing.T) {
	// Ensure all tiers have limits defined
	tiers := []QuotaTier{TierFree, TierStarter, TierPro, TierEnterprise}

	for _, tier := range tiers {
		limits, ok := TierLimitsMap[tier]
		if !ok {
			t.Errorf("TierLimitsMap missing entry for tier %s", tier)
			continue
		}
		if limits.CPU == "" {
			t.Errorf("TierLimitsMap[%s].CPU is empty", tier)
		}
		if limits.Memory == "" {
			t.Errorf("TierLimitsMap[%s].Memory is empty", tier)
		}
		if limits.Pods <= 0 {
			t.Errorf("TierLimitsMap[%s].Pods = %d, want > 0", tier, limits.Pods)
		}
	}
}
