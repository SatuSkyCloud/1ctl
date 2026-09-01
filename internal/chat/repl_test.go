package chat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	openai "github.com/sashabaranov/go-openai"
)

func replTestState(t *testing.T, serverURL string) *replState {
	t.Helper()
	info, _ := ParseProvider("openai")
	key := "sk-test-123"
	return &replState{
		info:   info,
		client: NewClientWithBaseURL(key, serverURL),
		key:    key,
		model:  "gpt-4o-mini",
	}
}

func TestRunTurnStreamsToOutput(t *testing.T) {
	color.NoColor = true
	srv, _ := fakeSSEServer(t, []string{streamChunk("Hel"), streamChunk("lo"), streamChunk(" world"), "[DONE]"})
	state := replTestState(t, srv.URL)
	session := NewSession("sys")

	var out strings.Builder
	if err := runTurn(context.Background(), state, session, "hi there", &out, true); err != nil {
		t.Fatalf("runTurn: %v", err)
	}
	if want := "assistant ▸ Hello world\n"; out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
	// Conversation must contain the user turn and the assistant reply.
	raw := session.Raw()
	if len(raw) != 2 {
		t.Fatalf("session has %d messages, want 2", len(raw))
	}
	if raw[0].Role != openai.ChatMessageRoleUser || raw[0].Content != "hi there" {
		t.Errorf("raw[0] = %+v", raw[0])
	}
	if raw[1].Role != openai.ChatMessageRoleAssistant || raw[1].Content != "Hello world" {
		t.Errorf("raw[1] = %+v", raw[1])
	}
}

func TestRunTurnErrorKeepsPartialAndUserMessage(t *testing.T) {
	color.NoColor = true
	var body strings.Builder
	body.WriteString("data: " + streamChunk("Par") + "\n\n")
	body.WriteString(`{"error":{"message":"stream interrupted","type":"server_error"}}` + "\n\n")
	srv := newRawSSEServer(t, body.String())
	state := replTestState(t, srv.URL)
	session := NewSession("sys")

	var out strings.Builder
	err := runTurn(context.Background(), state, session, "question", &out, true)
	if err == nil {
		t.Fatal("expected error from mid-stream failure")
	}
	if !strings.Contains(err.Error(), "stream interrupted") {
		t.Errorf("error = %q", err.Error())
	}
	// The partial assistant text is kept so the conversation stays coherent.
	raw := session.Raw()
	if len(raw) != 2 || raw[1].Content != "Par" {
		t.Errorf("session after failed turn = %+v, want user + partial assistant", raw)
	}
}

func TestRunLoopSlashHelpAndExit(t *testing.T) {
	color.NoColor = true
	st := NewStore(t.TempDir())
	state := replTestState(t, "http://127.0.0.1:1") // no network calls in this test
	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader("/help\n/exit\n"), Stdout: &out}

	if err := runLoop(context.Background(), st, state, opts); err != nil {
		t.Fatalf("runLoop: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "chat(openai·gpt-4o-mini)") {
		t.Errorf("prompt not rendered: %q", got)
	}
	if !strings.Contains(got, "/connect") || !strings.Contains(got, "/model") {
		t.Errorf("help text missing from output: %q", got)
	}
}

func TestRunLoopUnknownSlash(t *testing.T) {
	color.NoColor = true
	st := NewStore(t.TempDir())
	state := replTestState(t, "http://127.0.0.1:1")
	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader("/bogus\n/exit\n"), Stdout: &out}

	if err := runLoop(context.Background(), st, state, opts); err != nil {
		t.Fatalf("runLoop: %v", err)
	}
	if !strings.Contains(out.String(), "unknown command") {
		t.Errorf("unknown command not flagged: %q", out.String())
	}
}

func TestRunLoopEOFExits(t *testing.T) {
	color.NoColor = true
	st := NewStore(t.TempDir())
	state := replTestState(t, "http://127.0.0.1:1")
	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader(""), Stdout: &out}

	if err := runLoop(context.Background(), st, state, opts); err != nil {
		t.Fatalf("runLoop: %v", err)
	}
	if !strings.Contains(out.String(), "chat(openai·gpt-4o-mini)") {
		t.Errorf("prompt not rendered before EOF: %q", out.String())
	}
}

func TestToolProgressLineSatuskyRunRendersCommand(t *testing.T) {
	color.NoColor = true
	line := toolProgressLine(openai.ToolCall{
		Function: openai.FunctionCall{Name: "satusky_run", Arguments: `{"args":["app","status","x"]}`},
	})
	if !strings.Contains(line, "▸ tool: satusky_run 1ctl app status x") {
		t.Errorf("progress line = %q, want the 1ctl command rendered", line)
	}
}

func TestToolProgressLineSatuskyRunCommandString(t *testing.T) {
	color.NoColor = true
	line := toolProgressLine(openai.ToolCall{
		Function: openai.FunctionCall{Name: "satusky_run", Arguments: `{"command":"postgres list"}`},
	})
	if !strings.Contains(line, "1ctl postgres list") {
		t.Errorf("progress line = %q, want the command string rendered", line)
	}
}

func TestToolProgressLineRunShellRendersCommand(t *testing.T) {
	color.NoColor = true
	line := toolProgressLine(openai.ToolCall{
		Function: openai.FunctionCall{Name: "run_shell", Arguments: `{"command":"go test ./...","cwd":"."}`},
	})
	if !strings.Contains(line, "▸ tool: run_shell go test ./...") {
		t.Errorf("progress line = %q, want the shell command rendered", line)
	}
}

