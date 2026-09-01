package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	satuskyctx "1ctl/internal/context"
	"1ctl/internal/skill"
	"1ctl/internal/utils"

	"1ctl/internal/chat/satusky"
	chattools "1ctl/internal/chat/tools"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/term"
)

// ReplOptions configures the interactive chat REPL.
type ReplOptions struct {
	// Provider overrides the active provider for this session (--provider).
	// Not persisted unless the user runs /connect or /switch.
	Provider Provider
	// Model overrides the provider's model for this session (--model).
	// Not persisted unless the user runs /model.
	Model string
	// ShowKey echoes the API key while typing during /connect (--show-key).
	ShowKey bool
	// OneShot runs a single user turn and returns — no prompt loop. Used by
	// `1ctl chat "prompt"`.
	OneShot string
	// NoTools disables workspace tools (read/write/list/shell) for this
	// session. Tools default to on; /tools on|off toggles at runtime.
	NoTools bool
	// Stdin/Stdout are the interactive streams; injectable for tests.
	Stdin  io.Reader
	Stdout io.Writer
}

// replState tracks the active provider + model for the current session. It
// is refreshed after /connect and /switch.
type replState struct {
	store   *Store
	info    ProviderInfo
	pc      ProviderConfig
	client  *openai.Client
	key     string
	fromEnv bool
	model   string

	// toolsEnabled requests tool calls (Definitions) on every turn;
	// toggled by /tools and the NoTools opt-out.
	toolsEnabled bool
	// askFirst appends a hard question-first instruction to the system
	// prompt for this session; toggled by /ask and /go.
	askFirst bool
	// exec runs workspace tool calls; built once per Run with Cwd captured
	// at REPL start and Confirm wired to opts.Stdin (declining in OneShot).
	exec *chattools.Executor

	// runner executes real 1ctl commands (read-only freely, mutating only
	// after user confirmation). snapshot is the last SatuSky state digest
	// gathered by refreshSnapshot; it is injected into the system prompt.
	runner     *satusky.Runner
	snapshot   *satusky.Snapshot
	snapshotAt time.Time

	// out is the session's output stream, used for progress spinners
	// (nil in tests keeps them disabled).
	out io.Writer
}

// Run is the entry point for the chat engine: config resolution, guided
// first-run connect, the prompt loop (or a single turn in OneShot mode),
// and slash-command dispatch.
func Run(ctx context.Context, opts ReplOptions) error {
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	// Never emit ANSI escapes into a pipe or test buffer.
	color.NoColor = !stdoutIsTTY(opts.Stdout)

	// Ctrl-C cancels streaming (and exits the loop cleanly at rest).
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	st := NewStore("")
	if _, err := st.Load(); err != nil {
		return fmt.Errorf("load chat config: %w", err)
	}

	state, err := newReplState(st, opts.Provider, opts.Model)
	if err != nil {
		return err
	}

	if state.key == "" {
		if opts.OneShot != "" {
			return fmt.Errorf("no API key configured for %s — run '1ctl chat' and /connect %s, or set %s", state.info.Name, state.info.Name, state.info.EnvKey)
		}
		printFirstRunBanner(opts.Stdout)
		if err := HandleConnect(ctx, st, ConnectOptions{ShowKey: opts.ShowKey, Stdin: opts.Stdin, Stdout: opts.Stdout}); err != nil {
			return err
		}
		state, err = newReplState(st, opts.Provider, opts.Model)
		if err != nil {
			return err
		}
		if state.key == "" {
			return errors.New("connect did not configure an API key")
		}
		if satuskyctx.Default().GetToken() == "" {
			utils.PrintInfo("You're not signed in to SatuSky. Questions and advice work without it, but provisioning databases, domains or deploys need '1ctl auth login'.")
		}
	}

	state.exec = buildExecutor(opts)
	state.toolsEnabled = !opts.NoTools
	state.out = opts.Stdout

	// SatuSky copilot: the same confirmation gate as the workspace tools
	// gates mutating 1ctl commands. The state snapshot is refreshed at
	// REPL start, on /status, and by the satusky_status tool.
	state.runner = satusky.NewRunner(state.exec.Confirm)
	registerSatuskyTools(ctx, state)
	if _, err := refreshSnapshot(ctx, state); err != nil {
		// A broken snapshot must not kill the chat: report once, keep going.
		writef(opts.Stdout, "%s\n", utils.WarnColor("SatuSky state check failed: %v", err))
	}

	if opts.OneShot != "" {
		return runOneShot(ctx, state, opts.OneShot, opts.Stdout)
	}
	return runLoop(ctx, st, state, opts)
}

