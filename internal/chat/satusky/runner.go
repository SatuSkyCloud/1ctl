// Package satusky implements the SatuSky copilot integration for 1ctl
// chat: it runs the real 1ctl binary (the running executable itself, so
// the agent's powers always match the shipped CLI), classifies commands
// as read-only vs mutating against the real command tree, and gates
// mutating actions behind user confirmation with blast-radius warnings.
package satusky

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Result captures one 1ctl invocation: the command, stdout/stderr (each
// capped), and the exit code. ExitCode is 0 on success; a negative value
// is reserved for internal outcomes (e.g. a declined confirmation).
type Result struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
}

// defaultTimeout bounds every 1ctl subprocess unless the runner is
// configured otherwise (tests shrink it).
const defaultTimeout = 120 * time.Second

// maxStreamChars caps each captured stream (stdout, stderr) from a 1ctl
// invocation so a single tool result never floods the model context.
const maxStreamChars = 8 * 1024

// Runner executes 1ctl commands. Binary defaults to os.Executable() (the
// running chat binary itself); Exec is an injectable executor for tests.
// Timeout bounds each invocation. Confirm gates mutating commands via
// RunConfirmed; a nil Confirm declines.
type Runner struct {
	Binary  string
	Exec    func(ctx context.Context, args ...string) (Result, error)
	Timeout time.Duration
	Confirm func(action string) bool
}

// NewRunner builds a runner around the running binary with the default
// 120s timeout and the given confirmation gate.
func NewRunner(confirm func(string) bool) *Runner {
	return &Runner{Timeout: defaultTimeout, Confirm: confirm}
}

// Run executes exactly the given args (e.g. ["app", "list"]) through the
// injected Exec or the 1ctl binary itself. Stdout and stderr are captured
// separately and each capped at 8k characters.
func (r *Runner) Run(ctx context.Context, args ...string) (Result, error) {
	res, err := r.exec(ctx, args...)
	res.Command = strings.Join(args, " ")
	res.Stdout = capStream(res.Stdout)
	res.Stderr = capStream(res.Stderr)
	return res, err
}

// exec runs one 1ctl invocation: the injected Exec when set, otherwise
// the running binary itself.
func (r *Runner) exec(ctx context.Context, args ...string) (Result, error) {
	if r.Exec != nil {
		return r.Exec(ctx, args...)
	}
	binary := r.Binary
	if binary == "" {
		exe, err := os.Executable()
		if err != nil {
			return Result{}, fmt.Errorf("resolve 1ctl binary: %w", err)
		}
		binary = exe
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	out, err := cmd.Output()
	res := Result{}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			res.Stderr = string(exitErr.Stderr)
		} else {
			return Result{}, err
		}
	}
	res.Stdout = string(out)
	return res, nil
}

// RunJSON runs the command with "-o json" appended (the global output
// flag, values table/wide/json — see contracts/cli.json). If the result
// is not valid JSON — commands like `auth status` ignore the flag and
// print text — it re-runs without the flag (text fallback) and returns
// that result instead.
func (r *Runner) RunJSON(ctx context.Context, args ...string) (Result, error) {
	jsonArgs := append(append([]string{}, args...), "-o", "json")
	res, err := r.Run(ctx, jsonArgs...)
	if err == nil && res.ExitCode == 0 && json.Valid([]byte(res.Stdout)) {
		return res, nil
	}
	return r.Run(ctx, args...)
}

// Mutating classifies a 1ctl invocation against the real command tree
// (contracts/cli.json). Every known read-only path is listed explicitly;
// anything unknown — new commands, typos, aliases we did not map —
// defaults to mutating so the confirmation gate errs on the safe side.
func Mutating(args []string) bool {
	cmd, sub, sub2 := commandParts(args)
	if cmd == "" {
		return false // bare 1ctl prints help; harmless
	}
	if cmd == "launch" {
		return true // interactive wizard; refused in RunConfirmed
	}
	if ro, ok := readOnlyCommands[cmd]; ok {
		if len(ro) == 0 {
			return false // action command (doctor, cluster): read-only
		}
		for _, s := range ro {
			if s == sub {
				return false
			}
		}
	}
	if sub != "" {
		if ro, ok := nestedReadOnly[cmd+" "+sub]; ok {
			if len(ro) == 0 {
				return false
			}
			for _, s := range ro {
				if s == sub2 {
					return false
				}
			}
		}
	}
	return true
}

