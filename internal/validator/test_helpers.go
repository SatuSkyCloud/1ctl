package validator

import "testing"

// TestCase represents a common test case structure for validators
type TestCase struct {
	Name    string
	Input   string
	WantErr bool
}

// RunTests runs common validator test cases
func RunTests(t *testing.T, testCases []TestCase, validatorFunc func(string) error) {
	t.Helper()
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			err := validatorFunc(tt.Input)
			if (err != nil) != tt.WantErr {
				t.Errorf("Validation error = %v, wantErr %v", err, tt.WantErr)
			}
		})
	}
}
