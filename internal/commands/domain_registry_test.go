package commands

import (
	"testing"

	"1ctl/internal/api"

	"github.com/urfave/cli/v3"
)

// ---------------------------------------------------------------------------
// dnsTestFlags returns the common set of flag names used by DNS create/update commands.
func dnsTestFlags() []string {
	return []string{"type", "name", "data", "ttl", "priority", "port", "weight", "flags", "tag"}
}

// ---------------------------------------------------------------------------
// Tests for dnsCreateRequestFromFlags
// ---------------------------------------------------------------------------

func TestDnsCreateRequestFromFlags(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(cmd *cli.Command)
		want   api.DNSRecordCreateRequest
	}{
		{
			name: "all required fields",
			setup: func(cmd *cli.Command) {
				cmd.Set("type", "A")
				cmd.Set("name", "@")
				cmd.Set("data", "1.2.3.4")
			},
			want: api.DNSRecordCreateRequest{
				Type: "A",
				Name: "@",
				Data: "1.2.3.4",
			},
		},
		{
			name: "with ttl",
			setup: func(cmd *cli.Command) {
				cmd.Set("type", "CNAME")
				cmd.Set("name", "app")
				cmd.Set("data", "target.example.com")
				cmd.Set("ttl", "300")
			},
			want: api.DNSRecordCreateRequest{
				Type: "CNAME",
				Name: "app",
				Data: "target.example.com",
				TTL:  intPtr(300),
			},
		},
		{
			name: "SRV record with optional fields",
			setup: func(cmd *cli.Command) {
				cmd.Set("type", "SRV")
				cmd.Set("name", "_sip._tcp")
				cmd.Set("data", "10 5060 sip.example.com")
				cmd.Set("ttl", "600")
				cmd.Set("priority", "10")
				cmd.Set("port", "5060")
				cmd.Set("weight", "5")
			},
			want: api.DNSRecordCreateRequest{
				Type:     "SRV",
				Name:     "_sip._tcp",
				Data:     "10 5060 sip.example.com",
				TTL:      intPtr(600),
				Priority: intPtr(10),
				Port:     intPtr(5060),
				Weight:   intPtr(5),
			},
		},
		{
			name: "type is uppercased",
			setup: func(cmd *cli.Command) {
				cmd.Set("type", "a")
				cmd.Set("name", "@")
				cmd.Set("data", "1.2.3.4")
			},
			want: api.DNSRecordCreateRequest{
				Type: "A",
				Name: "@",
				Data: "1.2.3.4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cli.Command{Flags: dnsFlagsForTest()}
			tt.setup(cmd)
			got := dnsCreateRequestFromFlags(cmd)
			assertDNSRecordCreateEqual(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Tests for dnsUpdateRequestFromFlags
// ---------------------------------------------------------------------------

func TestDnsUpdateRequestFromFlags(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(cmd *cli.Command)
		want   api.DNSRecordUpdateRequest
	}{
		{
			name: "single field update (data only)",
			setup: func(cmd *cli.Command) {
				cmd.Set("data", "5.6.7.8")
			},
			want: api.DNSRecordUpdateRequest{
				Data: strPtr("5.6.7.8"),
			},
		},
		{
			name: "multiple field update (data + ttl)",
			setup: func(cmd *cli.Command) {
				cmd.Set("data", "10.0.0.1")
				cmd.Set("ttl", "1800")
			},
			want: api.DNSRecordUpdateRequest{
				Data: strPtr("10.0.0.1"),
				TTL:  intPtr(1800),
			},
		},
		{
			name: "all fields set",
			setup: func(cmd *cli.Command) {
				cmd.Set("type", "A")
				cmd.Set("name", "@")
				cmd.Set("data", "1.2.3.4")
				cmd.Set("ttl", "300")
				cmd.Set("priority", "5")
				cmd.Set("port", "8080")
				cmd.Set("weight", "10")
				cmd.Set("flags", "128")
				cmd.Set("tag", "issue")
			},
			want: api.DNSRecordUpdateRequest{
				Type:     strPtr("A"),
				Name:     strPtr("@"),
				Data:     strPtr("1.2.3.4"),
				TTL:      intPtr(300),
				Priority: intPtr(5),
				Port:     intPtr(8080),
				Weight:   intPtr(10),
				Flags:    intPtr(128),
				Tag:      strPtr("issue"),
			},
		},
		{
			name: "empty request (no flags set)",
			setup: func(cmd *cli.Command) {
			},
			want: api.DNSRecordUpdateRequest{},
		},
		{
			name: "type is uppercased in update",
			setup: func(cmd *cli.Command) {
				cmd.Set("type", "aaaa")
			},
			want: api.DNSRecordUpdateRequest{
				Type: strPtr("AAAA"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cli.Command{Flags: dnsFlagsForTest()}
			tt.setup(cmd)
			got := dnsUpdateRequestFromFlags(cmd)
			assertDNSRecordUpdateEqual(t, tt.want, got)
		})
	}
}

// dnsFlagsForTest returns flags used by DNS create/update commands.
func dnsFlagsForTest() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "type"},
		&cli.StringFlag{Name: "name"},
		&cli.StringFlag{Name: "data"},
		&cli.IntFlag{Name: "ttl"},
		&cli.IntFlag{Name: "priority"},
		&cli.IntFlag{Name: "port"},
		&cli.IntFlag{Name: "weight"},
		&cli.IntFlag{Name: "flags"},
		&cli.StringFlag{Name: "tag"},
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func intPtr(v int) *int     { return &v }
func strPtr(v string) *string { return &v }

func assertDNSRecordCreateEqual(t *testing.T, want, got api.DNSRecordCreateRequest) {
	t.Helper()
	if want.Type != got.Type {
		t.Errorf("Type: want %q, got %q", want.Type, got.Type)
	}
	if want.Name != got.Name {
		t.Errorf("Name: want %q, got %q", want.Name, got.Name)
	}
	if want.Data != got.Data {
		t.Errorf("Data: want %q, got %q", want.Data, got.Data)
	}
	assertIntPtrEqual(t, "TTL", want.TTL, got.TTL)
	assertIntPtrEqual(t, "Priority", want.Priority, got.Priority)
	assertIntPtrEqual(t, "Port", want.Port, got.Port)
	assertIntPtrEqual(t, "Weight", want.Weight, got.Weight)
	assertIntPtrEqual(t, "Flags", want.Flags, got.Flags)
	assertStrPtrEqual(t, "Tag", want.Tag, got.Tag)
}

func assertDNSRecordUpdateEqual(t *testing.T, want, got api.DNSRecordUpdateRequest) {
	t.Helper()
	assertStrPtrEqual(t, "Type", want.Type, got.Type)
	assertStrPtrEqual(t, "Name", want.Name, got.Name)
	assertStrPtrEqual(t, "Data", want.Data, got.Data)
	assertIntPtrEqual(t, "TTL", want.TTL, got.TTL)
	assertIntPtrEqual(t, "Priority", want.Priority, got.Priority)
	assertIntPtrEqual(t, "Port", want.Port, got.Port)
	assertIntPtrEqual(t, "Weight", want.Weight, got.Weight)
	assertIntPtrEqual(t, "Flags", want.Flags, got.Flags)
	assertStrPtrEqual(t, "Tag", want.Tag, got.Tag)
}

func assertIntPtrEqual(t *testing.T, field string, want, got *int) {
	t.Helper()
	switch {
	case want == nil && got == nil:
		return
	case want == nil:
		t.Errorf("%s: want nil, got %d", field, *got)
	case got == nil:
		t.Errorf("%s: want %d, got nil", field, *want)
	case *want != *got:
		t.Errorf("%s: want %d, got %d", field, *want, *got)
	}
}

func assertStrPtrEqual(t *testing.T, field string, want, got *string) {
	t.Helper()
	switch {
	case want == nil && got == nil:
		return
	case want == nil:
		t.Errorf("%s: want nil, got %q", field, *got)
	case got == nil:
		t.Errorf("%s: want %q, got nil", field, *want)
	case *want != *got:
		t.Errorf("%s: want %q, got %q", field, *want, *got)
	}
}