// BlastWarning inspects a mutating invocation for destructive or
// expensive patterns and returns a human warning when one applies.
func BlastWarning(args []string) (string, bool) {
	cmd, sub, _ := commandParts(args)
	if sub == "" {
		return "", false
	}
	switch sub {
	case "delete", "remove", "drop", "unset", "detach", "disable", "revoke", "logout":
		return fmt.Sprintf("this deletes or removes a resource — the action is irreversible (%s %s)", cmd, sub), true
	case "scale":
		return "this changes the replica count — it affects availability and cost", true
	case "rollback", "restart", "redeploy", "update":
		return "this changes a running resource — it affects live traffic", true
	case "rotate-password", "rotate-credentials":
		return "this rotates credentials — existing connections using them will break", true
	case "create", "add":
		return "this provisions a new resource — it costs money on an ongoing basis", true
	}
	// Plan/scale-up patterns: a size or plan flag on create/update/scale.
	for _, arg := range args {
		key := strings.ToLower(strings.TrimLeft(arg, "-"))
		if i := strings.IndexByte(key, '='); i >= 0 {
			key = key[:i]
		}
		switch key {
		case "plan", "type", "storage-size", "memory", "memory-request", "memory-limit",
			"cpu", "cpu-request", "cpu-limit", "replicas", "instances", "tier":
			return "this specifies a plan or resource size — larger sizes cost more", true
		}
	}
	return "", false
}

// RunConfirmed runs a 1ctl command through the confirmation gate:
// read-only commands run directly; mutating commands print a preview
// line, any blast-radius warning, and ask for confirmation. A declined
// (or missing) confirmation returns Result{ExitCode: -1, Stdout:
// "cancelled by user"} without executing. `launch` is refused outright —
// it is an interactive wizard that cannot run without a TTY.
func (r *Runner) RunConfirmed(ctx context.Context, args ...string) (Result, error) {
	action := strings.Join(args, " ")
	if len(args) > 0 && args[0] == "launch" {
		return Result{
			Command:  action,
			ExitCode: -1,
			Stdout:   "refused: `1ctl launch` is an interactive wizard and cannot run inside chat — write a satusky.toml and use `1ctl deploy` instead",
		}, nil
	}
	if !Mutating(args) {
		return r.Run(ctx, args...)
	}
	prompt := "🔧 1ctl " + action
	if warn, ok := BlastWarning(args); ok {
		prompt += "\n" + warn
	}
	prompt += "\nRun?"
	if r.Confirm == nil || !r.Confirm(prompt) {
		return Result{Command: action, ExitCode: -1, Stdout: "cancelled by user"}, nil
	}
	return r.Run(ctx, args...)
}

// Summary renders a result as trimmed text for the model: the exit code
// plus stdout/stderr streams.
func Summary(res Result) string {
	var b strings.Builder
	b.WriteString("exit code " + strconv.Itoa(res.ExitCode))
	if out := strings.TrimSpace(res.Stdout); out != "" {
		b.WriteString("\n--- stdout ---\n" + out)
	}
	if errOut := strings.TrimSpace(res.Stderr); errOut != "" {
		b.WriteString("\n--- stderr ---\n" + errOut)
	}
	return strings.TrimRight(b.String(), "\n")
}

