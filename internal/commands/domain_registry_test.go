package commands

import (
	"flag"
	"testing"

	"1ctl/internal/api"

	"github.com/urfave/cli/v2"
)

// testContext creates a *cli.Context that simulates how urfave/cli parses arguments.
//   - args: the full command-line arguments (flag.Parse stops at first non-flag arg)
//   - parsedFlags: flags that urfave/cli parsed before the first positional arg
//     (values are set via ctx.Set so c.IsSet returns true)
//   - flagNames: the names of flags defined on the underlying FlagSet
//
// After flag.Parse(args):
//   - c.Args().Slice() contains all args after the first positional arg
//     (including any flag-like args that weren't parsed)
//   - c.IsSet(name) is true only for names in parsedFlags
//   - c.String(name) returns the value from the FlagSet (set either by Parse or by ctx.Set)
func testContext(t *testing.T, args []string, parsedFlags map[string]string, flagNames ...string) *cli.Context {
	t.Helper()
	app := cli.NewApp()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, name := range flagNames {
		fs.String(name, "", "")
	}
	_ = fs.Parse(args)
	ctx := cli.NewContext(app, fs, nil)
	for name, value := range parsedFlags {
		if err := ctx.Set(name, value); err != nil {
			t.Fatalf("ctx.Set(%q, %q): %v", name, value, err)
		}
	}
	return ctx
}

// dnsTestFlags returns the common set of flag names used by DNS create/update commands.
func dnsTestFlags() []string {
	return []string{"type", "name", "data", "ttl", "priority", "port", "weight", "flags", "tag"}
}

// ---------------------------------------------------------------------------
// TestFlagValueFromArgs
// ---------------------------------------------------------------------------

func TestFlagValueFromArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		parsedFlags map[string]string
		query       string // flag name to look up
		want        string
	}{
		{
			name:  "flag before positional",
			args:  []string{"--type", "A", "--data", "1.2.3.4", "domain.com", "record-id"},
			query: "type",
			want:  "A",
			parsedFlags: map[string]string{"type": "A", "data": "1.2.3.4"},
		},
		{
			name:  "another flag before positional",
			args:  []string{"--type", "A", "--data", "1.2.3.4", "domain.com", "record-id"},
			query: "data",
			want:  "1.2.3.4",
			parsedFlags: map[string]string{"type": "A", "data": "1.2.3.4"},
		},
		{
			name:  "flag after positional",
			args:  []string{"domain.com", "record-id", "--data", "5.6.7.8", "--ttl", "600"},
			query: "data",
			want:  "5.6.7.8",
		},
		{
			name:  "ttl after positional",
			args:  []string{"domain.com", "record-id", "--data", "5.6.7.8", "--ttl", "600"},
			query: "ttl",
			want:  "600",
		},
		{
			name:  "mixed flags before and after positional",
			args:  []string{"--type", "A", "domain.com", "--data", "1.2.3.4", "record-id", "--ttl", "300"},
			query: "type",
			want:  "A",
			parsedFlags: map[string]string{"type": "A"},
		},
		{
			name:  "data in mixed scenario (after positional, not parsed)",
			args:  []string{"--type", "A", "domain.com", "--data", "1.2.3.4", "record-id", "--ttl", "300"},
			query: "data",
			want:  "1.2.3.4",
			parsedFlags: map[string]string{"type": "A"},
		},
		{
			name:  "ttl in mixed scenario (after positional, not parsed)",
			args:  []string{"--type", "A", "domain.com", "--data", "1.2.3.4", "record-id", "--ttl", "300"},
			query: "ttl",
			want:  "300",
			parsedFlags: map[string]string{"type": "A"},
		},
		{
			name:  "empty args returns empty string",
			args:  []string{},
			query: "type",
			want:  "",
		},
		{
			name:  "flag without value returns next arg or empty",
			args:  []string{"--data"},
			query: "data",
			want:  "", // no value after --data in args
		},
		{
			name:  "unknown flag is ignored",
			args:  []string{"--nonexistent", "value"},
			query: "nonexistent",
			want:  "",
		},
		{
			name:  "flag with = syntax before positional",
			args:  []string{"--type=MX", "domain.com"},
			query: "type",
			want:  "MX",
			parsedFlags: map[string]string{"type": "MX"},
		},
		{
			name:  "flag with = syntax after positional",
			args:  []string{"domain.com", "--type=AAAA"},
			query: "type",
			want:  "AAAA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext(t, tt.args, tt.parsedFlags, dnsTestFlags()...)
			got := flagValueFromArgs(ctx, tt.query)
			if got != tt.want {
				t.Errorf("flagValueFromArgs(ctx, %q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFlagIsSetInArgs
// ---------------------------------------------------------------------------

func TestFlagIsSetInArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		parsedFlags map[string]string
		query       string
		want        bool
	}{
		{
			name:  "flag parsed by urfave/cli",
			args:  []string{"--type", "A", "domain.com"},
			query: "type",
			want:  true,
			parsedFlags: map[string]string{"type": "A"},
		},
		{
			name:  "flag after positional",
			args:  []string{"domain.com", "--data", "1.2.3.4"},
			query: "data",
			want:  true,
		},
		{
			name:  "flag not present",
			args:  []string{"domain.com"},
			query: "data",
			want:  false,
		},
		{
			name:  "flag with = syntax after positional",
			args:  []string{"domain.com", "--tag=issue"},
			query: "tag",
			want:  true,
		},
		{
			name:  "empty args",
			args:  []string{},
			query: "type",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext(t, tt.args, tt.parsedFlags, dnsTestFlags()...)
			got := flagIsSetInArgs(ctx, tt.query)
			if got != tt.want {
				t.Errorf("flagIsSetInArgs(ctx, %q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFlagIntFromArgs
// ---------------------------------------------------------------------------

func TestFlagIntFromArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		parsedFlags map[string]string
		query       string
		want        int
	}{
		{
			name:  "parsed int flag",
			args:  []string{"--ttl", "600", "domain.com"},
			query: "ttl",
			want:  600,
			parsedFlags: map[string]string{"ttl": "600"},
		},
		{
			name:  "int flag after positional",
			args:  []string{"domain.com", "--ttl", "3600"},
			query: "ttl",
			want:  3600,
		},
		{
			name:  "int flag with = syntax after positional",
			args:  []string{"domain.com", "--priority=10"},
			query: "priority",
			want:  10,
		},
		{
			name:  "zero for missing flag",
			args:  []string{"domain.com"},
			query: "ttl",
			want:  0,
		},
		{
			name:  "invalid int returns zero",
			args:  []string{"domain.com", "--ttl", "abc"},
			query: "ttl",
			want:  0,
		},
		{
			name:  "flag without value returns zero",
			args:  []string{"--ttl"},
			query: "ttl",
			want:  0,
		},
		{
			name:  "empty args returns zero",
			args:  []string{},
			query: "ttl",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext(t, tt.args, tt.parsedFlags, dnsTestFlags()...)
			got := flagIntFromArgs(ctx, tt.query)
			if got != tt.want {
				t.Errorf("flagIntFromArgs(ctx, %q) = %d, want %d", tt.query, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestDnsCreateRequestFromFlags
// ---------------------------------------------------------------------------

func TestDnsCreateRequestFromFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		parsedFlags map[string]string
		want        api.DNSRecordCreateRequest
	}{
		{
			name: "all required fields, flags before positional",
			args: []string{"--type", "A", "--name", "@", "--data", "1.2.3.4", "example.com"},
			parsedFlags: map[string]string{"type": "A", "name": "@", "data": "1.2.3.4"},
			want: api.DNSRecordCreateRequest{
				Type: "A",
				Name: "@",
				Data: "1.2.3.4",
			},
		},
		{
			name: "only required fields, flags after positional",
			args: []string{"example.com", "--type", "AAAA", "--name", "www", "--data", "::1"},
			want: api.DNSRecordCreateRequest{
				Type: "AAAA",
				Name: "www",
				Data: "::1",
			},
		},
		{
			name: "mixed flags, with ttl",
			args: []string{"--type", "CNAME", "example.com", "--name", "app", "--data", "target.example.com", "--ttl", "300"},
			parsedFlags: map[string]string{"type": "CNAME"},
			want: api.DNSRecordCreateRequest{
				Type: "CNAME",
				Name: "app",
				Data: "target.example.com",
				TTL:  intPtr(300),
			},
		},
		{
			name: "all optional fields",
			args: []string{"example.com",
				"--type", "SRV",
				"--name", "_sip._tcp",
				"--data", "10 5060 sip.example.com",
				"--ttl", "600",
				"--priority", "10",
				"--port", "5060",
				"--weight", "5",
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
			name: "CAA record with flags and tag",
			args: []string{"--type", "CAA", "--name", "@", "--data", "0 issue letsencrypt.org", "example.com", "--flags", "0", "--tag", "issue"},
			parsedFlags: map[string]string{"type": "CAA", "name": "@", "data": "0 issue letsencrypt.org"},
			want: api.DNSRecordCreateRequest{
				Type:  "CAA",
				Name:  "@",
				Data:  "0 issue letsencrypt.org",
				Flags: intPtr(0),
				Tag:   strPtr("issue"),
			},
		},
		{
			name: "type is uppercased",
			args: []string{"example.com", "--type", "a", "--name", "@", "--data", "1.2.3.4"},
			want: api.DNSRecordCreateRequest{
				Type: "A",
				Name: "@",
				Data: "1.2.3.4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext(t, tt.args, tt.parsedFlags, dnsTestFlags()...)
			got := dnsCreateRequestFromFlags(ctx)
			assertDNSRecordCreateEqual(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestDnsUpdateRequestFromFlags
// ---------------------------------------------------------------------------

func TestDnsUpdateRequestFromFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		parsedFlags map[string]string
		want        api.DNSRecordUpdateRequest
	}{
		{
			name: "single field update (data only)",
			args: []string{"example.com", "rec-123", "--data", "5.6.7.8"},
			want: api.DNSRecordUpdateRequest{
				Data: strPtr("5.6.7.8"),
			},
		},
		{
			name: "multiple field update (data + ttl)",
			args: []string{"example.com", "rec-123", "--data", "10.0.0.1", "--ttl", "1800"},
			want: api.DNSRecordUpdateRequest{
				Data: strPtr("10.0.0.1"),
				TTL:  intPtr(1800),
			},
		},
		{
			name: "all fields set, flags before positional",
			args: []string{"--type", "A", "--name", "@", "--data", "1.2.3.4", "--ttl", "300", "example.com", "rec-123", "--priority", "5", "--port", "8080", "--weight", "10", "--flags", "128", "--tag", "issue"},
			parsedFlags: map[string]string{"type": "A", "name": "@", "data": "1.2.3.4", "ttl": "300"},
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
			args: []string{"example.com", "rec-123"},
			want: api.DNSRecordUpdateRequest{},
		},
		{
			name: "type is uppercased in update",
			args: []string{"example.com", "rec-123", "--type", "aaaa"},
			want: api.DNSRecordUpdateRequest{
				Type: strPtr("AAAA"),
			},
		},
		{
			name: "update with = syntax",
			args: []string{"example.com", "rec-123", "--ttl=7200"},
			want: api.DNSRecordUpdateRequest{
				TTL: intPtr(7200),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testContext(t, tt.args, tt.parsedFlags, dnsTestFlags()...)
			got := dnsUpdateRequestFromFlags(ctx)
			assertDNSRecordUpdateEqual(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Test helpers to compare DNS request structs
// ---------------------------------------------------------------------------

func intPtr(v int) *int    { return &v }
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
