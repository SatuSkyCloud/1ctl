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
		// Help never prompts — even on mutating commands.
		{[]string{"deploy", "--help"}, false, "deploy --help never prompts"},
		{[]string{"deploy", "-h"}, false, "deploy -h never prompts"},
		{[]string{"app", "--help"}, false, "app --help never prompts"},
		{[]string{"app", "delete", "my-app", "--help"}, false, "mutating command with --help never prompts"},
		{[]string{"app", "delete", "my-app", "-h"}, false, "-h short-circuits a delete"},
		{[]string{"app", "delete", "my-app", "help"}, false, "help word anywhere short-circuits"},
		{[]string{"help"}, false, "bare help is read-only"},
		{[]string{"help", "app"}, false, "help app is read-only"},
		{[]string{"--help", "app", "delete", "my-app"}, false, "leading --help never prompts"},

		// Verified read-only paths (contracts/cli.json).
		{[]string{"app", "list"}, false, "app list is read-only"},
		{[]string{"app", "get", "my-app"}, false, "app get is read-only"},
		{[]string{"app", "status"}, false, "app status is read-only"},
		{[]string{"app", "logs", "my-app"}, false, "app logs is read-only"},
		{[]string{"app", "events", "my-app"}, false, "app events is read-only"},
		{[]string{"app", "releases", "my-app"}, false, "app releases is read-only"},
		{[]string{"app", "open", "my-app"}, false, "app open is read-only"},
		{[]string{"postgres", "list"}, false, "postgres list is read-only"},
		{[]string{"postgres", "get", "abc"}, false, "postgres get is read-only"},
		{[]string{"postgres", "status", "abc"}, false, "postgres status is read-only"},
		{[]string{"postgres", "credentials"}, false, "postgres credentials is read-only"},
		{[]string{"postgres", "storage-classes"}, false, "postgres storage-classes is read-only"},
		{[]string{"postgres", "firewall", "list"}, false, "postgres firewall list is read-only"},
		{[]string{"postgres", "users", "list"}, false, "postgres users list is read-only"},
		{[]string{"valkey", "list"}, false, "valkey list is read-only"},
		{[]string{"valkey", "get", "v1"}, false, "valkey get is read-only"},
		{[]string{"valkey", "metrics"}, false, "valkey metrics is read-only"},
		{[]string{"valkey", "logs"}, false, "valkey logs is read-only"},
		{[]string{"valkey", "users", "list"}, false, "valkey users list is read-only"},
		{[]string{"nats", "list"}, false, "nats list is read-only"},
		{[]string{"nats", "get", "n1"}, false, "nats get is read-only"},
		{[]string{"nats", "status", "n1"}, false, "nats status is read-only"},
		{[]string{"nats", "credentials"}, false, "nats credentials is read-only"},
		{[]string{"domains", "list"}, false, "domains list is read-only"},
		{[]string{"domains", "check", "api.acme.com"}, false, "domains check is read-only"},
		{[]string{"domains", "setup", "x.com"}, false, "domains setup is read-only"},
		{[]string{"domains", "available", "x.com"}, false, "domains available is read-only"},
		{[]string{"domains", "search", "acme"}, false, "domains search is read-only"},
		{[]string{"domains", "purchase-status", "x.com"}, false, "domains purchase-status is read-only"},
		{[]string{"domains", "dns", "list", "--domain", "x.com"}, false, "domains dns list is read-only"},
		{[]string{"domains", "managed", "list"}, false, "domains managed list is read-only"},
		{[]string{"domains", "managed", "verify"}, false, "domains managed verify is read-only"},
		{[]string{"config", "list"}, false, "config list is read-only"},
		{[]string{"env", "list"}, false, "env list (alias) is read-only"},
		{[]string{"environment", "list"}, false, "environment list (alias) is read-only"},
		{[]string{"secret", "list"}, false, "secret list is read-only"},
		{[]string{"secret", "get", "abc"}, false, "secret get is read-only"},
		{[]string{"volumes", "list"}, false, "volumes list is read-only"},
		{[]string{"volumes", "inspect", "v1"}, false, "volumes inspect is read-only"},
		{[]string{"machine", "list"}, false, "machine list is read-only"},
		{[]string{"machine", "get", "m1"}, false, "machine get is read-only"},
		{[]string{"machine", "inspect", "m1"}, false, "machine inspect is read-only"},
		{[]string{"machine", "logs", "m1"}, false, "machine logs is read-only"},
		{[]string{"machine", "events"}, false, "machine events is read-only"},
		{[]string{"machine", "available"}, false, "machine available is read-only"},
		{[]string{"machine", "usage", "cost"}, false, "machine usage cost is read-only"},
		{[]string{"machine", "labels", "list"}, false, "machine labels list is read-only"},
		{[]string{"machine", "labels", "keys"}, false, "machine labels keys is read-only"},
		{[]string{"cluster", "list"}, false, "cluster list is read-only"},
		{[]string{"cluster", "zones"}, false, "cluster zones is read-only"},
		{[]string{"cluster", "zones", "list"}, false, "cluster zones list is read-only"},
		{[]string{"marketplace", "list"}, false, "marketplace list is read-only"},
		{[]string{"marketplace", "get", "x"}, false, "marketplace get is read-only"},
		{[]string{"package", "list"}, false, "package list is read-only"},
		{[]string{"package", "status"}, false, "package status is read-only"},
		{[]string{"auth", "status"}, false, "auth status is read-only"},
		{[]string{"profile", "list"}, false, "profile list is read-only"},
		{[]string{"profile", "current"}, false, "profile current is read-only"},
		{[]string{"org", "list"}, false, "org list is read-only"},
		{[]string{"org", "current"}, false, "org current is read-only"},
		{[]string{"org", "team", "list"}, false, "org team list is read-only"},
		{[]string{"user", "me"}, false, "user me is read-only"},
		{[]string{"user", "permissions"}, false, "user permissions is read-only"},
		{[]string{"user", "sessions"}, false, "bare user sessions prints help -> read-only"},
		{[]string{"user", "sessions", "list"}, false, "user sessions list is read-only"},
		{[]string{"token", "list"}, false, "token list is read-only"},
		{[]string{"token", "get", "t1"}, false, "token get is read-only"},
		{[]string{"credits", "balance"}, false, "credits balance is read-only"},
		{[]string{"credits", "transactions"}, false, "credits transactions is read-only"},
		{[]string{"credits", "usage"}, false, "credits usage is read-only"},
		{[]string{"billing", "balance"}, false, "billing balance (alias) is read-only"},
		{[]string{"pricing", "list"}, false, "pricing list is read-only"},
		{[]string{"pricing", "get", "p1"}, false, "pricing get is read-only"},
		{[]string{"pricing", "lookup"}, false, "pricing lookup is read-only"},
		{[]string{"pricing", "calculate"}, false, "pricing calculate is read-only"},
		{[]string{"audit", "list"}, false, "audit list is read-only"},
		{[]string{"audit", "get", "a1"}, false, "audit get is read-only"},
		{[]string{"notifications", "list"}, false, "notifications list is read-only"},
		{[]string{"notifications", "count"}, false, "notifications count is read-only"},
		{[]string{"completion", "bash"}, false, "completion bash is read-only"},
		{[]string{"completion", "zsh"}, false, "completion zsh is read-only"},
		{[]string{"completion", "fish"}, false, "completion fish is read-only"},
		{[]string{"completion", "powershell"}, false, "completion powershell is read-only"},
		{[]string{"service", "list"}, false, "service list is read-only"},
		{[]string{"ingress", "list"}, false, "ingress list is read-only"},
		{[]string{"doctor"}, false, "doctor is read-only"},
		{[]string{"doctor", "my-app"}, false, "doctor with app arg is read-only"},

		// Mutating paths.
		{[]string{"app", "delete", "my-app"}, true, "app delete mutates"},
		{[]string{"app", "restart", "my-app"}, true, "app restart mutates"},
		{[]string{"app", "rollback", "my-app", "--version", "2"}, true, "app rollback mutates"},
		{[]string{"app", "scale", "my-app", "3"}, true, "app scale mutates"},
		{[]string{"postgres", "create", "--name", "db"}, true, "postgres create mutates"},
		{[]string{"postgres", "delete", "abc"}, true, "postgres delete mutates"},
		{[]string{"postgres", "redeploy", "abc"}, true, "postgres redeploy mutates"},
		{[]string{"postgres", "firewall", "add"}, true, "postgres firewall add mutates"},
		{[]string{"postgres", "firewall", "enable"}, true, "postgres firewall enable mutates"},
		{[]string{"postgres", "firewall", "disable"}, true, "postgres firewall disable mutates"},
		{[]string{"postgres", "firewall", "delete"}, true, "postgres firewall delete mutates"},
		{[]string{"postgres", "users", "create"}, true, "postgres users create mutates"},
		{[]string{"valkey", "rotate-password"}, true, "valkey rotate-password mutates"},
		{[]string{"valkey", "rotate-credentials"}, true, "valkey rotate-credentials mutates"},
		{[]string{"valkey", "users", "create"}, true, "valkey users create mutates"},
		{[]string{"valkey", "update", "v1"}, true, "valkey update mutates"},
		{[]string{"valkey", "restart", "v1"}, true, "valkey restart mutates"},
		{[]string{"nats", "create", "--jetstream"}, true, "nats create mutates"},
		{[]string{"nats", "delete", "x"}, true, "nats delete mutates"},
		{[]string{"domains", "add", "--domain", "x.com", "--app", "my-app"}, true, "domains add mutates"},
		{[]string{"domains", "delete", "x.com"}, true, "domains delete mutates"},
		{[]string{"domains", "purchase", "x.com"}, true, "domains purchase mutates"},
		{[]string{"domains", "dns", "create"}, true, "domains dns create mutates"},
		{[]string{"domains", "dns", "update"}, true, "domains dns update mutates"},
		{[]string{"domains", "dns", "delete"}, true, "domains dns delete mutates"},
		{[]string{"domains", "managed", "add"}, true, "domains managed add mutates"},
		{[]string{"domains", "managed", "delete"}, true, "domains managed delete mutates"},
		{[]string{"config", "create", "--env", "A=B"}, true, "config create mutates"},
		{[]string{"config", "unset"}, true, "config unset mutates"},
		{[]string{"env", "unset", "--key", "A"}, true, "env unset mutates"},
		{[]string{"secret", "create"}, true, "secret create mutates"},
		{[]string{"secret", "unset"}, true, "secret unset mutates"},
		{[]string{"volumes", "detach", "v1"}, true, "volumes detach mutates"},
		{[]string{"volumes", "delete", "v1"}, true, "volumes delete mutates"},
		{[]string{"machine", "create"}, true, "machine create mutates"},
		{[]string{"machine", "update", "m1"}, true, "machine update mutates"},
		{[]string{"machine", "delete", "m1"}, true, "machine delete mutates"},
		{[]string{"machine", "labels", "set"}, true, "machine labels set mutates"},
		{[]string{"machine", "labels", "unset"}, true, "machine labels unset mutates"},
		{[]string{"marketplace", "deploy", "x"}, true, "marketplace deploy mutates"},
		{[]string{"package", "create"}, true, "package create mutates"},
		{[]string{"package", "publish"}, true, "package publish mutates"},
		{[]string{"package", "delete"}, true, "package delete mutates"},
		{[]string{"auth", "login"}, true, "auth login mutates"},
		{[]string{"auth", "logout"}, true, "auth logout mutates"},
		{[]string{"profile", "create", "prod"}, true, "profile create mutates"},
		{[]string{"profile", "use", "prod"}, true, "profile use mutates"},
		{[]string{"profile", "delete", "prod"}, true, "profile delete mutates"},
		{[]string{"org", "switch"}, true, "org switch mutates"},
		{[]string{"org", "create"}, true, "org create mutates"},
		{[]string{"org", "delete"}, true, "org delete mutates"},
		{[]string{"org", "team", "add"}, true, "org team add mutates"},
		{[]string{"org", "team", "delete"}, true, "org team delete mutates"},
		{[]string{"user", "update"}, true, "user update mutates"},
		{[]string{"user", "sessions", "revoke"}, true, "user sessions revoke mutates"},
		{[]string{"token", "create"}, true, "token create mutates"},
		{[]string{"token", "enable"}, true, "token enable mutates"},
		{[]string{"token", "disable"}, true, "token disable mutates"},
		{[]string{"token", "delete"}, true, "token delete mutates"},
		{[]string{"notifications", "read"}, true, "notifications read mutates"},
		{[]string{"notifications", "delete"}, true, "notifications delete mutates"},
		{[]string{"completion", "install"}, true, "completion install mutates (edits shell configs)"},
		{[]string{"service", "delete"}, true, "service delete mutates"},
		{[]string{"ingress", "delete"}, true, "ingress delete mutates"},
		{[]string{"deploy", "--port", "3000"}, true, "deploy mutates"},
		{[]string{"init"}, true, "init mutates"},
		{[]string{"launch"}, true, "launch is mutating-class (refused in RunConfirmed)"},

		// Bare commands: only deploy/init/launch/admin mutate.
		{[]string{"app"}, false, "bare app prints help -> read-only"},
		{[]string{"domains"}, false, "bare domains prints help -> read-only"},
		{[]string{"postgres"}, false, "bare postgres prints help -> read-only"},
		{[]string{"auth"}, false, "bare auth prints help -> read-only"},
		{[]string{"profile"}, false, "bare profile prints help -> read-only"},
		{[]string{"org"}, false, "bare org prints help -> read-only"},
		{[]string{"user"}, false, "bare user prints help -> read-only"},
		{[]string{"deploy"}, true, "bare deploy mutates"},
		{[]string{"admin"}, true, "bare admin mutates (guarded platform ops)"},
		{[]string{"admin", "deployment", "adopt"}, true, "admin subcommands always mutate (adopt is state-changing)"},
		{[]string{"admin", "deployment", "routing-adopt", "--help"}, false, "admin --help never prompts"},

		// Unknown commands/subcommands: keyword heuristic.
		{[]string{"logs", "my-app"}, false, "1ctl logs X (nonexistent top-level) runs free"},
		{[]string{"app", "show", "my-app"}, false, "1ctl app show X (nonexistent sub) runs free"},
		{[]string{"app", "frobnicate"}, false, "unknown subcommand without a verb runs free"},
		{[]string{"mystery", "sub"}, false, "unknown command runs free"},
		{[]string{"foo", "bar", "baz"}, false, "unknown command with benign words runs free"},
		{[]string{"frobnicate", "delete"}, true, "unknown command with a mutating verb is mutating"},
		{[]string{"things", "create", "widget"}, true, "unknown command with create is mutating"},
		{[]string{"app", "frobnicate", "restart"}, true, "unknown sub with restart is mutating"},
		{[]string{"--profile", "prod", "app", "list"}, false, "global flag before command still classifies"},
		{[]string{}, false, "bare invocation is harmless"},
	}
	for _, c := range cases {
		if got := Mutating(c.args); got != c.want {
			t.Errorf("Mutating(%v) = %v, want %v (%s)", c.args, got, c.want, c.label)
		}
	}
}

