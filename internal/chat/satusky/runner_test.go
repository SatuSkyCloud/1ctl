package satusky

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeRunner builds a runner whose Exec records calls and returns the
// given results keyed by joined args.
func fakeRunner(confirm func(string) bool, results map[string]Result, calls *[][]string) *Runner {
	r := NewRunner(confirm)
	r.Exec = func(ctx context.Context, args ...string) (Result, error) {
		if calls != nil {
			*calls = append(*calls, args)
		}
		if res, ok := results[strings.Join(args, " ")]; ok {
			return res, nil
		}
		return Result{ExitCode: 0}, nil
	}
	return r
}

func TestRunExecCalledWithExactArgs(t *testing.T) {
	var calls [][]string
	r := fakeRunner(nil, nil, &calls)
	res, err := r.Run(context.Background(), "app", "list")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(calls) != 1 || calls[0][0] != "app" || calls[0][1] != "list" {
		t.Fatalf("calls = %v, want [app list]", calls)
	}
	if res.Command != "app list" {
		t.Errorf("res.Command = %q, want %q", res.Command, "app list")
	}
}

func TestRunExecErrorPassthrough(t *testing.T) {
	r := NewRunner(nil)
	r.Exec = func(ctx context.Context, args ...string) (Result, error) {
		return Result{}, errors.New("boom")
	}
	if _, err := r.Run(context.Background(), "doctor"); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Run error = %v, want boom", err)
	}
}

func TestRunOutputCapping(t *testing.T) {
	long := strings.Repeat("x", 20_000)
	r := NewRunner(nil)
	r.Exec = func(ctx context.Context, args ...string) (Result, error) {
		return Result{Stdout: long, Stderr: long, ExitCode: 1}, nil
	}
	res, err := r.Run(context.Background(), "app", "list")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Stdout) > maxStreamChars+len("…(truncated)")+1 {
		t.Errorf("stdout not capped: %d chars", len(res.Stdout))
	}
	if !strings.HasSuffix(res.Stdout, "…(truncated)") {
		t.Errorf("stdout missing truncation notice: %q", res.Stdout[len(res.Stdout)-30:])
	}
	if !strings.HasSuffix(res.Stderr, "…(truncated)") {
		t.Errorf("stderr missing truncation notice")
	}
}

func TestRunJSONAppendsFlagAndFallsBackToText(t *testing.T) {
	var calls [][]string
	// First call (-o json) returns non-JSON text with exit 0; the fallback
	// re-run without the flag returns the real output.
	r := NewRunner(nil)
	r.Exec = func(ctx context.Context, args ...string) (Result, error) {
		calls = append(calls, args)
		joined := strings.Join(args, " ")
		if strings.HasSuffix(joined, "-o json") {
			return Result{Stdout: "Some human text\n", ExitCode: 0}, nil
		}
		return Result{Stdout: "clean text\n", ExitCode: 0}, nil
	}
	res, err := r.RunJSON(context.Background(), "app", "list")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %v, want 2 (json attempt + text fallback)", calls)
	}
	if res.Stdout != "clean text" {
		t.Errorf("Stdout = %q, want the text-fallback output", res.Stdout)
	}
}

func TestRunJSONKeepsValidJSON(t *testing.T) {
	var calls [][]string
	r := NewRunner(nil)
	r.Exec = func(ctx context.Context, args ...string) (Result, error) {
		calls = append(calls, args)
		return Result{Stdout: `[{"app_label":"my-app"}]`, ExitCode: 0}, nil
	}
	res, err := r.RunJSON(context.Background(), "app", "list")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1 (no fallback for valid JSON)", len(calls))
	}
	if !strings.Contains(res.Stdout, "my-app") {
		t.Errorf("Stdout = %q", res.Stdout)
	}
}

