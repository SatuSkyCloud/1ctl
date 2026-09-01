package chat

import (
	"context"
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