func TestBlocked(t *testing.T) {
	cases := []struct {
		args   []string
		reason string // substring expected; empty means not blocked
	}{
		{[]string{"launch"}, "interactive wizard"},
		{[]string{"launch", "--non-interactive"}, "interactive wizard"},
		{[]string{"postgres", "connect", "db1"}, "psql"},
		{[]string{"postgres", "proxy", "db1", "--local-port", "5432"}, "port forward"},
		{[]string{"app", "logs", "stream", "my-app"}, "tails logs"},
		{[]string{"user", "password"}, "interactive"},
		{[]string{"chat"}, "nested chat"},
		{[]string{"chat", "what can you do?"}, "nested chat"},
		// Help is never blocked.
		{[]string{"chat", "--help"}, ""},
		{[]string{"launch", "-h"}, ""},
		{[]string{"postgres", "connect", "--help"}, ""},
		// Not blocked.
		{[]string{"postgres", "list"}, ""},
		{[]string{"app", "logs", "my-app"}, ""},
		{[]string{"app", "logs", "streaming"}, ""}, // not the stream subcommand
		{[]string{"deploy"}, ""},
		{[]string{"doctor"}, ""},
	}
	for _, c := range cases {
		reason, ok := Blocked(c.args)
		if c.reason == "" {
			if ok {
				t.Errorf("Blocked(%v) = %q, want not blocked", c.args, reason)
			}
			continue
		}
		if !ok {
			t.Errorf("Blocked(%v) = not blocked, want reason containing %q", c.args, c.reason)
			continue
		}
		if !strings.Contains(reason, c.reason) {
			t.Errorf("Blocked(%v) reason = %q, want substring %q", c.args, reason, c.reason)
		}
	}
}