// newReplState resolves the active provider (honouring a session override),
// the model, and the resolved API key.
func newReplState(st *Store, providerOverride Provider, modelOverride string) (*replState, error) {
	var (
		info ProviderInfo
		pc   ProviderConfig
		err  error
	)
	if providerOverride != "" {
		info, ok := ParseProvider(string(providerOverride))
		if !ok {
			return nil, fmt.Errorf("unknown provider %q — use openai, claude or deepseek", providerOverride)
		}
		cfg, loadErr := st.Load()
		if loadErr != nil {
			return nil, loadErr
		}
		pc = cfg.Providers[info.Name]
	} else {
		info, pc, err = st.Active()
		if err != nil {
			return nil, err
		}
	}
	key, fromEnv := st.ResolvedKey(info, pc)
	return &replState{
		store:   st,
		info:    info,
		pc:      pc,
		client:  NewClient(info, key),
		key:     key,
		fromEnv: fromEnv,
		model:   firstNonEmpty(modelOverride, pc.Model, info.DefaultModel),
	}, nil
}

// runLoop is the interactive REPL: prompt, read, dispatch slash commands or
// stream a completion. Line input goes through a lineReader: a readline
// editor on real terminals (history, arrows, Ctrl-C/D handling), the plain
// bufio path everywhere else (tests, pipes, scripting).
func runLoop(ctx context.Context, st *Store, state *replState, opts ReplOptions) error {
	reader := newLineReader(opts.Stdin, st, opts.Stdout)
	defer reader.Close() //nolint:errcheck // flushes readline history; nothing actionable on failure
	skillContent, err := skill.Load()
	if err != nil {
		return fmt.Errorf("load chat skill: %w", err)
	}
	session := NewSession(skillContent)

	for {
		if ctx.Err() != nil {
			return nil
		}
		line, err := readTurn(reader, opts.Stdout, buildPrompt(state))
		if err != nil {
			if errors.Is(err, io.EOF) {
				writef(opts.Stdout, "\n")
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if ctx.Err() != nil {
			return nil // Ctrl-C while idle
		}

		cmd, arg, isSlash := ParseSlash(line)
		if isSlash {
			exit, err := dispatchSlash(ctx, st, state, session, cmd, arg, opts)
			if err != nil {
				return err
			}
			if exit {
				return nil
			}
			continue
		}

		if state.key == "" {
			writef(opts.Stdout, "%s\n", utils.WarnColor("no API key configured for %s — run /connect %s", state.info.Name, state.info.Name))
			continue
		}

		if reader.Interactive() {
			// Clear the prompt line so the assistant's reply starts on a
			// fresh line; the next read redraws the prompt.
			writef(opts.Stdout, "\r\x1b[K")
		}
		if err := runTurn(ctx, state, session, line, opts.Stdout, true); err != nil {
			if ctx.Err() != nil {
				return nil // Ctrl-C mid-stream: clean stop, partial answer kept
			}
			writef(opts.Stdout, "%s %v\n", utils.ErrorColor("❌"), err)
		}
		if reader.Interactive() {
			// Faint divider between turns, terminal-only.
			writef(opts.Stdout, "%s\n", color.New(color.Faint).Sprint("───"))
		}
	}
}

// errLineCancelled is returned by lineReader.ReadLine when the user aborts
// the current input (Ctrl-C with text on the line). The caller discards the
// partial input and redraws the prompt.
var errLineCancelled = errors.New("input cancelled")

// lineReader abstracts REPL line input: a readline-backed editor on real
// terminals, a plain bufio reader everywhere else.
type lineReader interface {
	// SetPrompt updates the prompt shown before the next ReadLine. It is a
	// no-op for the bufio fallback (the caller prints the prompt itself).
	SetPrompt(string)
	// ReadLine reads one input line. Ctrl-C on an empty line and Ctrl-D
	// return io.EOF (exit the REPL); Ctrl-C with text returns
	// errLineCancelled (cancel the current input).
	ReadLine() (string, error)
	// Interactive reports whether the reader is a TTY line editor, which
	// drives prompt redraw behaviour (e.g. clearing the prompt line before
	// streaming output).
	Interactive() bool
	// Close releases the reader (flushes history for the interactive one).
	Close() error
}

// interactiveReader wraps chzyer/readline: arrows, inline editing and a
// per-session history file under the chat config directory.
type interactiveReader struct {
	rl *readline.Instance
}

// historyLimit bounds the persisted readline history.
const historyLimit = 500

// newInteractiveReader sets up the readline editor: history file at
// <configDir>/chat/history (directory created, file 0600), 500-entry limit,
// and prompts drawn to out.
func newInteractiveReader(stdin *os.File, st *Store, out io.Writer) (*interactiveReader, error) {
	historyPath := st.HistoryPath()
	if err := os.MkdirAll(filepath.Dir(historyPath), 0750); err != nil {
		return nil, fmt.Errorf("create chat history directory: %w", err)
	}
	// Pre-create the history file with restrictive perms so prompts never
	// sit in a world-readable file (readline itself uses 0666).
	if f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_RDWR, 0600); err == nil {
		_ = f.Close() //nolint:errcheck // best-effort permission hardening
	}
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "",
		HistoryFile:  historyPath,
		HistoryLimit: historyLimit,
		Stdin:        stdin,
		Stdout:       out,
	})
	if err != nil {
		return nil, err
	}
	return &interactiveReader{rl: rl}, nil
}