func TestMutating(t *testing.T) {
	cases := []struct {
		args  []string
		want  bool
		label string
	}{
		{[]string{"app", "list"}, false, "app list is read-only"},
		{[]string{"app", "get", "my-app"}, false, "app get is read-only"},
		{[]string{"app", "status"}, false, "app status is read-only"},
		{[]string{"app", "logs", "my-app"}, false, "app logs is read-only"},
		{[]string{"app", "stream"}, false, "app stream is read-only"},
		{[]string{"app", "delete", "my-app"}, true, "app delete mutates"},
		{[]string{"app", "restart", "my-app"}, true, "app restart mutates"},
		{[]string{"app", "rollback", "my-app", "--version", "2"}, true, "app rollback mutates"},
		{[]string{"app", "scale", "my-app", "3"}, true, "app scale mutates"},
		{[]string{"postgres", "list"}, false, "postgres list is read-only"},
		{[]string{"postgres", "get", "abc"}, false, "postgres get is read-only"},
		{[]string{"postgres", "status", "abc"}, false, "postgres status is read-only"},
		{[]string{"postgres", "credentials"}, false, "postgres credentials is read-only"},
		{[]string{"postgres", "create", "--name", "db"}, true, "postgres create mutates"},
		{[]string{"postgres", "delete", "abc"}, true, "postgres delete mutates"},
		{[]string{"postgres", "firewall", "list"}, false, "postgres firewall list is read-only"},
		{[]string{"postgres", "firewall", "add"}, true, "postgres firewall add mutates"},
		{[]string{"postgres", "users", "list"}, false, "postgres users list is read-only"},
		{[]string{"valkey", "list"}, false, "valkey list is read-only"},
		{[]string{"valkey", "rotate-password"}, true, "valkey rotate-password mutates"},
		{[]string{"valkey", "users", "create"}, true, "valkey users create mutates"},
		{[]string{"nats", "list"}, false, "nats list is read-only"},
		{[]string{"nats", "create", "--jetstream"}, true, "nats create mutates"},
		{[]string{"nats", "delete", "x"}, true, "nats delete mutates"},
		{[]string{"domains", "list"}, false, "domains list is read-only"},
		{[]string{"domains", "check", "api.acme.com"}, false, "domains check is read-only"},
		{[]string{"domains", "add", "--domain", "x.com", "--app", "my-app"}, true, "domains add mutates"},
		{[]string{"domains", "delete", "x.com"}, true, "domains delete mutates"},
		{[]string{"domains", "dns", "list", "--domain", "x.com"}, false, "domains dns list is read-only"},
		{[]string{"domains", "dns", "create"}, true, "domains dns create mutates"},
		{[]string{"domains", "managed", "list"}, false, "domains managed list is read-only"},
		{[]string{"config", "list"}, false, "config list is read-only"},
		{[]string{"config", "create", "--env", "A=B"}, true, "config create mutates"},
		{[]string{"env", "list"}, false, "env list (alias) is read-only"},
		{[]string{"env", "unset", "--key", "A"}, true, "env unset mutates"},
		{[]string{"secret", "get", "abc"}, false, "secret get is read-only"},
		{[]string{"secret", "create"}, true, "secret create mutates"},
		{[]string{"secret", "unset"}, true, "secret unset mutates"},
		{[]string{"volumes", "list"}, false, "volumes list is read-only"},
		{[]string{"volumes", "delete", "v1"}, true, "volumes delete mutates"},
		{[]string{"machine", "list"}, false, "machine list is read-only"},
		{[]string{"machine", "usage", "cost"}, false, "machine usage cost is read-only"},
		{[]string{"machine", "labels", "list"}, false, "machine labels list is read-only"},
		{[]string{"machine", "labels", "set"}, true, "machine labels set mutates"},
		{[]string{"machine", "create"}, true, "machine create mutates"},
		{[]string{"cluster", "zones", "list"}, false, "cluster zones list is read-only"},
		{[]string{"marketplace", "get", "x"}, false, "marketplace get is read-only"},
		{[]string{"marketplace", "deploy", "x"}, true, "marketplace deploy mutates"},
		{[]string{"package", "list"}, false, "package list is read-only"},
		{[]string{"package", "create"}, true, "package create mutates"},
		{[]string{"auth", "status"}, false, "auth status is read-only"},
		{[]string{"auth", "login"}, true, "auth login mutates"},
		{[]string{"auth", "logout"}, true, "auth logout mutates"},
		{[]string{"profile", "list"}, false, "profile list is read-only"},
		{[]string{"profile", "current"}, false, "profile current is read-only"},
		{[]string{"profile", "create", "prod"}, true, "profile create mutates"},
		{[]string{"profile", "use", "prod"}, true, "profile use mutates"},
		{[]string{"profile", "delete", "prod"}, true, "profile delete mutates"},
		{[]string{"org", "switch"}, true, "org switch mutates"},
		{[]string{"credits", "balance"}, false, "credits balance is read-only"},
		{[]string{"credits", "transactions"}, false, "credits transactions is read-only"},
		{[]string{"billing", "balance"}, false, "billing balance (alias) is read-only"},
		{[]string{"pricing", "calculate"}, false, "pricing calculate is read-only"},
		{[]string{"doctor"}, false, "doctor is read-only"},
		{[]string{"doctor", "my-app"}, false, "doctor with app arg is read-only"},
		{[]string{"audit", "list"}, false, "audit list is read-only"},
		{[]string{"notifications", "read"}, true, "notifications read mutates"},
		{[]string{"service", "delete"}, true, "service delete mutates"},
		{[]string{"ingress", "delete"}, true, "ingress delete mutates"},
		{[]string{"deploy", "--port", "3000"}, true, "deploy mutates"},
		{[]string{"init"}, true, "init mutates"},
		{[]string{"launch"}, true, "launch is mutating-class (refused in RunConfirmed)"},
		{[]string{"app", "frobnicate"}, true, "unknown subcommand defaults to mutating"},
		{[]string{"mystery", "sub"}, true, "unknown command defaults to mutating"},
		{[]string{"--profile", "prod", "app", "list"}, false, "global flag before command still classifies"},
		{[]string{}, false, "bare invocation is harmless"},
	}
	for _, c := range cases {
		if got := Mutating(c.args); got != c.want {
			t.Errorf("Mutating(%v) = %v, want %v (%s)", c.args, got, c.want, c.label)
		}
	}
}