func TestRunConfirmedHelpNeverPrompts(t *testing.T) {
	for _, args := range [][]string{
		{"deploy", "--help"},
		{"app", "--help"},
		{"deploy", "-h"},
		{"app", "delete", "my-app", "--help"},
	} {
		var calls [][]string
		confirmed := 0
		r := fakeRunner(func(string) bool { confirmed++; return true }, nil, &calls)
		res, err := r.RunConfirmed(context.Background(), args...)
		if err != nil {
			t.Fatalf("RunConfirmed(%v): %v", args, err)
		}
		if confirmed != 0 {
			t.Errorf("RunConfirmed(%v): Confirm called %d times for a help request", args, confirmed)
		}
		if len(calls) != 1 {
			t.Errorf("RunConfirmed(%v): calls = %v, want the help invocation executed", args, calls)
		}
		if res.ExitCode != 0 {
			t.Errorf("RunConfirmed(%v): exit = %d, want 0", args, res.ExitCode)
		}
	}
}

func TestRunConfirmedReadOnlyQueriesRunFree(t *testing.T) {
	for _, args := range [][]string{
		{"app", "logs", "my-app"},
		{"app", "get", "my-app"},
		{"app", "status", "my-app"},
		{"doctor"},
		{"postgres", "list"},
		{"credits", "balance"},
		{"logs", "my-app"},        // nonexistent top-level: still free
		{"app", "show", "my-app"}, // nonexistent subcommand: still free
	} {
		var calls [][]string
		confirmed := 0
		r := fakeRunner(func(string) bool { confirmed++; return true }, nil, &calls)
		if _, err := r.RunConfirmed(context.Background(), args...); err != nil {
			t.Fatalf("RunConfirmed(%v): %v", args, err)
		}
		if confirmed != 0 {
			t.Errorf("RunConfirmed(%v): Confirm called %d times, want 0", args, confirmed)
		}
		if len(calls) != 1 {
			t.Errorf("RunConfirmed(%v): calls = %v, want executed without confirmation", args, calls)
		}
	}
}