func (r *interactiveReader) SetPrompt(p string) { r.rl.SetPrompt(p) }
func (r *interactiveReader) Interactive() bool  { return true }
func (r *interactiveReader) Close() error       { return r.rl.Close() }

func (r *interactiveReader) ReadLine() (string, error) {
	line, err := r.rl.Readline()
	if err != nil {
		if errors.Is(err, readline.ErrInterrupt) {
			if strings.TrimSpace(line) == "" {
				return "", io.EOF // Ctrl-C on an empty line: leave the REPL
			}
			return "", errLineCancelled // Ctrl-C with text: cancel the input
		}
		return "", err // io.EOF (Ctrl-D) or a real error
	}
	return line, nil
}

// bufioReader is the plain line reader used when stdin is not a terminal
// (tests, pipes, --non-interactive scripting).
type bufioReader struct {
	r *bufio.Reader
}

func newBufioReader(in io.Reader) *bufioReader { return &bufioReader{r: bufio.NewReader(in)} }
func (b *bufioReader) SetPrompt(string)        {}
func (b *bufioReader) Interactive() bool       { return false }
func (b *bufioReader) Close() error            { return nil }

func (b *bufioReader) ReadLine() (string, error) {
	line, err := b.r.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			if line == "" {
				return "", io.EOF
			}
			// Flush a trailing partial line; the next read exits.
			return strings.TrimRight(line, "\r\n"), nil
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// newLineReader builds the REPL line reader. A readline editor is used only
// when opts.Stdin is a real terminal; anything else (injected readers,
// pipes) falls back to bufio. readline setup failures also fall back so the
// chat never dies on editor quirks.
func newLineReader(stdin io.Reader, st *Store, out io.Writer) lineReader {
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		if r, err := newInteractiveReader(f, st, out); err == nil {
			return r
		}
	}
	return newBufioReader(stdin)
}

// readTurn reads a full user turn: the main prompt plus any continuation
// lines needed to complete a multiline input (trailing backslash, unbalanced
// braces or double quotes — see completeMultiline). The continuation prompt
// is the subtle "... " hint. io.EOF exits the REPL; errLineCancelled aborts
// the current input and redraws the prompt.
func readTurn(r lineReader, out io.Writer, prompt string) (string, error) {
	p := prompt
	lines := make([]string, 0, 1)
	for {
		if !r.Interactive() {
			writef(out, "%s", p)
		}
		r.SetPrompt(p)
		line, err := r.ReadLine()
		if err != nil {
			if errors.Is(err, errLineCancelled) {
				return "", nil // user aborted the input; redraw the prompt
			}
			return "", err // io.EOF or a real error
		}
		lines = append(lines, line)
		if completeMultiline(lines) || len(lines) >= maxContinuationLines {
			break
		}
		p = continuationPrompt
	}
	return joinMultiline(lines), nil
}

// runOneShot executes a single user turn without a prompt and returns.
func runOneShot(ctx context.Context, state *replState, prompt string, out io.Writer) error {
	skillContent, err := skill.Load()
	if err != nil {
		return fmt.Errorf("load chat skill: %w", err)
	}
	session := NewSession(skillContent)
	return runTurn(ctx, state, session, prompt, out, false)
}

