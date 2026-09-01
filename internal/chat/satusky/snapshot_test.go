package satusky

import (
	"context"
	"strings"
	"testing"
	"time"
)

// canned1ctl is a fake Exec keyed by full arg lists; unknown commands
// return a non-zero result. The "-o json" suffix marks JSON attempts so
// tests can simulate commands that ignore the flag (text output).
func canned1ctl(outputs map[string]Result) func(context.Context, ...string) (Result, error) {
	return func(ctx context.Context, args ...string) (Result, error) {
		key := strings.Join(args, " ")
		if res, ok := outputs[key]; ok {
			return res, nil
		}
		return Result{ExitCode: 2, Stderr: "unknown command: " + key}, nil
	}
}

func authStatusOK() Result {
	return Result{ExitCode: 0, Stdout: `Authenticated with Satusky
User Email: dev@acme.com
Organization: Acme Inc
Organization ID: 111
Namespace: acme
Token expires: in 30 days`}
}

func authedSnapshotRunner(outputs map[string]Result) *Runner {
	r := NewRunner(nil)
	r.Exec = canned1ctl(outputs)
	return r
}

func TestSnapshotAuthenticated(t *testing.T) {
	outputs := map[string]Result{
		"auth status":             authStatusOK(),
		"profile list -o json":    {ExitCode: 0, Stdout: `[{"name":"dev","api_url":"https://dev","active":false},{"name":"prod","api_url":"https://prod","active":true}]`},
		"app list -o json":        {ExitCode: 0, Stdout: `[{"app_label":"my-app","status":"running"},{"app_label":"dashboard","status":"building"}]`},
		"postgres list -o json":   {ExitCode: 0, Stdout: `[{"cluster_name":"staging-db"},{"cluster_name":"analytics"}]`},
		"valkey list -o json":     {ExitCode: 0, Stdout: `[{"cluster_name":"cache-1"}]`},
		"nats list -o json":       {ExitCode: 0, Stdout: `[{"app_label":"events-bus"}]`},
		"domains list -o json":    {ExitCode: 0, Stdout: `[{"domain_name":"api.acme.com"},{"domain_name":"www.acme.com"}]`},
		"credits balance -o json": {ExitCode: 0, Stdout: `{"organization_id":"111","balance":12.4,"currency":"USD"}`},
	}
	snap, err := authedSnapshotRunner(outputs).Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !snap.Authenticated {
		t.Error("Authenticated = false, want true")
	}
	if snap.Profile != "prod" {
		t.Errorf("Profile = %q, want prod", snap.Profile)
	}
	if snap.Org != "Acme Inc" || snap.Namespace != "acme" {
		t.Errorf("Org/Namespace = %q/%q", snap.Org, snap.Namespace)
	}
	if strings.Join(snap.Apps, ",") != "my-app,dashboard" {
		t.Errorf("Apps = %v", snap.Apps)
	}
	if strings.Join(snap.Postgres, ",") != "staging-db,analytics" {
		t.Errorf("Postgres = %v", snap.Postgres)
	}
	if strings.Join(snap.Valkey, ",") != "cache-1" {
		t.Errorf("Valkey = %v", snap.Valkey)
	}
	if strings.Join(snap.NATS, ",") != "events-bus" {
		t.Errorf("NATS = %v", snap.NATS)
	}
	if strings.Join(snap.Domains, ",") != "api.acme.com,www.acme.com" {
		t.Errorf("Domains = %v", snap.Domains)
	}
	if snap.Credits != "$12.40 USD" {
		t.Errorf("Credits = %q, want $12.40 USD", snap.Credits)
	}
	if len(snap.Errors) != 0 {
		t.Errorf("Errors = %v, want none", snap.Errors)
	}
	if snap.TakenAt.IsZero() {
		t.Error("TakenAt not set")
	}
}

func TestSnapshotUnauthenticatedSkipsRest(t *testing.T) {
	var calls []string
	r := NewRunner(nil)
	r.Exec = func(ctx context.Context, args ...string) (Result, error) {
		calls = append(calls, strings.Join(args, " "))
		return Result{ExitCode: 1, Stderr: "not authenticated. Please run '1ctl auth login' to authenticate"}, nil
	}
	snap, err := r.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Authenticated {
		t.Error("Authenticated = true, want false")
	}
	if len(calls) != 1 || calls[0] != "auth status" {
		t.Errorf("calls = %v, want only auth status", calls)
	}
	if len(snap.Errors) != 1 || snap.Errors[0] != "auth status" {
		t.Errorf("Errors = %v, want [auth status]", snap.Errors)
	}
	// Digest must reflect the unauthenticated state without noise.
	d := snap.Digest()
	if !strings.Contains(d, "not authenticated") || !strings.Contains(d, "auth login") {
		t.Errorf("Digest = %q", d)
	}
	if strings.Contains(d, "state checks failed") {
		t.Errorf("unauthenticated digest leaks failed-command noise: %q", d)
	}
}

