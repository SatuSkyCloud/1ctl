package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	chattools "1ctl/internal/chat/tools"

	"github.com/fatih/color"
	openai "github.com/sashabaranov/go-openai"
)

// toolLoopServer streams one SSE response per request: the first request
// gets toolCall events, every later request gets the given final events.
// It records the request bodies for assertions.
func toolLoopServer(t *testing.T, toolEvents []string, finalEvents []string) (*httptest.Server, *[][]byte) {
	t.Helper()
	var bodies [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, raw)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		events := toolEvents
		if len(bodies) > 1 {
			events = finalEvents
		}
		for _, e := range events {
			fmt.Fprintf(w, "data: %s\n\n", e)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &bodies
}

// stateWithTools builds a replState with tools enabled and an executor
// sandboxed to dir (confirm injected by the caller).
func stateWithTools(t *testing.T, srvURL, dir string, confirm func(string) bool) *replState {
	t.Helper()
	state := replTestState(t, srvURL)
	state.toolsEnabled = true
	state.exec = chattools.NewExecutor(dir, confirm)
	return state
}

func TestRunTurnToolCallLoopExecutesAndSummarizes(t *testing.T) {
	color.NoColor = true
	dir := t.TempDir()
	srv, bodies := toolLoopServer(t,
		[]string{
			toolDeltaChunk(0, "call_1", "function", "write_file", ""),
			toolDeltaChunk(0, "", "", "", `{"path":"hello.txt","content":"hi"}`),
			toolFinishChunk("tool_calls"),
			"[DONE]",
		},
		[]string{streamChunk("created "), streamChunk("hello.txt"), "[DONE]"})
	state := stateWithTools(t, srv.URL, dir, func(string) bool { return true })
	session := NewSession("sys")

	var out strings.Builder
	if err := runTurn(context.Background(), state, session, "write a file for me", &out, true); err != nil {
		t.Fatalf("runTurn: %v", err)
	}

	// The tool ran: the file exists in the sandbox.
	data, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if err != nil {
		t.Fatalf("tool did not write the file: %v", err)
	}
	if string(data) != "hi" {
		t.Errorf("file content = %q, want hi", data)
	}

	// Progress line + streamed final answer rendered.
	if !strings.Contains(out.String(), "▸ tool: write_file") {
		t.Errorf("tool progress line missing: %q", out.String())
	}
	if !strings.Contains(out.String(), "created hello.txt") {
		t.Errorf("final answer not streamed: %q", out.String())
	}

	// Session holds only the user turn and the final answer — no tool
	// messages leak into history.
	raw := session.Raw()
	if len(raw) != 2 {
		t.Fatalf("session has %d messages, want 2 (user + final answer)", len(raw))
	}
	if raw[1].Role != openai.ChatMessageRoleAssistant || raw[1].Content != "created hello.txt" {
		t.Errorf("final session message = %+v", raw[1])
	}

	// The second request must carry the assistant message WITH its tool
	// calls and the tool result keyed to the right ToolCallID.
	if len(*bodies) != 2 {
		t.Fatalf("server received %d requests, want 2", len(*bodies))
	}
	var req openai.ChatCompletionRequest
	if err := json.Unmarshal((*bodies)[1], &req); err != nil {
		t.Fatalf("decode second request: %v", err)
	}
	var sawAssistant, sawTool bool
	for _, m := range req.Messages {
		switch m.Role {
		case openai.ChatMessageRoleAssistant:
			sawAssistant = true
			if len(m.ToolCalls) != 1 || m.ToolCalls[0].ID != "call_1" || m.ToolCalls[0].Function.Name != "write_file" {
				t.Errorf("assistant tool call in request = %+v, want call_1 write_file", m.ToolCalls)
			}
		case openai.ChatMessageRoleTool:
			sawTool = true
			if m.ToolCallID != "call_1" {
				t.Errorf("tool message ToolCallID = %q, want call_1", m.ToolCallID)
			}
			if !strings.Contains(m.Content, "wrote hello.txt") {
				t.Errorf("tool result = %q, want write confirmation", m.Content)
			}
		}
	}
	if !sawAssistant || !sawTool {
		t.Errorf("second request missing assistant/tool messages (assistant=%v tool=%v)", sawAssistant, sawTool)
	}
}

func TestRunTurnDeclinedConfirmFeedsResult(t *testing.T) {
	color.NoColor = true
	dir := t.TempDir()
	srv, bodies := toolLoopServer(t,
		[]string{
			toolDeltaChunk(0, "call_9", "function", "run_shell", ""),
			toolDeltaChunk(0, "", "", "", `{"command":"echo hi"}`),
			toolFinishChunk("tool_calls"),
			"[DONE]",
		},
		[]string{streamChunk("ok"), "[DONE]"})
	state := stateWithTools(t, srv.URL, dir, func(string) bool { return false }) // always decline
	session := NewSession("sys")

	var out strings.Builder
	if err := runTurn(context.Background(), state, session, "run a command", &out, true); err != nil {
		t.Fatalf("runTurn: %v", err)
	}
	if len(*bodies) != 2 {
		t.Fatalf("server received %d requests, want 2", len(*bodies))
	}
	var req openai.ChatCompletionRequest
	if err := json.Unmarshal((*bodies)[1], &req); err != nil {
		t.Fatalf("decode second request: %v", err)
	}
	found := false
	for _, m := range req.Messages {
		if m.Role == openai.ChatMessageRoleTool && m.ToolCallID == "call_9" {
			found = true
			if !strings.Contains(m.Content, "cancelled by user") {
				t.Errorf("tool result = %q, want decline message", m.Content)
			}
		}
	}
	if !found {
		t.Error("second request missing declined tool result")
	}
}

func TestRunTurnRoundCapStops(t *testing.T) {
	color.NoColor = true
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	// The server never finishes: every request returns another tool call.
	loop := []string{
		toolDeltaChunk(0, "call_x", "function", "read_file", ""),
		toolDeltaChunk(0, "", "", "", `{"path":"f.txt"}`),
		toolFinishChunk("tool_calls"),
		"[DONE]",
	}
	srv, bodies := toolLoopServer(t, loop, loop)
	state := stateWithTools(t, srv.URL, dir, func(string) bool { return true })
	session := NewSession("sys")

	var out strings.Builder
	if err := runTurn(context.Background(), state, session, "keep going", &out, true); err != nil {
		t.Fatalf("runTurn: %v", err)
	}
	if len(*bodies) != maxToolRounds {
		t.Errorf("server received %d requests, want the %d-round cap", len(*bodies), maxToolRounds)
	}
	if !strings.Contains(out.String(), "tool-call limit") {
		t.Errorf("round-cap note missing: %q", out.String())
	}
}

func TestRunTurnAskFirstAppendsInstruction(t *testing.T) {
	color.NoColor = true
	srv, bodies := toolLoopServer(t, []string{streamChunk("sure, questions:"), "[DONE]"}, nil)
	state := replTestState(t, srv.URL)
	state.askFirst = true
	session := NewSession("sys")

	var out strings.Builder
	if err := runTurn(context.Background(), state, session, "create an app", &out, true); err != nil {
		t.Fatalf("runTurn: %v", err)
	}
	var req openai.ChatCompletionRequest
	if err := json.Unmarshal((*bodies)[0], &req); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if len(req.Messages) == 0 || req.Messages[0].Role != openai.ChatMessageRoleSystem {
		t.Fatalf("first message = %+v, want system", req.Messages)
	}
	if !strings.Contains(req.Messages[0].Content, "MUST ask up to 3 clarifying questions") {
		t.Errorf("system prompt missing question-first instruction: %q", req.Messages[0].Content)
	}
}

func TestBuildExecutorOneShotAlwaysDeclines(t *testing.T) {
	e := buildExecutor(ReplOptions{OneShot: "create a project", Stdin: strings.NewReader("y\n"), Stdout: io.Discard})
	if e.Confirm("anything") {
		t.Error("OneShot Confirm must always decline")
	}
}

func TestBuildExecutorConfirmReadsStdin(t *testing.T) {
	color.NoColor = true
	e := buildExecutor(ReplOptions{Stdin: strings.NewReader("y\n"), Stdout: io.Discard})
	if !e.Confirm("proceed?") {
		t.Error("Confirm should accept a y reply from Stdin")
	}
	e = buildExecutor(ReplOptions{Stdin: strings.NewReader("n\n"), Stdout: io.Discard})
	if e.Confirm("proceed?") {
		t.Error("Confirm should decline an n reply from Stdin")
	}
}

func TestDispatchToolsToggle(t *testing.T) {
	color.NoColor = true
	st := NewStore(t.TempDir())
	state := replTestState(t, "http://127.0.0.1:1")
	session := NewSession("sys")
	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader(""), Stdout: &out}

	if _, err := dispatchSlash(context.Background(), st, state, session, CmdTools, "off", opts); err != nil {
		t.Fatalf("/tools off: %v", err)
	}
	if state.toolsEnabled {
		t.Error("toolsEnabled still true after /tools off")
	}
	if !strings.Contains(out.String(), "tools: off") {
		t.Errorf("output = %q, want tools: off", out.String())
	}
	if _, err := dispatchSlash(context.Background(), st, state, session, CmdTools, "on", opts); err != nil {
		t.Fatalf("/tools on: %v", err)
	}
	if !state.toolsEnabled {
		t.Error("toolsEnabled false after /tools on")
	}
}