func TestBlastWarning(t *testing.T) {
	cases := []struct {
		args  []string
		want  bool
		label string
	}{
		{[]string{"postgres", "delete", "abc"}, true, "delete warns"},
		{[]string{"app", "delete", "my-app"}, true, "app delete warns"},
		{[]string{"domains", "delete", "x.com"}, true, "domains delete warns"},
		{[]string{"env", "unset", "--key", "A"}, true, "unset warns"},
		{[]string{"secret", "unset"}, true, "secret unset warns"},
		{[]string{"app", "scale", "my-app", "3"}, true, "scale warns"},
		{[]string{"postgres", "create", "--name", "db", "--memory", "2Gi"}, true, "create with size flag warns"},
		{[]string{"postgres", "create", "--name", "db"}, true, "create warns (new resource)"},
		{[]string{"domains", "add", "--domain", "x.com"}, true, "add warns"},
		{[]string{"valkey", "rotate-password"}, true, "rotate warns"},
		{[]string{"app", "list"}, false, "read-only has no warning"},
		{[]string{"postgres", "list"}, false, "list has no warning"},
		{[]string{"app", "get", "my-app"}, false, "get has no warning"},
	}
	for _, c := range cases {
		msg, ok := BlastWarning(c.args)
		if ok != c.want {
			t.Errorf("BlastWarning(%v) ok = %v, want %v", c.args, ok, c.want)
		}
		if c.want && msg == "" {
			t.Errorf("BlastWarning(%v) returned empty message", c.args)
		}
	}
	msg, ok := BlastWarning([]string{"app", "delete", "my-app"})
	if !ok || !strings.Contains(msg, "irreversible") {
		t.Errorf("delete warning = %q, want irreversible mention", msg)
	}
}

func TestRunConfirmedReadOnlyNoConfirm(t *testing.T) {
	var calls [][]string
	confirmed := 0
	r := fakeRunner(func(string) bool { confirmed++; return true }, map[string]Result{
		"app list": {Stdout: "my-app", ExitCode: 0},
	}, &calls)
	res, err := r.RunConfirmed(context.Background(), "app", "list")
	if err != nil {
		t.Fatal(err)
	}
	if confirmed != 0 {
		t.Errorf("Confirm called %d times for a read-only command", confirmed)
	}
	if res.Stdout != "my-app" {
		t.Errorf("Stdout = %q, want my-app", res.Stdout)
	}
	if len(calls) != 1 {
		t.Errorf("calls = %v, want 1", calls)
	}
}

