package validator

import (
	"testing"
)

type validatorTestCase struct {
	name    string
	input   string
	wantErr bool
}

func runValidatorTests(t *testing.T, testCases []validatorTestCase, validatorFunc func(string) error) {
	t.Helper()
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := validatorFunc(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCPU(t *testing.T) {
	tests := []validatorTestCase{
		{name: "valid whole CPU", input: "1", wantErr: false},
		{name: "valid millicpu", input: "100m", wantErr: false},
		{name: "invalid format", input: "1g", wantErr: true},
		{name: "zero CPU", input: "0", wantErr: true},
		{name: "negative CPU", input: "-1", wantErr: true},
	}

	runValidatorTests(t, tests, ValidateCPU)
}

func TestValidateMemory(t *testing.T) {
	tests := []validatorTestCase{
		{name: "valid Mi", input: "512Mi", wantErr: false},
		{name: "valid Gi", input: "2Gi", wantErr: false},
		{name: "invalid unit", input: "512MB", wantErr: true},
		{name: "zero value", input: "0Mi", wantErr: true},
		{name: "negative value", input: "-1Mi", wantErr: true},
	}

	runValidatorTests(t, tests, ValidateMemory)
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{"valid domain", "example.com", false},
		{"valid subdomain", "sub.example.com", false},
		{"valid wildcard", "*.example.com", false},
		{"valid multi-level", "a.b.example.com", false},
		{"empty domain", "", false},
		{"invalid format", "invalid", true},
		{"invalid chars", "test!.com", true},
		{"missing tld", "example", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDomain(%q) error = %v, wantErr %v", tt.domain, err, tt.wantErr)
			}
		})
	}
}