func TestSnapshotPartialFailuresRecorded(t *testing.T) {
	outputs := map[string]Result{
		"auth status":           authStatusOK(),
		"profile list -o json":  {ExitCode: 0, Stdout: `[{"name":"prod","api_url":"https://prod","active":true}]`},
		"app list -o json":      {ExitCode: 0, Stdout: `[{"app_label":"my-app"}]`},
		"postgres list -o json": {ExitCode: 1, Stderr: "failed to list Postgres clusters: boom"},
		"valkey list -o json":   {ExitCode: 0, Stdout: `[{"cluster_name":"cache-1"}]`},
		// nats list missing → canned unknown (exit 2)
		"domains list -o json":    {ExitCode: 0, Stdout: `[{"domain_name":"api.acme.com"}]`},
		"credits balance -o json": {ExitCode: 0, Stdout: `{"balance":1.5,"currency":"USD"}`},
	}
	snap, err := authedSnapshotRunner(outputs).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Errors) != 2 {
		t.Fatalf("Errors = %v, want 2 failures (postgres, nats)", snap.Errors)
	}
	if snap.Errors[0] != "postgres list" || snap.Errors[1] != "nats list" {
		t.Errorf("Errors = %v", snap.Errors)
	}
	d := snap.Digest()
	if !strings.Contains(d, "2 state checks failed: postgres list, nats list") {
		t.Errorf("Digest missing failures line: %q", d)
	}
}

func TestSnapshotTextFallback(t *testing.T) {
	// Commands that ignore -o json and print tables/text: profile list
	// (table), credits balance (status line), lists (table).
	outputs := map[string]Result{
		"auth status": authStatusOK(),
		// -o json attempt returns text (not valid JSON) → RunJSON falls
		// back to the plain invocation, which returns the same text.
		"profile list": {ExitCode: 0, Stdout: `Profiles
───────────
* prod
  API URL: https://prod
  Auth: dev@acme.com
  Org: Acme Inc`},
		"app list": {ExitCode: 0, Stdout: `APP      STATUS
my-app   running
dashboard building`},
		"postgres list": {ExitCode: 0, Stdout: "No Postgres clusters found"},
		"valkey list":   {ExitCode: 0, Stdout: "No Valkey services found"},
		"nats list":     {ExitCode: 0, Stdout: "No NATS services found"},
		"domains list": {ExitCode: 0, Stdout: `DOMAIN        APP
api.acme.com  my-app`},
		"credits balance": {ExitCode: 0, Stdout: "Balance: $12.40 USD"},
	}
	snap, err := authedSnapshotRunner(outputs).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Authenticated {
		t.Error("not authenticated")
	}
	if snap.Profile != "prod" {
		t.Errorf("Profile = %q, want prod (text fallback)", snap.Profile)
	}
	if strings.Join(snap.Apps, ",") != "my-app,dashboard" {
		t.Errorf("Apps = %v", snap.Apps)
	}
	if len(snap.Postgres) != 0 || len(snap.Valkey) != 0 || len(snap.NATS) != 0 {
		t.Errorf("empty lists not empty: %v %v %v", snap.Postgres, snap.Valkey, snap.NATS)
	}
	if strings.Join(snap.Domains, ",") != "api.acme.com" {
		t.Errorf("Domains = %v", snap.Domains)
	}
	if snap.Credits != "$12.40 USD" {
		t.Errorf("Credits = %q", snap.Credits)
	}
	if len(snap.Errors) != 0 {
		t.Errorf("Errors = %v", snap.Errors)
	}
}

func TestSnapshotDigestAuthenticated(t *testing.T) {
	snap := &Snapshot{
		Authenticated: true,
		Profile:       "prod",
		Org:           "Acme Inc",
		Namespace:     "acme",
		Apps:          []string{"my-app"},
		Postgres:      []string{"staging-db"},
		Domains:       []string{"api.acme.com"},
		Credits:       "$12.40 USD",
		TakenAt:       time.Now(),
	}
	d := snap.Digest()
	for _, want := range []string{
		"## Current SatuSky state",
		"authenticated: yes",
		"profile: prod",
		"org: Acme Inc",
		"namespace: acme",
		"apps: my-app",
		"postgres: staging-db",
		"valkey: none",
		"nats: none",
		"domains: api.acme.com",
		"credits: $12.40 USD",
	} {
		if !strings.Contains(d, want) {
			t.Errorf("Digest missing %q:\n%s", want, d)
		}
	}
	if len(d) > 1500 {
		t.Errorf("Digest too long: %d chars", len(d))
	}
}

func TestSnapshotDigestCapsLongLists(t *testing.T) {
	var apps []string
	for i := 0; i < 25; i++ {
		apps = append(apps, "app-"+string(rune('a'+i)))
	}
	snap := &Snapshot{Authenticated: true, Apps: apps}
	d := snap.Digest()
	if !strings.Contains(d, "…5 more") {
		t.Errorf("Digest missing cap notice: %q", d)
	}
	if strings.Count(d, "app-") > maxListEntries {
		t.Errorf("Digest listed more than %d entries", maxListEntries)
	}
}

func TestSnapshotUnauthenticatedDigest(t *testing.T) {
	snap := &Snapshot{Authenticated: false, TakenAt: time.Now()}
	d := snap.Digest()
	if !strings.Contains(d, "not authenticated — SatuSky actions unavailable; suggest `1ctl auth login`") {
		t.Errorf("Digest = %q", d)
	}
	if strings.Contains(d, "authenticated: yes") {
		t.Errorf("Digest claims authentication: %q", d)
	}
}
