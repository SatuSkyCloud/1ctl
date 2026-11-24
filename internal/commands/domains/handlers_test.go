package domains

import (
	"encoding/json"
	"strings"
	"testing"

	"1ctl/internal/api"
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

func TestDomainEdgeText(t *testing.T) {
	cases := []struct {
		edge api.DomainEdgeStatus
		want string
	}{
		{api.DomainEdgeStatus{Status: "published", AutoPublished: true}, "published (automatic)"},
		{api.DomainEdgeStatus{Status: "published"}, "published"},
		{api.DomainEdgeStatus{Status: "pending", Message: "waiting for DNS"}, "publishing - waiting for DNS"},
		{api.DomainEdgeStatus{Status: "not_eligible", Message: "zone example.com is not managed"}, "not_eligible - zone example.com is not managed"},
		{api.DomainEdgeStatus{Status: "not_requested"}, "not published"},
		{api.DomainEdgeStatus{}, "unknown"},
	}
	for _, tc := range cases {
		if got := domainEdgeText(tc.edge); got != tc.want {
			t.Errorf("domainEdgeText(%+v) = %q, want %q", tc.edge, got, tc.want)
		}
	}
}

func TestDomainStatusDecodesEdgeBlock(t *testing.T) {
	var status api.DomainStatusResponse
	if err := json.Unmarshal([]byte(`{"domain_name":"a.example.com","edge":{"status":"published","auto_published":true}}`), &status); err != nil {
		t.Fatal(err)
	}
	if status.Edge == nil || status.Edge.Status != "published" || !status.Edge.AutoPublished {
		t.Fatalf("edge block not decoded: %+v", status.Edge)
	}
	var old api.DomainStatusResponse
	if err := json.Unmarshal([]byte(`{"domain_name":"a.example.com"}`), &old); err != nil || old.Edge != nil {
		t.Fatalf("older backend response should leave Edge nil, got %+v err %v", old.Edge, err)
	}
}