func TestDispatchAskGo(t *testing.T) {
	color.NoColor = true
	st := NewStore(t.TempDir())
	state := replTestState(t, "http://127.0.0.1:1")
	session := NewSession("sys")
	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader(""), Stdout: &out}

	if _, err := dispatchSlash(context.Background(), st, state, session, CmdAsk, "", opts); err != nil {
		t.Fatalf("/ask: %v", err)
	}
	if !state.askFirst {
		t.Error("askFirst false after /ask")
	}
	if _, err := dispatchSlash(context.Background(), st, state, session, CmdGo, "", opts); err != nil {
		t.Fatalf("/go: %v", err)
	}
	if state.askFirst {
		t.Error("askFirst true after /go")
	}
}

func TestDispatchSkillShowAndLoad(t *testing.T) {
	color.NoColor = true
	st := NewStore(t.TempDir())
	state := replTestState(t, "http://127.0.0.1:1")
	session := NewSession("sys")

	// /skill without an arg shows the loaded name + line count.
	var out strings.Builder
	opts := ReplOptions{Stdin: strings.NewReader(""), Stdout: &out}
	if _, err := dispatchSlash(context.Background(), st, state, session, CmdSkill, "", opts); err != nil {
		t.Fatalf("/skill: %v", err)
	}
	if !strings.Contains(out.String(), "skill: embedded SKILL.md (") {
		t.Errorf("skill display missing: %q", out.String())
	}

	// /skill <path> loads an alternate skill into the session.
	dir := t.TempDir()
	alt := filepath.Join(dir, "alt.md")
	if err := os.WriteFile(alt, []byte("alternate skill\nbody\n"), 0644); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	out.Reset()
	if _, err := dispatchSlash(context.Background(), st, state, session, CmdSkill, "alt.md", opts); err != nil {
		t.Fatalf("/skill alt.md: %v", err)
	}
	if session.System != "alternate skill\nbody\n" {
		t.Errorf("session system = %q, want loaded skill", session.System)
	}
	if !strings.Contains(out.String(), "skill loaded from alt.md (3 lines)") {
		t.Errorf("load confirmation missing: %q", out.String())
	}

	// Traversal is rejected with a friendly error.
	out.Reset()
	if _, err := dispatchSlash(context.Background(), st, state, session, CmdSkill, "../etc/passwd", opts); err != nil {
		t.Fatalf("/skill ../etc/passwd: %v", err)
	}
	if !strings.Contains(out.String(), "escapes the current directory") {
		t.Errorf("traversal not rejected: %q", out.String())
	}
}