// SplitCommand parses a command string with shell-like splitting into
// argument tokens: whitespace separates, double and single quotes group
// ("…" with \-escapes, '…' literal), and backslashes escape the next
// character outside quotes. Used for satusky_run's optional "command"
// argument.
func SplitCommand(s string) []string {
	var args []string
	var cur strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if cur.Len() > 0 {
			args = append(args, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\':
			// Inside double quotes a backslash stays literal unless it
			// escapes ", \ or the quote itself; inside single quotes it
			// is always literal.
			if quote == '\'' {
				cur.WriteRune(r)
			} else {
				escaped = true
			}
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return args
}

// commandParts extracts the first three positional words of an argument
// list, skipping flags (and the values of known value-taking flags) so
// classification keys on the real command path even when global flags
// precede the command (e.g. "--profile prod app list").
func commandParts(args []string) (cmd, sub, sub2 string) {
	pos := make([]string, 0, 3)
	for i := 0; i < len(args) && len(pos) < 3; i++ {
		a := args[i]
		if a == "" {
			continue
		}
		if a[0] == '-' {
			// --flag=value is self-contained; --flag value needs the next
			// token skipped when the flag is known to take a value.
			if !strings.Contains(a, "=") && valueFlags[a] && i+1 < len(args) {
				i++
			}
			continue
		}
		pos = append(pos, a)
	}
	if len(pos) > 0 {
		cmd = pos[0]
	}
	if len(pos) > 1 {
		sub = pos[1]
	}
	if len(pos) > 2 {
		sub2 = pos[2]
	}
	return cmd, sub, sub2
}

// valueFlags are flags that take a value (so commandParts skips the
// following token). Unknown flags are treated as booleans; that is safe
// because classification keys on the command path, and flag values only
// matter for nested sub-subcommand lookups.
var valueFlags = map[string]bool{
	"--profile": true, "-p": true, "--api-url": true, "--output": true, "-o": true,
	"--app": true, "--deployment-id": true, "--config": true, "--name": true,
	"--domain": true, "--key": true, "--id": true, "--volume-id": true,
	"--storage-id": true, "--version": true, "--token": true, "--url": true,
	"--email": true, "--role": true, "--replicas": true, "--plan": true,
	"--type": true, "--region": true, "--zone": true, "--limit": true,
	"--days": true, "--user": true, "--database": true, "--cpu": true,
	"--memory": true, "--storage-size": true, "--port": true, "--image": true,
	"--dockerfile": true, "--env": true, "--kv": true, "--from-file": true,
	"--bind-addr": true, "--local-port": true, "--description": true,
	"--expires": true, "--path": true, "--record-id": true, "--data": true,
	"--ttl": true, "--priority": true, "--weight": true, "--period": true,
	"--org-id": true, "--org-name": true, "--reason": true,
	"--expected-uid": true, "--expected-resource-version": true,
	"--expected-generation": true, "--request-id": true, "--health-path": true,
	"--probe": true, "--ip": true, "--machine-id": true, "--topology": true,
	"--persistence": true, "--cpu-request": true, "--memory-request": true,
	"--append-fsync": true, "--preset": true, "--key-pattern": true,
	"--channel-pattern": true, "--output-dir": true, "--tail": true,
	"--source": true, "--since": true, "--filter": true, "--component": true,
	"--previous": true, "--refresh": true, "--min-cpu": true, "--min-memory": true,
	"--gpu": true, "--pricing-tier": true, "--hostname": true, "--chart": true,
	"--first-name": true, "--last-name": true, "--phone-country-code": true,
	"--phone-number": true, "--street": true,
}

// capStream trims whitespace and truncates a captured stream to 8k
// characters with a truncation notice.
func capStream(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxStreamChars {
		s = s[:maxStreamChars] + "\n…(truncated)"
	}
	return s
}

// readOnlyCommands lists every top-level command's read-only subcommands,
// verified against contracts/cli.json. An entry with an empty list means
// the command's bare action is read-only (e.g. `1ctl doctor`).
var readOnlyCommands = map[string][]string{
	"app":           {"list", "get", "status", "logs", "stream", "events", "releases", "open"},
	"doctor":        {},
	"domains":       {"list", "check", "setup", "available", "search", "purchase-status"},
	"config":        {"list"},
	"env":           {"list"}, // alias of config
	"environment":   {"list"}, // alias of config
	"secret":        {"list", "get"},
	"volumes":       {"list", "inspect"},
	"postgres":      {"list", "get", "status", "credentials", "connect", "storage-classes"},
	"valkey":        {"list", "get", "status", "credentials", "metrics", "logs"},
	"nats":          {"list", "get", "status", "credentials"},
	"machine":       {"list", "get", "inspect", "logs", "events", "available", "keys"},
	"cluster":       {},
	"marketplace":   {"list", "get"},
	"package":       {"list", "status"},
	"auth":          {"status"},
	"profile":       {"list", "current"},
	"org":           {"list", "current"},
	"team":          {"list"},
	"user":          {"me", "permissions", "sessions"},
	"token":         {"list", "get"},
	"credits":       {"balance", "transactions", "usage"},
	"billing":       {"balance", "transactions", "usage"}, // alias of credits
	"pricing":       {"list", "get", "lookup", "calculate"},
	"audit":         {"list", "get"},
	"notifications": {"list", "count"},
	"completion":    {"install", "bash", "zsh", "fish", "powershell"},
	"service":       {"list"},
	"ingress":       {"list"},
}

// nestedReadOnly lists read-only sub-subcommands for two-level command
// paths, e.g. "domains dns list", "postgres firewall list".
var nestedReadOnly = map[string][]string{
	"domains managed":   {"list", "verify"},
	"domains dns":       {"list"},
	"postgres users":    {"list"},
	"postgres firewall": {"list"},
	"valkey users":      {"list"},
	"machine usage":     {"list", "get", "cost"},
	"machine labels":    {"list"},
	"cluster zones":     {"list"},
}