// maxToolRounds caps the tool-call loop per turn: at most 8 requests (and
// their tool executions) before the agent stops and reports.
const maxToolRounds = 8

// askInstruction is appended to the system prompt while /ask is active:
// an explicit hard instruction that overrides the model's inclination to
// act immediately.
const askInstruction = "\n\n## User instruction\nYou MUST ask up to 3 clarifying questions before taking any action. Do not use tools until the user has answered.\n"

// buildSystem returns the system prompt for this turn: the session skill,
// the question-first instruction when /ask is active, and the SatuSky
// state digest when a snapshot exists (refreshed at REPL start, on
// /status, and by the satusky_status tool).
func buildSystem(session *Session, state *replState) string {
	sys := session.System
	if state.askFirst {
		sys += askInstruction
	}
	if state.snapshot != nil {
		sys += "\n" + state.snapshot.Digest()
	}
	return sys
}

// firstWriteSpinner stops a spinner on the first byte written through it,
// so the "thinking…" animation gives way to streamed output exactly when
// the first delta arrives. The spinner's own Stop is idempotent, so the
// caller can also stop it after an empty reply.
//
// It implements io.Writer so it can stand in for `out` without changing
// StreamCompletion's signature.
type firstWriteSpinner struct {
	sp   *Spinner
	w    io.Writer
	once sync.Once
}

func (f *firstWriteSpinner) Write(p []byte) (int, error) {
	f.once.Do(func() { f.sp.Stop() })
	return f.w.Write(p)
}

// runTurn appends the user message and runs the agent loop: stream a
// completion (with workspace tools when enabled), execute any tool calls
// the model requests, feed the results back, and re-request until the
// model answers without tool calls. Only the final answer is streamed to
// out and recorded in the session; intermediate tool messages stay local
// to the turn (session.Add records the user message + final answer only).
func runTurn(ctx context.Context, state *replState, session *Session, userMsg string, out io.Writer, printPrefix bool) error {
	session.Add(openai.ChatMessageRoleUser, userMsg)
	prefix := ""
	if printPrefix {
		prefix = color.New(color.FgGreen).Sprint("assistant ▸ ")
		writef(out, "%s", prefix)
	}
	messages := make([]openai.ChatCompletionMessage, 0, len(session.Raw())+1)
	messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: buildSystem(session, state)})
	messages = append(messages, session.Raw()...)

	var reqTools []openai.Tool
	if state.toolsEnabled {
		reqTools = chattools.Definitions()
	}

	for round := 0; ; round++ {
		sp := NewSpinner(out, prefix+"thinking…").Elapsed()
		sp.Start()
		res, err := StreamCompletion(ctx, state.client, state.model, messages, reqTools, &firstWriteSpinner{sp: sp, w: out})
		// Stopped by the first streamed delta (firstWriteSpinner) or here
		// for tool-only/empty replies; Stop is idempotent.
		sp.Stop()
		if err != nil {
			writef(out, "\n")
			// Keep any partial text so the conversation stays coherent.
			if res.Text != "" {
				session.Add(openai.ChatMessageRoleAssistant, res.Text)
				session.Trim()
			}
			return err
		}

		if len(res.ToolCalls) == 0 {
			writef(out, "\n")
			if res.Text != "" {
				session.Add(openai.ChatMessageRoleAssistant, res.Text)
				session.Trim()
			}
			printUsage(out, state.info.Name, res)
			return nil
		}

		// The model asked for tools: keep its message (with the calls) for
		// the next request, execute each call, and append the results.
		writef(out, "\n")
		messages = append(messages, openai.ChatCompletionMessage{
			Role:      openai.ChatMessageRoleAssistant,
			Content:   res.Text,
			ToolCalls: res.ToolCalls,
		})
		for _, tc := range res.ToolCalls {
			writef(out, "%s\n", toolProgressLine(tc))
			runSp := NewSpinner(out, "running…").Elapsed()
			runSp.Start()
			result := "error: workspace tools are not available in this session"
			if state.exec != nil {
				result = state.exec.Execute(tc.Function.Name, []byte(tc.Function.Arguments))
			}
			runSp.StopWith(toolMarker(tc.Function.Name, result))
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
		if round+1 >= maxToolRounds {
			writef(out, "%s\n", utils.WarnColor("reached the tool-call limit for this turn (%d rounds) — continuing in a new turn", maxToolRounds))
			return nil
		}
	}
}