func TestRunConfirmedMutatingConfirmed(t *testing.T) {
	var calls [][]string
	var prompts []string
	r := fakeRunner(func(p string) bool { prompts = append(prompts, p); return true }, map[string]Result{
		"postgres create --name staging-db": {Stdout: "created", ExitCode: 0},
	}, &calls)
	res, err := r.RunConfirmed(context.Background(), "postgres", "create", "--name", "staging-db")
	if err != nil {
		t.Fatal(err)
	}
	if len(prompts) != 1 {
		t.Fatalf("prompts = %v, want 1", prompts)
	}
	if !strings.Contains(prompts[0], "🔧 1ctl postgres create --name staging-db") {
		t.Errorf("preview line missing from prompt: %q", prompts[0])
	}
	if !strings.Contains(prompts[0], "irreversible") && !strings.Contains(prompts[0], "costs money") {
		t.Errorf("blast warning missing from prompt: %q", prompts[0])
	}
	if len(calls) != 1 || calls[0][0] != "postgres" {
		t.Errorf("calls = %v, want postgres create executed", calls)
	}
	if res.Stdout != "created" {
		t.Errorf("Stdout = %q", res.Stdout)
	}
}

func TestRunConfirmedMutatingDeclined(t *testing.T) {
	var calls [][]string
	r := fakeRunner(func(string) bool { return false }, nil, &calls)
	res, err := r.RunConfirmed(context.Background(), "app", "delete", "my-app")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Errorf("executed despite decline: %v", calls)
	}
	if res.ExitCode != -1 || !strings.Contains(res.Stdout, "cancelled by user") {
		t.Errorf("declined result = %+v, want exit -1 + cancelled by user", res)
	}
}

func TestRunConfirmedNilConfirmDeclines(t *testing.T) {
	var calls [][]string
	r := fakeRunner(nil, nil, &calls)
	res, err := r.RunConfirmed(context.Background(), "app", "delete", "my-app")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 || res.ExitCode != -1 {
		t.Errorf("nil Confirm must decline: calls=%v res=%+v", calls, res)
	}
}

func TestRunConfirmedLaunchBlocked(t *testing.T) {
	var calls [][]string
	r := fakeRunner(func(string) bool { return true }, nil, &calls)
	res, err := r.RunConfirmed(context.Background(), "launch")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Errorf("launch executed: %v", calls)
	}
	if !strings.Contains(res.Stdout, "interactive wizard") || !strings.Contains(res.Stdout, "refused") {
		t.Errorf("launch refusal = %q", res.Stdout)
	}
	if res.ExitCode != -1 {
		t.Errorf("launch refusal exit = %d, want -1", res.ExitCode)
	}
}

func TestRunConfirmedReadOnlyLaunchNotBlockedByClassification(t *testing.T) {
	// Mutating("launch") is true (safe default), but RunConfirmed refuses
	// it before the confirm gate — verify no double prompt.
	var prompts []string
	r := fakeRunner(func(p string) bool { prompts = append(prompts, p); return true }, nil, nil)
	if _, err := r.RunConfirmed(context.Background(), "launch", "--non-interactive"); err != nil {
		t.Fatal(err)
	}
	if len(prompts) != 0 {
		t.Errorf("launch reached the confirm gate: %v", prompts)
	}
}

func TestSplitCommand(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"postgres list", []string{"postgres", "list"}},
		{"  app   get   my-app ", []string{"app", "get", "my-app"}},
		{`domains add --domain "api.acme.com"`, []string{"domains", "add", "--domain", "api.acme.com"}},
		{`echo 'it'"'"'s fine'`, []string{"echo", "it's fine"}},
		{`run\ shell`, []string{"run shell"}},
		{"", nil},
		{"   ", nil},
	}
	for _, c := range cases {
		if got := SplitCommand(c.in); strings.Join(got, "\x00") != strings.Join(c.want, "\x00") {
			t.Errorf("SplitCommand(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSummary(t *testing.T) {
	res := Result{ExitCode: 1, Stdout: "out", Stderr: "err"}
	s := Summary(res)
	if !strings.Contains(s, "exit code 1") || !strings.Contains(s, "out") || !strings.Contains(s, "err") {
		t.Errorf("Summary = %q", s)
	}
	if strings.Contains(Summary(Result{ExitCode: 0}), "stdout") {
		t.Errorf("Summary rendered empty streams")
	}
}
