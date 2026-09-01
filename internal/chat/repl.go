package chat

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	satuskyctx "1ctl/internal/context"
	"1ctl/internal/skill"
	"1ctl/internal/utils"

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
		writef(opts.Stdout, "1ctl chat — your SatuSky Cloud developer copilot\n")
		utils.PrintInfo("No LLM provider connected yet. 30 seconds to set up.")
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
// stream a completion.
func runLoop(ctx context.Context, st *Store, state *replState, opts ReplOptions) error {
	reader := bufio.NewReader(opts.Stdin)
	session := NewSession(skill.Load())

	for {
		if ctx.Err() != nil {
			return nil
		}
		writef(opts.Stdout, "%s", buildPrompt(state))
		line, err := reader.ReadString('\n')
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

		if err := runTurn(ctx, state, session, line, opts.Stdout, true); err != nil {
			if ctx.Err() != nil {
				return nil // Ctrl-C mid-stream: clean stop, partial answer kept
			}
			writef(opts.Stdout, "%s %v\n", utils.ErrorColor("❌"), err)
		}
	}
}

// runOneShot executes a single user turn without a prompt and returns.
func runOneShot(ctx context.Context, state *replState, prompt string, out io.Writer) error {
	session := NewSession(skill.Load())
	return runTurn(ctx, state, session, prompt, out, false)
}

// runTurn appends the user message, streams the completion to out, and
// records the assistant reply (including any partial text on error).
func runTurn(ctx context.Context, state *replState, session *Session, userMsg string, out io.Writer, printPrefix bool) error {
	session.Add(openai.ChatMessageRoleUser, userMsg)
	if printPrefix {
		writef(out, "%s", color.New(color.FgGreen).Sprint("assistant ▸ "))
	}
	messages := make([]openai.ChatCompletionMessage, 0, len(session.Raw())+1)
	messages = append(messages, session.SystemMessage())
	messages = append(messages, session.Raw()...)

	res, err := StreamCompletion(ctx, state.client, state.model, messages, out)
	writef(out, "\n")
	if res.Text != "" {
		session.Add(openai.ChatMessageRoleAssistant, res.Text)
		session.Trim()
	}
	if err != nil {
		return err
	}
	if res.Tokens > 0 {
		writef(out, "%s %s tokens\n", color.New(color.Faint).Sprint("⏎"), formatThousands(res.Tokens))
	}
	return nil
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
func reloadState(st *Store, state *replState) error {
	next, err := newReplState(st, "", "")
	if err != nil {
		return err
	}
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
