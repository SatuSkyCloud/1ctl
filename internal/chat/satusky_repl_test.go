package chat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"1ctl/internal/chat/satusky"

	chattools "1ctl/internal/chat/tools"

	"github.com/fatih/color"
	openai "github.com/sashabaranov/go-openai"
)

// fakeSatuskyRunner builds a runner with a canned Exec: read-only
// commands return canned JSON, mutating commands record execution.
func fakeSatuskyRunner(t *testing.T, confirm func(string) bool, executed *[][]string) *satusky.Runner {
	t.Helper()
	r := satusky.NewRunner(confirm)
	r.Exec = func(ctx context.Context, args ...string) (satusky.Result, error) {
		joined := strings.Join(args, " ")
		switch joined {
		case "auth status":
			return satusky.Result{ExitCode: 0, Stdout: "Authenticated with Satusky\nOrganization: Acme Inc\nNamespace: acme"}, nil
		case "profile list -o json", "profile list":
			return satusky.Result{ExitCode: 0, Stdout: `[{"name":"prod","active":true}]`}, nil
		case "app list -o json", "app list":
			return satusky.Result{ExitCode: 0, Stdout: `[{"app_label":"my-app"},{"app_label":"dashboard"}]`}, nil
		case "postgres list -o json", "postgres list":
			return satusky.Result{ExitCode: 0, Stdout: `[{"cluster_name":"staging-db"}]`}, nil
		case "valkey list -o json", "valkey list":
			return satusky.Result{ExitCode: 0, Stdout: `[]`}, nil
		case "nats list -o json", "nats list":
			return satusky.Result{ExitCode: 0, Stdout: `[]`}, nil
		case "domains list -o json", "domains list":
			return satusky.Result{ExitCode: 0, Stdout: `[{"domain_name":"api.acme.com"}]`}, nil
		case "credits balance -o json", "credits balance":
			return satusky.Result{ExitCode: 0, Stdout: `{"balance":12.4,"currency":"USD"}`}, nil
		case "doctor":
			return satusky.Result{ExitCode: 0, Stdout: "All systems healthy"}, nil
		}
		if executed != nil {
			*executed = append(*executed, args)
		}
		return satusky.Result{ExitCode: 0, Stdout: "ran: " + joined}, nil
	}
	return r
}

// stateWithSatusky builds a replState with the executor + satusky runner
// wired exactly as Run() does (shared confirm callback, custom tools).
func stateWithSatusky(t *testing.T, srvURL string, confirm func(string) bool, executed *[][]string) *replState {
	t.Helper()
	state := replTestState(t, srvURL)
	state.exec = chattools.NewExecutor(t.TempDir(), confirm)
	state.runner = fakeSatuskyRunner(t, confirm, executed)
	registerSatuskyTools(context.Background(), state)
	state.toolsEnabled = true
	return state
}

func TestRunSatuskyToolReadOnlyNoConfirm(t *testing.T) {
	color.NoColor = true
	confirmed := 0
	state := stateWithSatusky(t, "http://127.0.0.1:1", func(string) bool { confirmed++; return true }, nil)

	out := runSatuskyTool(context.Background(), state, []byte(`{"args":["app","list"]}`))
	if confirmed != 0 {
		t.Errorf("Confirm called %d times for read-only command", confirmed)
	}
	if !strings.Contains(out, "my-app") {
		t.Errorf("result = %q, want app list output", out)
	}
}

func TestRunSatuskyToolMutatingConfirmed(t *testing.T) {
	color.NoColor = true
	var executed [][]string
	state := stateWithSatusky(t, "http://127.0.0.1:1", func(string) bool { return true }, &executed)

	out := runSatuskyTool(context.Background(), state, []byte(`{"args":["postgres","create","--name","staging-db","--plan","small"]}`))
	if len(executed) != 1 {
		t.Fatalf("executed = %v, want postgres create to run after confirmation", executed)
	}
	if !strings.Contains(out, "ran: postgres create") {
		t.Errorf("result = %q", out)
	}
}

func TestRunSatuskyToolMutatingDeclined(t *testing.T) {
	color.NoColor = true
	var executed [][]string
	state := stateWithSatusky(t, "http://127.0.0.1:1", func(string) bool { return false }, &executed)

	out := runSatuskyTool(context.Background(), state, []byte(`{"args":["app","delete","my-app"]}`))
	if len(executed) != 0 {
		t.Errorf("executed despite decline: %v", executed)
	}
	if !strings.Contains(out, "cancelled by user") {
		t.Errorf("result = %q, want cancelled by user", out)
	}
	if !strings.Contains(out, "exit code -1") {
		t.Errorf("result = %q, want exit code -1", out)
	}
}

