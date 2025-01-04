package helpers

import (
	"testing"
)

// ValidatorTestCase represents a common test case structure for validators
type ValidatorTestCase struct {
	Name    string
	Input   string
	WantErr bool
}

// RunValidatorTests runs common validator test cases
func RunValidatorTests(t *testing.T, testCases []ValidatorTestCase, validatorFunc func(string) error) {
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			err := validatorFunc(tt.Input)
			if (err != nil) != tt.WantErr {
				t.Errorf("Validation error = %v, wantErr %v", err, tt.WantErr)
			}
		})
	}
}
