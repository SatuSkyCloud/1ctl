package api

import "testing"

func TestIngressDNSVerified(t *testing.T) {
	verified := &DNSCondition{Status: DNSConditionStatusVerified}
	tests := []struct {
		name   string
		status *DNSStatusResponse
		want   bool
	}{
		{
			name:   "older backend resolved without condition",
			status: &DNSStatusResponse{Status: DNSStatusResolved},
			want:   true,
		},
		{
			name:   "older backend propagating without condition",
			status: &DNSStatusResponse{Status: DNSStatusPropagating},
			want:   false,
		},
		{
			name:   "verified condition is authoritative",
			status: &DNSStatusResponse{Status: DNSStatusResolved, Condition: verified},
			want:   true,
		},
		{
			name:   "pending condition does not pass even when legacy status is resolved",
			status: &DNSStatusResponse{Status: DNSStatusResolved, Condition: &DNSCondition{Status: DNSConditionStatusPending}},
			want:   false,
		},
		{
			name:   "NXDOMAIN condition does not pass",
			status: &DNSStatusResponse{Status: DNSStatusResolved, Condition: &DNSCondition{Status: DNSConditionStatusNXDomain}},
			want:   false,
		},
		{
			name:   "wrong target condition does not pass",
			status: &DNSStatusResponse{Status: DNSStatusResolved, Condition: &DNSCondition{Status: DNSConditionStatusWrongTarget}},
			want:   false,
		},
		{
			name:   "error condition does not pass",
			status: &DNSStatusResponse{Status: DNSStatusResolved, Condition: &DNSCondition{Status: DNSConditionStatusError}},
			want:   false,
		},
		{name: "nil status does not pass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ingressDNSVerified(tt.status); got != tt.want {
				t.Fatalf("ingressDNSVerified(%+v) = %t, want %t", tt.status, got, tt.want)
			}
		})
	}
}