func TestRunSatuskyToolLaunchBlocked(t *testing.T) {
	color.NoColor = true
	var executed [][]string
	state := stateWithSatusky(t, "http://127.0.0.1:1", func(string) bool { return true }, &executed)

	out := runSatuskyTool(context.Background(), state, []byte(`{"args":["launch"]}`))
	if len(executed) != 0 {
		t.Errorf("launch executed: %v", executed)
	}
	if !strings.Contains(out, "refused") || !strings.Contains(out, "interactive wizard") {
		t.Errorf("result = %q, want refusal", out)
	}
}

func TestRunSatuskyToolCommandString(t *testing.T) {
	color.NoColor = true
	var executed [][]string
	state := stateWithSatusky(t, "http://127.0.0.1:1", func(string) bool { return true }, &executed)

	out := runSatuskyTool(context.Background(), state, []byte(`{"command":"postgres create --name db"}`))
	if len(executed) != 1 || executed[0][0] != "postgres" || executed[0][2] != "--name" {
		t.Errorf("executed = %v, want split postgres create --name db", executed)
	}
	if !strings.Contains(out, "ran: postgres create") {
		t.Errorf("result = %q", out)
	}
}

func TestRunSatuskyToolInvalidArgs(t *testing.T) {
	color.NoColor = true
	state := stateWithSatusky(t, "http://127.0.0.1:1", func(string) bool { return true }, nil)
	out := runSatuskyTool(context.Background(), state, []byte(`{"bogus":1}`))
	if !strings.Contains(out, "error") {
		t.Errorf("result = %q, want error", out)
	}
	out = runSatuskyTool(context.Background(), state, []byte(`not json`))
	if !strings.Contains(out, "error") {
		t.Errorf("result = %q, want error", out)
	}
}

func TestSatuskyStatusToolDigest(t *testing.T) {
	color.NoColor = true
	state := stateWithSatusky(t, "http://127.0.0.1:1", func(string) bool { return false }, nil)

	out := state.exec.Custom["satusky_status"](nil)
	if !strings.Contains(out, "## Current SatuSky state") {
		t.Errorf("digest = %q", out)
	}
	for _, want := range []string{"authenticated: yes", "profile: prod", "org: Acme Inc", "namespace: acme", "apps: my-app, dashboard", "postgres: staging-db", "domains: api.acme.com", "credits: $12.40 USD"} {
		if !strings.Contains(out, want) {
			t.Errorf("digest missing %q: %q", want, out)
		}
	}
	// The tool must have refreshed the cached snapshot.
	if state.snapshot == nil || state.snapshotAt.IsZero() {
		t.Error("satusky_status did not refresh the snapshot cache")
	}
}