// toolMarker renders the completion marker for a tool call: "✓ exit 0"
// when a command tool reported success, "✗ <first line of the result>"
// when it failed or was refused, and "✓ done" for file/list tools.
func toolMarker(name, result string) string {
	switch name {
	case "run_shell", "satusky_run":
		if strings.HasPrefix(strings.TrimSpace(result), "exit code 0") {
			return "✓ exit 0"
		}
		return "✗ " + truncateFirstLine(result, 100)
	default:
		return "✓ done"
	}
}

// truncateFirstLine takes the first line of s, trimmed, capped at
// maxWidth runes with an ellipsis when cut.
func truncateFirstLine(s string, maxWidth int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxWidth {
		runes := []rune(s)
		return string(runes[:maxWidth]) + "…"
	}
	return s
}

// toolProgressLine renders the one-line progress marker for a tool call:
// "▸ tool: <name> <summary>" in cyan.
func toolProgressLine(tc openai.ToolCall) string {
	return color.New(color.FgCyan).Sprint("▸ tool: " + tc.Function.Name + " " + toolCallSummary(tc.Function.Name, tc.Function.Arguments))
}

// toolCallSummary renders a human-readable summary of a tool call's
// arguments for the progress line: the 1ctl command for satusky_run
// (parsed from the JSON args), the shell command for run_shell, and the
// flattened JSON for everything else.
func toolCallSummary(name, args string) string {
	switch name {
	case "satusky_run":
		var parsed struct {
			Args    []string `json:"args"`
			Command string   `json:"command"`
		}
		if err := json.Unmarshal([]byte(args), &parsed); err == nil {
			if len(parsed.Args) > 0 {
				return "1ctl " + strings.Join(parsed.Args, " ")
			}
			if parsed.Command != "" {
				return "1ctl " + parsed.Command
			}
		}
	case "run_shell":
		var parsed struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(args), &parsed); err == nil && parsed.Command != "" {
			return parsed.Command
		}
	}
	return summarizeArgs(args)
}

// summarizeArgs flattens a tool-call argument JSON for the progress line:
// newlines and runs of spaces collapse, and long arguments are truncated.
func summarizeArgs(args string) string {
	s := strings.Join(strings.Fields(args), " ")
	if len(s) > 80 {
		s = s[:80] + "…"
	}
	return strings.TrimSpace(s)
}

// buildExecutor constructs the turn executor for this REPL run. Cwd is the
// directory chat started in (captured once, at Run); Confirm reads y/N
// replies from opts.Stdin. In OneShot mode Confirm always declines so
// scripting never blocks on a prompt.
func buildExecutor(opts ReplOptions) *chattools.Executor {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	confirm := func(action string) bool {
		return confirmAction(opts.Stdin, opts.Stdout, action)
	}
	if opts.OneShot != "" {
		confirm = func(string) bool { return false }
	}
	return chattools.NewExecutor(cwd, confirm)
}

