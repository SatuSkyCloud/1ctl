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
