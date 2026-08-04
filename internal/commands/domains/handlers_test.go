package domains

import (
	"strings"
	"testing"
)

func TestDetachedDomainHintsUsePositionalDomain(t *testing.T) {
	command := domainAttachCommand("example.com")
	if command != "1ctl domains add example.com --app <app>" {
		t.Fatalf("domainAttachCommand() = %q, want positional domain hint", command)
	}
	if strings.Contains(command, "--domain") {
		t.Fatalf("domainAttachCommand() = %q, contains nonexistent --domain flag", command)
	}
}

func TestManagedDomainValidationUsesPositionalSyntax(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"add", handleManagedDomainsAdd(nil, domainsManagedAddInput{})},
		{"verify", handleManagedDomainsVerify(nil, domainsManagedVerifyInput{})},
		{"delete", handleManagedDomainsDelete(nil, domainsManagedDeleteInput{})},
	}
	for _, tt := range tests {
		if tt.err == nil {
			t.Fatalf("%s validation error = nil", tt.name)
		}
		if strings.Contains(tt.err.Error(), "--domain") {
			t.Fatalf("%s validation error advertises nonexistent flag: %q", tt.name, tt.err)
		}
		if !strings.Contains(tt.err.Error(), "<domain") {
			t.Fatalf("%s validation error lacks positional usage: %q", tt.name, tt.err)
		}
	}
}