// confirmAction prompts for y/N confirmation, reading the reply from in
// (the same reader the REPL loop uses; a pending confirmation consumes the
// next typed line, which is acceptable). Any read failure declines.
func confirmAction(in io.Reader, out io.Writer, prompt string) bool {
	writef(out, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(in)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// refreshSnapshot re-runs the read-only state batch and caches the result
// on the replState (used at REPL start, by /status, and by the
// satusky_status tool). On a terminal a spinner shows the refresh; it is
// a no-op elsewhere.
func refreshSnapshot(ctx context.Context, state *replState) (*satusky.Snapshot, error) {
	if state.runner == nil {
		return nil, errors.New("SatuSky runner unavailable in this session")
	}
	sp := NewSpinner(state.out, "refreshing SatuSky state…").Elapsed()
	sp.Start()
	// Bound the whole batch: a degraded backend (e.g. the Loki proxy)
	// must never pin the chat for minutes. Per-step CommandTimeout caps
	// individual calls; this caps the total.
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	snap, err := state.runner.Snapshot(ctx)
	if err != nil {
		sp.StopWith(utils.ErrorColor("✗ failed: %v", err))
		return nil, err
	}
	sp.StopWith(utils.SuccessColor("✓ refreshed"))
	state.snapshot = snap
	state.snapshotAt = time.Now()
	return snap, nil
}

// registerSatuskyTools wires the satusky_status and satusky_run tools
// into the workspace executor's custom-dispatch map, keeping the tools
// package free of SatuSky dependencies. The handlers capture ctx so
// Ctrl-C cancels in-flight 1ctl subprocesses.
func registerSatuskyTools(ctx context.Context, state *replState) {
	if state.exec == nil {
		return
	}
	state.exec.Custom = map[string]func(argsJSON []byte) string{
		"satusky_status": func([]byte) string {
			snap, err := refreshSnapshot(ctx, state)
			if err != nil {
				return "error: " + err.Error()
			}
			return snap.Digest()
		},
		"satusky_run": func(argsJSON []byte) string {
			return runSatuskyTool(ctx, state, argsJSON)
		},
	}
}

// runSatuskyTool implements satusky_run: parse {args:[...]} (preferred) or
// {command:"..."} (shell-like splitting), run through the confirmation
// gate, and summarize the result for the model.
func runSatuskyTool(ctx context.Context, state *replState, argsJSON []byte) string {
	if state.runner == nil {
		return "error: SatuSky runner unavailable in this session"
	}
	var args struct {
		Args    []string `json:"args"`
		Command string   `json:"command"`
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "error: invalid arguments for satusky_run: " + err.Error()
	}
	var cmd []string
	switch {
	case len(args.Args) > 0:
		cmd = args.Args
	case strings.TrimSpace(args.Command) != "":
		cmd = satusky.SplitCommand(args.Command)
	default:
		return "error: satusky_run needs \"args\": a JSON array of 1ctl arguments, e.g. {\"args\":[\"postgres\",\"list\"]}"
	}
	if len(cmd) == 0 {
		return "error: satusky_run received no arguments"
	}
	res, err := state.runner.RunConfirmed(ctx, cmd...)
	if err != nil {
		return "error: " + err.Error()
	}
	return satusky.Summary(res)
}

// printUsage renders the token/cost footer after a completion:
// "⏎ 1,204 tokens" plus a cost estimate ("· $0.0003") when a rate is
// known for the provider. Nothing is printed when the provider reports no
// usage.
func printUsage(out io.Writer, provider Provider, res StreamResult) {
	if res.TotalTokens <= 0 {
		return
	}
	line := color.New(color.Faint).Sprint("⏎") + " " + formatThousands(res.TotalTokens) + " tokens"
	if rate, ok := RateFor(provider); ok {
		line += fmt.Sprintf(" · $%.4f", rate.Cost(res.PromptTokens, res.TotalTokens))
	}
	writef(out, "%s\n", line)
}

// dispatchSlash handles one slash command. Returns exit=true to leave the
// REPL. Command errors are printed (the REPL keeps running) except for
// unexpected I/O/config errors, which are returned.
func dispatchSlash(ctx context.Context, st *Store, state *replState, session *Session, cmd SlashCommand, arg string, opts ReplOptions) (bool, error) {
	switch cmd {
	case CmdConnect:
		if err := HandleConnect(ctx, st, ConnectOptions{Provider: arg, ShowKey: opts.ShowKey, Stdin: opts.Stdin, Stdout: opts.Stdout}); err != nil {
			return false, nil // HandleConnect already printed the failure
		}
		return false, reloadState(st, state)

	case CmdSwitch:
		if arg == "" {
			writef(opts.Stdout, "%s\n", utils.ErrorColor("/switch needs a provider name — e.g. /switch claude"))
			return false, nil
		}
		info, ok := ParseProvider(arg)
		if !ok {
			writef(opts.Stdout, "%s\n", utils.ErrorColor("unknown provider %q — use openai, claude or deepseek", arg))
			return false, nil
		}
		pc, _ := st.GetProvider(info.Name)
		key, _ := st.ResolvedKey(info, pc)
		if key == "" {
			writef(opts.Stdout, "%s\n", utils.ErrorColor("no API key configured for %s — run /connect %s or set %s", info.Name, info.Name, info.EnvKey))
			return false, nil
		}
		if !pc.Connected {
			writef(opts.Stdout, "%s\n", utils.InfoColor("Testing %s connection…", info.DisplayName))
			if err := Connect(ctx, NewClient(info, key), info); err != nil {
				writef(opts.Stdout, "%s\n", utils.ErrorColor("connection failed: %v", err))
				return false, nil
			}
			pc.Connected = true
			pc.LastVerifiedAt = time.Now().UTC().Format(time.RFC3339)
			if err := st.SetProvider(info.Name, pc); err != nil {
				return false, err
			}
		}
		if err := st.SetActiveProvider(info.Name); err != nil {
			return false, err
		}
		if err := reloadState(st, state); err != nil {
			return false, err
		}
		writef(opts.Stdout, "%s\n", utils.SuccessColor("switched to %s — model: %s", state.info.Name, state.model))
		return false, nil

	case CmdDisconnect:
		target := state.info.Name
		if arg != "" {
			info, ok := ParseProvider(arg)
			if !ok {
				writef(opts.Stdout, "%s\n", utils.ErrorColor("unknown provider %q — use openai, claude or deepseek", arg))
				return false, nil
			}
			target = info.Name
		}
		if err := st.Disconnect(target); err != nil {
			return false, err
		}
		if target == state.info.Name {
			if err := reloadState(st, state); err != nil {
				return false, err
			}
		}
		writef(opts.Stdout, "%s\n", utils.InfoColor("disconnected %s — API key removed (model and base URL kept)", target))
		return false, nil

	case CmdProviders:
		printProviders(st)
		return false, nil

	case CmdModel:
		if arg == "" {
			writef(opts.Stdout, "%s\n", utils.InfoColor("current model: %s (%s)", state.model, state.info.Name))
			return false, nil
		}
		cfg, err := st.Load()
		if err != nil {
			return false, err
		}
		pc := cfg.Providers[state.info.Name]
		pc.Model = arg
		if err := st.SetProvider(state.info.Name, pc); err != nil {
			return false, err
		}
		state.model = arg
		state.pc = pc
		writef(opts.Stdout, "%s\n", utils.SuccessColor("model set to %s for %s", arg, state.info.Name))
		return false, nil

	case CmdStatus:
		snap, err := refreshSnapshot(ctx, state)
		if err != nil {
			writef(opts.Stdout, "%s\n", utils.ErrorColor("SatuSky state check failed: %v", err))
			return false, nil
		}
		if !snap.Authenticated {
			writef(opts.Stdout, "%s\n", utils.InfoColor("not authenticated — run '1ctl auth login' to connect SatuSky"))
		}
		writef(opts.Stdout, "%s\n", snap.Digest())
		return false, nil

	case CmdTools:
		switch strings.ToLower(arg) {
		case "on":
			state.toolsEnabled = true
		case "off":
			state.toolsEnabled = false
		default:
			state.toolsEnabled = !state.toolsEnabled // bare /tools: toggle
		}
		label := "off"
		if state.toolsEnabled {
			label = "on"
		}
		writef(opts.Stdout, "%s\n", utils.InfoColor("tools: %s — the agent can read/write files, list directories and run shell commands", label))
		return false, nil

	case CmdAsk:
		state.askFirst = true
		writef(opts.Stdout, "%s\n", utils.InfoColor("question-first mode on — I'll ask clarifying questions before acting"))
		return false, nil

	case CmdGo:
		state.askFirst = false
		writef(opts.Stdout, "%s\n", utils.InfoColor("question-first mode off — acting on unambiguous requests directly"))
		return false, nil

	case CmdSkill:
		if arg == "" {
			lines := strings.Count(session.System, "\n") + 1
			writef(opts.Stdout, "%s\n", utils.InfoColor("skill: %s (%d lines)", skill.Name(), lines))
			return false, nil
		}
		cwd, err := os.Getwd()
		if err != nil {
			return false, err
		}
		path, err := resolveWithinCwd(cwd, arg, "skill")
		if err != nil {
			writef(opts.Stdout, "%s\n", utils.ErrorColor("%v", err))
			return false, nil
		}
		content, err := skill.LoadPath(path)
		if err != nil {
			writef(opts.Stdout, "%s\n", utils.ErrorColor("cannot load skill: %v", err))
			return false, nil
		}
		session.System = content
		writef(opts.Stdout, "%s\n", utils.SuccessColor("skill loaded from %s (%d lines) — for this session only", arg, strings.Count(content, "\n")+1))
		return false, nil

	case CmdExport:
		cwd, err := os.Getwd()
		if err != nil {
			return false, err
		}
		path, err := exportTranscript(session, state.info.Name, state.model, cwd, arg)
		if err != nil {
			writef(opts.Stdout, "%s\n", utils.ErrorColor("%v", err))
			return false, nil
		}
		if path == "" {
			writef(opts.Stdout, "%s\n", utils.InfoColor("nothing to export yet — say something first"))
			return false, nil
		}
		writef(opts.Stdout, "%s\n", utils.SuccessColor("transcript saved to %s", path))
		return false, nil

	case CmdClear:
		session.Messages = nil
		writef(opts.Stdout, "%s\n", utils.InfoColor("conversation cleared — connection kept"))
		return false, nil

	case CmdHelp:
		writef(opts.Stdout, "%s\n", helpText)
		return false, nil

	case CmdExit:
		return true, nil

	default: // CmdUnknown with isSlash=true: unrecognized slash command
		name := strings.TrimSpace(strings.TrimPrefix(arg, "/"))
		if name == "" {
			name = arg
		}
		writef(opts.Stdout, "%s\n", utils.WarnColor("unknown command %q — type /help for the command list", name))
		return false, nil
	}
}

// reloadState re-resolves the active provider/model/key from the store and
// swaps it into the live state (used after /connect, /switch, /disconnect).
// Session-level agent settings (tools, ask-first, executor) are preserved.
func reloadState(st *Store, state *replState) error {
	next, err := newReplState(st, "", "")
	if err != nil {
		return err
	}
	next.exec = state.exec
	next.runner = state.runner
	next.snapshot = state.snapshot
	next.snapshotAt = state.snapshotAt
	next.toolsEnabled = state.toolsEnabled
	next.askFirst = state.askFirst
	next.out = state.out
	*state = *next
	return nil
}

// printProviders renders the provider table (provider · model · status).
func printProviders(st *Store) {
	rows := make([][]string, 0, len(AllProviders()))
	for _, p := range AllProviders() {
		pc, _ := st.GetProvider(p.Name)
		model := firstNonEmpty(pc.Model, p.DefaultModel)
		key, fromEnv := st.ResolvedKey(p, pc)
		status := "not connected"
		switch {
		case fromEnv:
			status = "env"
		case key != "" && pc.Connected:
			status = "connected"
		}
		rows = append(rows, []string{string(p.Name), model, status})
	}
	utils.PrintTable([]string{"provider", "model", "status"}, rows)
}

// printFirstRunBanner renders the first-run welcome: a compact
// box-drawing banner (┌─┐│└┘) with the product title plus a hint line.
// Colours degrade to plain text on pipes and in tests via color.NoColor.
func printFirstRunBanner(out io.Writer) {
	title := "1ctl chat — SatuSky Cloud developer copilot"
	width := utf8.RuneCountInString(title) + 2
	box := color.New(color.FgCyan, color.Bold)
	writef(out, "%s\n", box.Sprint("┌"+strings.Repeat("─", width)+"┐"))
	writef(out, "%s %s %s\n", box.Sprint("│"), box.Sprint(title), box.Sprint("│"))
	writef(out, "%s\n", box.Sprint("└"+strings.Repeat("─", width)+"┘"))
	writef(out, "%s\n", color.New(color.Faint).Sprint("  No LLM provider connected yet — 30 seconds to set up."))
}

// buildPrompt renders the REPL prompt: chat(provider·model) [ns] ~/cwd ▸
func buildPrompt(state *replState) string {
	label := color.New(color.FgCyan, color.Bold).Sprintf("chat(%s·%s)", state.info.Name, state.model)
	parts := []string{label}
	ctx := satuskyctx.Default()
	if ctx.GetToken() != "" {
		if ns := ctx.GetCurrentNamespace(); ns != "" {
			parts = append(parts, color.New(color.FgYellow).Sprint("["+ns+"]"))
		}
	}
	parts = append(parts, shortCWD())
	return strings.Join(parts, " ") + " ▸ "
}

// shortCWD renders the working directory with $HOME collapsed to ~.
func shortCWD() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(cwd, home) {
		return "~" + strings.TrimPrefix(cwd, home)
	}
	return cwd
}

// stdoutIsTTY reports whether w is a terminal (drives color.NoColor).
func stdoutIsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// formatThousands renders an integer with thousands separators (1204 → 1,204).
func formatThousands(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		b.WriteByte(',')
	}
	for i := rem; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte(',')
		}
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