func TestToolMarker(t *testing.T) {
	cases := []struct {
		name   string
		result string
		want   string
	}{
		{"run_shell", "exit code 0\n--- stdout ---\nhi", "✓ exit 0"},
		{"satusky_run", "exit code 0\n--- stdout ---\nok", "✓ exit 0"},
		{"run_shell", "exit code 1\n--- stderr ---\nboom\nlong line that keeps going after the first line", "✗ exit code 1"},
		{"satusky_run", "refused: `1ctl launch` is an interactive wizard", "✗ refused: `1ctl launch` is an interactive wizard"},
		{"write_file", "wrote hello.txt", "✓ done"},
		{"read_file", "file contents", "✓ done"},
	}
	for _, c := range cases {
		if got := toolMarker(c.name, c.result); got != c.want {
			t.Errorf("toolMarker(%s, %q) = %q, want %q", c.name, c.result, got, c.want)
		}
	}
}

func TestRunTurnPrintsTokenAndCostFooter(t *testing.T) {
	color.NoColor = true
	events := []string{streamChunk("Hello "), streamChunk("world"), usageChunk(1000, 204), "[DONE]"}
	srv, _ := fakeSSEServer(t, events)
	state := replTestState(t, srv.URL)
	session := NewSession("sys")

	var out strings.Builder
	if err := runTurn(context.Background(), state, session, "hi", &out, true); err != nil {
		t.Fatalf("runTurn: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "1,204 tokens") {
		t.Errorf("token count missing from footer: %q", got)
	}
	// openai rate: 1000 prompt @0.15 + 204 completion @0.60 per 1M → $0.0003.
	if !strings.Contains(got, "· $0.0003") {
		t.Errorf("cost estimate missing from footer: %q", got)
	}
}

func TestRunTurnOmitsFooterWithoutUsage(t *testing.T) {
	color.NoColor = true
	events := []string{streamChunk("Hello "), streamChunk("world"), "[DONE]"}
	srv, _ := fakeSSEServer(t, events)
	state := replTestState(t, srv.URL)
	session := NewSession("sys")

	var out strings.Builder
	if err := runTurn(context.Background(), state, session, "hi", &out, true); err != nil {
		t.Fatalf("runTurn: %v", err)
	}
	if strings.Contains(out.String(), "tokens") {
		t.Errorf("footer printed without usage data: %q", out.String())
	}
}

func TestRunLoopMultilineBackslashContinuation(t *testing.T) {
	color.NoColor = true
	srv, _ := fakeSSEServer(t, []string{streamChunk("ok"), "[DONE]"})
	st := NewStore(t.TempDir())
	state := replTestState(t, srv.URL)
	var out strings.Builder
	// First line ends with a backslash: the loop must keep reading and
	// join the lines before sending the turn.
	opts := ReplOptions{Stdin: strings.NewReader("write a poem \\\nthen stop\n/exit\n"), Stdout: &out}

	if err := runLoop(context.Background(), st, state, opts); err != nil {
		t.Fatalf("runLoop: %v", err)
	}
	// The continuation hint must be shown, and the streamed reply must
	// appear after the joined user message.
	if !strings.Contains(out.String(), "... ") {
		t.Errorf("continuation hint not rendered: %q", out.String())
	}
	if !strings.Contains(out.String(), "assistant ▸ ok") {
		t.Errorf("reply missing after multiline input: %q", out.String())
	}
}

func TestRunLoopMultilineUnbalancedBrace(t *testing.T) {
	color.NoColor = true
	srv, _ := fakeSSEServer(t, []string{streamChunk("ok"), "[DONE]"})
	st := NewStore(t.TempDir())
	state := replTestState(t, srv.URL)
	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader("func foo() {\n  return 1\n}\n/exit\n"), Stdout: &out}

	if err := runLoop(context.Background(), st, state, opts); err != nil {
		t.Fatalf("runLoop: %v", err)
	}
	if !strings.Contains(out.String(), "assistant ▸ ok") {
		t.Errorf("reply missing after multiline brace input: %q", out.String())
	}
}

func TestExportThroughDispatchWritesFile(t *testing.T) {
	color.NoColor = true
	st := NewStore(t.TempDir())
	state := replTestState(t, "http://127.0.0.1:1")
	session := NewSession("sys")
	session.Add(openai.ChatMessageRoleUser, "hello")
	session.Add(openai.ChatMessageRoleAssistant, "hi!")

	cwd := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader(""), Stdout: &out}
	exit, err := dispatchSlash(context.Background(), st, state, session, CmdExport, "out.md", opts)
	if err != nil {
		t.Fatalf("dispatchSlash(/export): %v", err)
	}
	if exit {
		t.Fatal("/export must not exit the REPL")
	}
	if !strings.Contains(out.String(), "transcript saved to") {
		t.Errorf("success message missing: %q", out.String())
	}
	data, err := os.ReadFile(filepath.Join(cwd, "out.md"))
	if err != nil {
		t.Fatalf("read exported file: %v", err)
	}
	if !strings.Contains(string(data), "## assistant") || !strings.Contains(string(data), "hello") {
		t.Errorf("exported content wrong:\n%s", data)
	}
}

func TestExportEmptyConversationThroughDispatch(t *testing.T) {
	color.NoColor = true
	st := NewStore(t.TempDir())
	state := replTestState(t, "http://127.0.0.1:1")
	session := NewSession("sys")

	cwd := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader(""), Stdout: &out}
	if _, err := dispatchSlash(context.Background(), st, state, session, CmdExport, "", opts); err != nil {
		t.Fatalf("dispatchSlash(/export): %v", err)
	}
	if !strings.Contains(out.String(), "nothing to export") {
		t.Errorf("empty-conversation message missing: %q", out.String())
	}
	entries, err := os.ReadDir(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("file created for empty conversation: %v", entries)
	}
}
