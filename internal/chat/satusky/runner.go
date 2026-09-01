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
// (contracts/cli.json). Help requests never prompt. Known read-only
// paths run free; bare commands only mutate for the handful of actions
// that change state (deploy, init, launch, admin); unknown commands and
// subcommands fall back to a keyword heuristic — a mutating verb marks
// the invocation as mutating, anything else runs free (so `1ctl logs X`
// or `1ctl app show X` never prompt).
func Mutating(args []string) bool {
	if hasHelp(args) {
		return false // --help / -h / help: never prompt, ever
	}
	cmd, sub, sub2 := commandParts(args)
	if cmd == "" {
		return false // bare 1ctl prints help; harmless
	}
	if forceMutating[cmd] {
		return true // e.g. admin: every invocation changes state
	}
	if ro, ok := readOnlyCommands[cmd]; ok {
		if len(ro) == 0 {
			return false // action command (doctor): whole command read-only
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
	if sub == "" {
		// Bare command: only a handful of top-level actions change state
		// (deploy, init, launch, admin); every other bare command prints
		// help, which is read-only.
		return bareMutating[cmd]
	}
	// Unknown path: keyword heuristic. Read-only words win first so a
	// benign query word can never be caught by a mutating verb.
	words := []string{cmd, sub, sub2}
	for _, w := range words {
		if readOnlyKeywords[w] {
			return false
		}
	}
	for _, w := range words {
		if mutatingKeywords[w] {
			return true
		}
	}
	return false
}

// Blocked reports whether an invocation is refused outright with a
// reason: interactive wizards, long-running streams, and the nested chat
// recursion guard. Blocked commands never run inside chat, even with
// confirmation. Help requests are never blocked.
func Blocked(args []string) (string, bool) {
	if hasHelp(args) {
		return "", false
	}
	cmd, sub, sub2 := commandParts(args)
	switch cmd {
	case "launch":
		return blockedCommands["launch"], true
	case "postgres":
		if sub == "connect" || sub == "proxy" {
			return blockedCommands[cmd+" "+sub], true
		}
	case "app":
		if sub == "logs" && sub2 == "stream" {
			return blockedCommands["app logs stream"], true
		}
	case "user":
		if sub == "password" {
			return blockedCommands["user password"], true
		}
	case "chat":
		return blockedCommands["chat"], true
	}
	return "", false
}

// hasHelp reports whether the invocation asks for help: the --help/-h
// flags anywhere (case-insensitive), or the bare `help` positional word
// (flag values such as `--name help` do not count, matching how
// commandParts skips them). Help must never prompt, refuse, or block, so
// it short-circuits every classifier.
func hasHelp(args []string) bool {
	// The bare word `help` counts only as a positional word — flag values
	// are skipped exactly like commandParts skips them.
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "" {
			continue
		}
		if a[0] == '-' {
			if !strings.Contains(a, "=") && valueFlags[a] && i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.EqualFold(a, "help") {
			return true
		}
	}
	for _, a := range args {
		if strings.EqualFold(a, "--help") || strings.EqualFold(a, "-h") {
			return true
		}
	}
	return false
}

// BlastWarning inspects a mutating invocation for destructive or
// expensive patterns and returns a human warning when one applies.
func BlastWarning(args []string) (string, bool) {
	cmd, sub, sub2 := commandParts(args)
	if sub == "" {
		return "", false
	}
	// Check the deepest positional word first: nested paths carry the
	// action at sub2 (`postgres firewall enable`, `domains dns update`).
	if w, ok := warningFor(cmd, sub2); ok {
		return w, true
	}
	if w, ok := warningFor(cmd, sub); ok {
		return w, true
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

// warningFor maps a mutating verb to its blast-radius warning.
func warningFor(cmd, verb string) (string, bool) {
	switch verb {
	case "delete", "remove", "drop", "unset", "detach", "disable", "revoke", "logout", "purge", "wipe", "destroy":
		return fmt.Sprintf("this deletes or removes a resource — the action is irreversible (%s %s)", cmd, verb), true
	case "scale":
		return "this changes the replica count — it affects availability and cost", true
	case "rollback", "restart", "redeploy", "update":
		return "this changes a running resource — it affects live traffic", true
	case "enable":
		return "this toggles a resource on — it may affect availability or access", true
	case "rotate-password", "rotate-credentials":
		return "this rotates credentials — existing connections using them will break", true
	case "create", "add":
		return "this provisions a new resource — it costs money on an ongoing basis", true
	case "purchase":
		return "this purchases a resource or domain — it is charged against your credits", true
	case "publish":
		return "this publishes an artifact — it becomes visible to others", true
	case "switch", "use":
		return "this switches the active context (profile, org, or similar)", true
	case "login":
		return "this changes your stored authentication state", true
	case "mark", "read":
		return "this changes notification state", true
	}
	return "", false
}

// RunConfirmed runs a 1ctl command through the gate: blocked commands
// are refused with a reason (no confirmation), read-only commands run
// directly, and mutating commands print a preview line, any blast-radius
// warning, and ask for confirmation. A declined (or missing)
// confirmation returns Result{ExitCode: -1, Stdout: "cancelled by user"}
// without executing.
func (r *Runner) RunConfirmed(ctx context.Context, args ...string) (Result, error) {
	action := strings.Join(args, " ")
	if reason, ok := Blocked(args); ok {
		return Result{
			Command:  action,
			ExitCode: -1,
			Stdout:   "refused: " + reason,
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
// the command's bare action is read-only (e.g. `1ctl doctor`). Anything
// not listed falls through to the keyword heuristic (unknown paths) or
// the bare-command rule.
var readOnlyCommands = map[string][]string{
	"app":           {"list", "get", "status", "logs", "events", "releases", "open"},
	"domains":       {"list", "check", "setup", "available", "search", "purchase-status"},
	"config":        {"list"},
	"env":           {"list"}, // alias of config
	"environment":   {"list"}, // alias of config
	"secret":        {"list", "get"},
	"volumes":       {"list", "inspect"},
	"postgres":      {"list", "get", "status", "credentials", "storage-classes"},
	"valkey":        {"list", "get", "status", "credentials", "metrics", "logs"},
	"nats":          {"list", "get", "status", "credentials"},
	"machine":       {"list", "get", "inspect", "logs", "events", "available"},
	"cluster":       {"list", "zones"},
	"marketplace":   {"list", "get"},
	"package":       {"list", "status"},
	"auth":          {"status"},
	"profile":       {"list", "current"},
	"org":           {"list", "current"},
	"user":          {"me", "permissions"},
	"token":         {"list", "get"},
	"credits":       {"balance", "transactions", "usage"},
	"billing":       {"balance", "transactions", "usage"}, // alias of credits
	"pricing":       {"list", "get", "lookup", "calculate"},
	"audit":         {"list", "get"},
	"notifications": {"list", "count"},
	"completion":    {"bash", "zsh", "fish", "powershell"},
	"service":       {"list"},
	"ingress":       {"list"},
	"doctor":        {}, // bare action is read-only diagnostics
}

// nestedReadOnly lists read-only sub-subcommands for two-level command
// paths, e.g. "domains dns list", "postgres firewall list". Empty list
// would mean the bare two-level path is read-only; here every entry has
// explicit subs and bare paths fall through to the keyword heuristic
// (which treats them as read-only).
var nestedReadOnly = map[string][]string{
	"domains managed":   {"list", "verify"},
	"domains dns":       {"list"},
	"postgres users":    {"list"},
	"postgres firewall": {"list"},
	"valkey users":      {"list"},
	"machine usage":     {"list", "get", "cost"},
	"machine labels":    {"list", "keys"},
	"org team":          {"list"},
	"user sessions":     {"list"},
}

// bareMutating lists top-level commands whose bare action changes state
// (deploys, writes satusky.toml, or runs guarded platform admin). Every
// other bare command prints help, which is read-only.
var bareMutating = map[string]bool{
	"deploy": true,
	"init":   true,
	"launch": true,
	"admin":  true,
}

// forceMutating names commands where EVERY invocation changes state (they
// are never read-only regardless of subcommand), e.g. guarded platform
// administration. --help still short-circuits before this.
var forceMutating = map[string]bool{
	"admin": true,
}

// blockedCommands maps invocations that must never run inside chat to the
// reason they are refused. These need an interactive TTY (wizards,
// password prompts, psql) or spawn long-running processes (port
// forwards, log streams, nested chat sessions).
var blockedCommands = map[string]string{
	"launch":           "`1ctl launch` is an interactive wizard and cannot run inside chat — write a satusky.toml and use `1ctl deploy` instead",
	"postgres connect": "`1ctl postgres connect` spawns interactive psql and cannot run inside chat",
	"postgres proxy":   "`1ctl postgres proxy` opens a long-running port forward and cannot run inside chat",
	"app logs stream":  "`1ctl app logs stream` tails logs forever and cannot run inside chat — use `1ctl app logs` for the recent tail",
	"user password":    "`1ctl user password` is an interactive password prompt and cannot run inside chat",
	"chat":             "`1ctl chat` spawns a nested chat session and cannot run inside chat",
}

// mutatingKeywords are verbs that mark an unknown invocation as
// mutating. They cover every state-changing action across the command
// tree (and then some), so unverified or brand-new commands that mutate
// still hit the confirmation gate instead of running free.
var mutatingKeywords = map[string]bool{
	"delete": true, "remove": true, "drop": true, "create": true, "add": true,
	"set": true, "unset": true, "deploy": true, "scale": true, "rotate": true,
	"restart": true, "rollback": true, "redeploy": true, "update": true,
	"upgrade": true, "downgrade": true, "switch": true, "use": true,
	"login": true, "logout": true, "enable": true, "disable": true,
	"attach": true, "detach": true, "revoke": true, "purchase": true,
	"publish": true, "submit": true, "apply": true, "destroy": true,
	"migrate": true, "reset": true, "install": true, "renew": true,
	"rename": true, "suspend": true, "resume": true, "purge": true,
	"wipe": true, "grant": true, "invite": true, "mark": true, "read": true,
	"change": true, "password": true,
	// Kebab-case verbs that appear as subcommand names in the tree.
	"rotate-password": true, "rotate-credentials": true,
}

// readOnlyKeywords are words that must never be treated as mutating,
// even on unknown commands — the fallback that lets query-style
// invocations like `1ctl logs X` or `1ctl app show X` run free.
var readOnlyKeywords = map[string]bool{
	"verify": true, "check": true, "search": true, "available": true,
	"setup": true, "list": true, "get": true, "status": true, "show": true,
	"inspect": true, "logs": true, "events": true, "releases": true,
	"open": true, "current": true, "me": true, "permissions": true,
	"balance": true, "transactions": true, "usage": true, "lookup": true,
	"calculate": true, "credentials": true, "metrics": true,
	"storage-classes": true, "zones": true, "count": true, "keys": true,
	"cost": true,
}