func TestRefreshSnapshotCaches(t *testing.T) {
	color.NoColor = true
	state := stateWithSatusky(t, "http://127.0.0.1:1", nil, nil)
	snap, err := refreshSnapshot(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	if state.snapshot != snap || state.snapshotAt.IsZero() {
		t.Error("refreshSnapshot did not cache")
	}
	if !snap.Authenticated || snap.Profile != "prod" {
		t.Errorf("snapshot = %+v", snap)
	}
}

func TestRefreshSnapshotNoRunnerErrors(t *testing.T) {
	state := replTestState(t, "http://127.0.0.1:1")
	if _, err := refreshSnapshot(context.Background(), state); err == nil {
		t.Error("refreshSnapshot without runner should error")
	}
}

func TestBuildSystemAppendsSnapshotDigest(t *testing.T) {
	state := replTestState(t, "http://127.0.0.1:1")
	state.snapshot = &satusky.Snapshot{
		Authenticated: true,
		Profile:       "prod",
		Apps:          []string{"my-app"},
		Credits:       "$1.00 USD",
		TakenAt:       time.Now(),
	}
	sys := buildSystem(NewSession("skill body"), state)
	if !strings.Contains(sys, "skill body") {
		t.Errorf("skill missing: %q", sys)
	}
	if !strings.Contains(sys, "## Current SatuSky state") || !strings.Contains(sys, "apps: my-app") {
		t.Errorf("snapshot digest missing from system prompt: %q", sys)
	}
}

func TestBuildSystemAskFirstPlusDigest(t *testing.T) {
	state := replTestState(t, "http://127.0.0.1:1")
	state.askFirst = true
	state.snapshot = &satusky.Snapshot{Authenticated: false, TakenAt: time.Now()}
	sys := buildSystem(NewSession("skill"), state)
	if !strings.Contains(sys, "MUST ask up to 3 clarifying questions") {
		t.Errorf("askFirst instruction missing: %q", sys)
	}
	if !strings.Contains(sys, "not authenticated") {
		t.Errorf("digest missing: %q", sys)
	}
}

func TestRunTurnInjectsSnapshotIntoRequest(t *testing.T) {
	color.NoColor = true
	srv, bodies := toolLoopServer(t, []string{streamChunk("sure"), "[DONE]"}, nil)
	state := replTestState(t, srv.URL)
	state.toolsEnabled = true
	state.snapshot = &satusky.Snapshot{
		Authenticated: true,
		Profile:       "prod",
		Apps:          []string{"my-app"},
		Postgres:      []string{"staging-db"},
		TakenAt:       time.Now(),
	}
	session := NewSession("sys")

	var out strings.Builder
	if err := runTurn(context.Background(), state, session, "what do I have deployed?", &out, true); err != nil {
		t.Fatalf("runTurn: %v", err)
	}
	var req openai.ChatCompletionRequest
	if err := json.Unmarshal((*bodies)[0], &req); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if len(req.Messages) == 0 || req.Messages[0].Role != openai.ChatMessageRoleSystem {
		t.Fatalf("first message = %+v, want system", req.Messages)
	}
	if !strings.Contains(req.Messages[0].Content, "## Current SatuSky state") {
		t.Errorf("system prompt missing snapshot digest: %q", req.Messages[0].Content)
	}
	if !strings.Contains(req.Messages[0].Content, "postgres: staging-db") {
		t.Errorf("system prompt missing resource names: %q", req.Messages[0].Content)
	}
	// The satusky tools must be advertised in the request.
	if len(req.Tools) < 6 {
		t.Errorf("request Tools = %d, want workspace + satusky tools", len(req.Tools))
	}
	var names []string
	for _, tl := range req.Tools {
		names = append(names, tl.Function.Name)
	}
	if !containsStr(names, "satusky_status") || !containsStr(names, "satusky_run") {
		t.Errorf("satusky tools missing from request: %v", names)
	}
}

func TestDispatchStatusPrintsDigest(t *testing.T) {
	color.NoColor = true
	state := stateWithSatusky(t, "http://127.0.0.1:1", nil, nil)
	session := NewSession("sys")
	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader(""), Stdout: &out}

	exit, err := dispatchSlash(context.Background(), NewStore(t.TempDir()), state, session, CmdStatus, "", opts)
	if err != nil {
		t.Fatalf("/status: %v", err)
	}
	if exit {
		t.Fatal("/status must not exit the REPL")
	}
	got := out.String()
	if !strings.Contains(got, "## Current SatuSky state") || !strings.Contains(got, "apps: my-app, dashboard") {
		t.Errorf("digest missing from /status output: %q", got)
	}
	if !strings.Contains(got, "credits: $12.40 USD") {
		t.Errorf("credits missing: %q", got)
	}
}

func TestDispatchStatusUnauthenticatedHint(t *testing.T) {
	color.NoColor = true
	r := satusky.NewRunner(nil)
	r.Exec = func(ctx context.Context, args ...string) (satusky.Result, error) {
		return satusky.Result{ExitCode: 1, Stderr: "not authenticated"}, nil
	}
	state := replTestState(t, "http://127.0.0.1:1")
	state.runner = r
	session := NewSession("sys")
	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader(""), Stdout: &out}

	if _, err := dispatchSlash(context.Background(), NewStore(t.TempDir()), state, session, CmdStatus, "", opts); err != nil {
		t.Fatalf("/status: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "1ctl auth login") || !strings.Contains(got, "not authenticated") {
		t.Errorf("/status unauthenticated output = %q", got)
	}
}

// Demo flow: provision a postgres database (mutating, confirmed).
func TestDemoFlowProvisionPostgres(t *testing.T) {
	color.NoColor = true
	var executed [][]string
	state := stateWithSatusky(t, "http://127.0.0.1:1", func(string) bool { return true }, &executed)

	// The agent inspects state first (read-only, no confirm), then
	// proposes the creation (confirmed).
	status := state.exec.Custom["satusky_status"](nil)
	if !strings.Contains(status, "postgres: staging-db") {
		t.Errorf("status digest = %q", status)
	}
	out := runSatuskyTool(context.Background(), state, []byte(`{"args":["postgres","create","--name","staging-db","--plan","small"]}`))
	if len(executed) != 1 {
		t.Fatalf("postgres create not executed: %v", executed)
	}
	if !strings.Contains(out, "ran: postgres create") {
		t.Errorf("result = %q", out)
	}
}

// Demo flow: diagnose with doctor (read-only, runs without confirmation).
func TestDemoFlowDoctorDiagnosis(t *testing.T) {
	color.NoColor = true
	confirmed := 0
	state := stateWithSatusky(t, "http://127.0.0.1:1", func(string) bool { confirmed++; return true }, nil)

	out := runSatuskyTool(context.Background(), state, []byte(`{"args":["doctor"]}`))
	if confirmed != 0 {
		t.Errorf("doctor required confirmation (%d)", confirmed)
	}
	if !strings.Contains(out, "All systems healthy") {
		t.Errorf("doctor output = %q", out)
	}
}

func containsStr(values []string, needle string) bool {
	for _, v := range values {
		if v == needle {
			return true
		}
	}
	return false
}