func TestRunConfirmedMutatingPrompts(t *testing.T) {
	for _, args := range [][]string{
		{"app", "delete", "my-app"},
		{"postgres", "create", "--name", "x"},
		{"deploy"},
		{"config", "unset"},
		{"notifications", "read"},
		{"org", "switch"},
	} {
		var calls [][]string
		confirmed := 0
		r := fakeRunner(func(string) bool { confirmed++; return true }, nil, &calls)
		if _, err := r.RunConfirmed(context.Background(), args...); err != nil {
			t.Fatalf("RunConfirmed(%v): %v", args, err)
		}
		if confirmed != 1 {
			t.Errorf("RunConfirmed(%v): Confirm called %d times, want 1", args, confirmed)
		}
		if len(calls) != 1 {
			t.Errorf("RunConfirmed(%v): calls = %v, want executed after confirmation", args, calls)
		}
	}
}

func TestRunConfirmedBlockedRefused(t *testing.T) {
	for _, args := range [][]string{
		{"launch"},
		{"postgres", "connect", "db1"},
		{"postgres", "proxy", "db1"},
		{"app", "logs", "stream", "my-app"},
		{"user", "password"},
		{"chat"},
	} {
		var calls [][]string
		confirmed := 0
		r := fakeRunner(func(string) bool { confirmed++; return true }, nil, &calls)
		res, err := r.RunConfirmed(context.Background(), args...)
		if err != nil {
			t.Fatalf("RunConfirmed(%v): %v", args, err)
		}
		if len(calls) != 0 {
			t.Errorf("RunConfirmed(%v): executed despite block: %v", args, calls)
		}
		if confirmed != 0 {
			t.Errorf("RunConfirmed(%v): Confirm called for a blocked command", args)
		}
		if res.ExitCode != -1 || !strings.Contains(res.Stdout, "refused") {
			t.Errorf("RunConfirmed(%v) = %+v, want refusal with exit -1", args, res)
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
		{[]string{"app", "restart", "my-app"}, true, "restart warns"},
		{[]string{"app", "rollback", "my-app"}, true, "rollback warns"},
		{[]string{"org", "switch"}, true, "switch warns"},
		{[]string{"notifications", "read"}, true, "read warns"},
		{[]string{"postgres", "firewall", "enable"}, true, "enable warns"},
		{[]string{"domains", "purchase", "x.com"}, true, "purchase warns"},
		{[]string{"package", "publish"}, true, "publish warns"},
		{[]string{"auth", "login"}, true, "login warns"},
		{[]string{"profile", "use", "prod"}, true, "use warns"},
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
