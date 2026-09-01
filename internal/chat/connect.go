package chat

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"1ctl/internal/utils"

	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/term"
)

// connectTimeout bounds the test completion sent during connect and /switch.
const connectTimeout = 30 * time.Second

// Connect sends a minimal chat completion to the provider to prove the API
// key and endpoint work: model = p.DefaultModel, one user message "ping",
// MaxTokens 5. Errors are classified into friendly, actionable messages.
func Connect(ctx context.Context, client *openai.Client, p ProviderInfo) error {
	req := openai.ChatCompletionRequest{
		Model: p.DefaultModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: "ping"},
		},
		MaxTokens: 5,
	}
	if err := withRetry(ctx, func() error {
		_, err := client.CreateChatCompletion(ctx, req)
		return err
	}); err != nil {
		return classifyError(err)
	}
	return nil
}

// ConnectOptions configures HandleConnect.
type ConnectOptions struct {
	// Provider names the provider to connect; when empty the user picks
	// from a numbered menu.
	Provider string
	// ShowKey echoes the key while typing (--show-key); otherwise input
	// is hidden via term.ReadPassword.
	ShowKey bool
	// Stdin/Stdout are the interactive streams (defaults: os.Stdin/out).
	Stdin  io.Reader
	Stdout io.Writer
}

// HandleConnect runs the /connect flow: provider selection (when needed),
// hidden key input, a live test message, and persistence with
// Connected=true. On success the provider is made active and "Connected!"
// is printed via utils.
func HandleConnect(ctx context.Context, st *Store, opts ConnectOptions) error {
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	reader := bufio.NewReader(opts.Stdin)

	info, err := selectProvider(reader, opts.Stdout, opts.Provider)
	if err != nil {
		return err
	}

	writef(opts.Stdout, "Paste your %s API key", info.DisplayName)
	if opts.ShowKey {
		writef(opts.Stdout, ": ")
	} else {
		writef(opts.Stdout, " (input hidden): ")
	}
	key, err := readAPIKey(reader, opts.Stdout, opts.ShowKey)
	if err != nil {
		return err
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	sp := NewSpinner(opts.Stdout, "Testing connection…")
	sp.Start()
	testCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := Connect(testCtx, NewClient(info, key), info); err != nil {
		sp.StopWith(utils.ErrorColor("✗ connection failed: %v", err))
		utils.PrintError("connection failed: %v — nothing was saved; fix the issue and run /connect again", err)
		return err
	}
	sp.StopWith(utils.SuccessColor("✅ Connected! model: %s (change anytime with /model)", info.DefaultModel))

	pc := ProviderConfig{
		APIKey:         key,
		BaseURL:        info.BaseURL,
		Model:          info.DefaultModel,
		Connected:      true,
		LastVerifiedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := st.SetProvider(info.Name, pc); err != nil {
		return err
	}
	if err := st.SetActiveProvider(info.Name); err != nil {
		return err
	}
	return nil
}

// selectProvider resolves the provider for a connect: an explicit name is
// parsed and validated; otherwise a numbered menu is shown.
func selectProvider(reader *bufio.Reader, out io.Writer, name string) (ProviderInfo, error) {
	if name != "" {
		info, ok := ParseProvider(name)
		if !ok {
			return ProviderInfo{}, fmt.Errorf("unknown provider %q — use openai, claude or deepseek", name)
		}
		return info, nil
	}

	providers := AllProviders()
	writef(out, "Choose a provider:\n")
	for i, p := range providers {
		writef(out, "  %d) %-10s %s\n", i+1, p.Name, p.DisplayName)
	}
	writef(out, "Choose a provider [1]: ")
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return ProviderInfo{}, fmt.Errorf("reading provider choice: %w", err)
	}
	choice := strings.TrimSpace(line)
	if choice == "" {
		choice = "1"
	}
	n, err := strconv.Atoi(choice)
	if err != nil || n < 1 || n > len(providers) {
		return ProviderInfo{}, fmt.Errorf("invalid choice %q — pick a number between 1 and %d", choice, len(providers))
	}
	return providers[n-1], nil
}

// writef writes formatted output, ignoring write errors: REPL output is
// best-effort and never worth aborting the session over.
func writef(out io.Writer, format string, a ...interface{}) {
	_, _ = fmt.Fprintf(out, format, a...) //nolint:errcheck
}

// readAPIKey reads the API key: hidden via term.ReadPassword when the
// terminal supports it and ShowKey is false, otherwise a plain line.
func readAPIKey(reader *bufio.Reader, out io.Writer, showKey bool) (string, error) {
	if showKey {
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", fmt.Errorf("reading API key: %w", err)
		}
		return strings.TrimSpace(line), nil
	}

	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err == nil {
		writef(out, "\n")
		return strings.TrimSpace(string(pw)), nil
	}
	// stdin is not a terminal (e.g. tests, pipes): fall back to a line read.
	line, readErr := reader.ReadString('\n')
	if readErr != nil && line == "" {
		return "", fmt.Errorf("reading API key: %w", readErr)
	}
	return strings.TrimSpace(line), nil
}
