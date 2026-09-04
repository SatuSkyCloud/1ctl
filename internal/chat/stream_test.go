package chat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

// fakeSSEServer streams SSE events. Each entry in events is wrapped as
// "data: <entry>\n\n"; pass "[DONE]" for the terminal event.
func fakeSSEServer(t *testing.T, events []string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	var body strings.Builder
	for _, e := range events {
		body.WriteString("data: " + e + "\n\n")
	}
	var captured capturedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		captured = capturedRequest{Path: r.URL.Path, AuthHeader: r.Header.Get("Authorization"), Body: raw}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body.String()))
	}))
	t.Cleanup(srv.Close)
	return srv, &captured
}

// newRawSSEServer serves a prebuilt SSE body verbatim.
func newRawSSEServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func streamChunk(content string) string {
	return `{"id":"x","object":"chat.completion.chunk","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":"` + content + `"},"finish_reason":null}]}`
}

// usageChunk is a stream chunk carrying only usage (the final chunk in a
// stream with include_usage=true).
func usageChunk(promptTokens, completionTokens int) string {
	return `{"id":"x","object":"chat.completion.chunk","created":1,"model":"gpt-4o-mini","choices":[],"usage":{"prompt_tokens":` + strconv.Itoa(promptTokens) + `,"completion_tokens":` + strconv.Itoa(completionTokens) + `,"total_tokens":` + strconv.Itoa(promptTokens+completionTokens) + `}}`
}

func TestStreamCompletionAssemblesInOrder(t *testing.T) {
	events := []string{streamChunk("Wor"), streamChunk("ld"), streamChunk("!"), streamChunk(" Hello"), "[DONE]"}
	srv, captured := fakeSSEServer(t, events)
	client := NewClientWithBaseURL("sk-test-123", srv.URL)

	var out strings.Builder
	res, err := StreamCompletion(context.Background(), client, "gpt-4o-mini",
		[]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "hi"}}, nil, &out)
	if err != nil {
		t.Fatalf("StreamCompletion: %v", err)
	}
	if out.String() != "World! Hello" {
		t.Errorf("streamed output = %q, want %q", out.String(), "World! Hello")
	}
	if res.Text != "World! Hello" {
		t.Errorf("result.Text = %q, want %q", res.Text, "World! Hello")
	}
	if res.Model != "gpt-4o-mini" {
		t.Errorf("result.Model = %q, want gpt-4o-mini", res.Model)
	}

	// The request must set stream + include_usage.
	var req openai.ChatCompletionRequest
	if err := json.Unmarshal(captured.Body, &req); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if !req.Stream {
		t.Error("request Stream = false, want true")
	}
	if req.StreamOptions == nil || !req.StreamOptions.IncludeUsage {
		t.Errorf("request StreamOptions = %+v, want IncludeUsage=true", req.StreamOptions)
	}
	if captured.AuthHeader != "Bearer sk-test-123" {
		t.Errorf("Authorization = %q", captured.AuthHeader)
	}
	if len(req.Tools) != 0 {
		t.Errorf("request Tools = %+v, want none when not provided", req.Tools)
	}
}

// jsonStr renders s as a JSON string literal (quoted + escaped), for
// embedding raw values into hand-built SSE payloads.
func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// toolDeltaChunk is a stream chunk carrying one tool-call fragment for the
// call at index idx. Empty fields are omitted, matching how providers
// stream deltas (id/type/name on the first fragment, arguments later).
func toolDeltaChunk(idx int, id, typ, name, args string) string {
	fields := ""
	if id != "" {
		fields += `,"id":` + jsonStr(id)
	}
	if typ != "" {
		fields += `,"type":` + jsonStr(typ)
	}
	fn := ""
	if name != "" {
		fn += `"name":` + jsonStr(name)
	}
	if args != "" {
		if fn != "" {
			fn += ","
		}
		fn += `"arguments":` + jsonStr(args)
	}
	tc := `{"index":` + strconv.Itoa(idx) + fields
	if fn != "" {
		tc += `,"function":{` + fn + `}`
	}
	tc += `}`
	return `{"id":"x","object":"chat.completion.chunk","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"tool_calls":[` + tc + `]},"finish_reason":null}]}`
}

// toolFinishChunk is the final delta of a tool-call stream.
func toolFinishChunk(reason string) string {
	return `{"id":"x","object":"chat.completion.chunk","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{},"finish_reason":` + jsonStr(reason) + `}]}`
}

func TestStreamCompletionAssemblesToolCalls(t *testing.T) {
	// Arguments arrive split across fragments (as real providers stream
	// them); id/type/name only on the first fragment.
	events := []string{
		toolDeltaChunk(0, "call_abc123", "function", "read_file", ""),
		toolDeltaChunk(0, "", "", "", `{"path":`),
		toolDeltaChunk(0, "", "", "", `"main.go"}`),
		toolFinishChunk("tool_calls"),
		"[DONE]",
	}
	srv, _ := fakeSSEServer(t, events)
	client := NewClientWithBaseURL("sk-test-123", srv.URL)

	var out strings.Builder
	res, err := StreamCompletion(context.Background(), client, "gpt-4o-mini", nil, toolsForTest(), &out)
	if err != nil {
		t.Fatalf("StreamCompletion: %v", err)
	}
	if len(res.ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %+v, want 1 assembled call", res.ToolCalls)
	}
	call := res.ToolCalls[0]
	if call.ID != "call_abc123" {
		t.Errorf("call.ID = %q", call.ID)
	}
	if string(call.Type) != "function" {
		t.Errorf("call.Type = %q", call.Type)
	}
	if call.Function.Name != "read_file" {
		t.Errorf("call.Function.Name = %q", call.Function.Name)
	}
	if call.Function.Arguments != `{"path":"main.go"}` {
		t.Errorf("call.Function.Arguments = %q", call.Function.Arguments)
	}
	if res.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want tool_calls", res.FinishReason)
	}
	// Tool-call deltas must never leak to the streamed output.
	if out.String() != "" {
		t.Errorf("stray output written: %q", out.String())
	}
	// The assembled call must not carry a stream index.
	if call.Index != nil {
		t.Errorf("assembled call keeps stream index %d", *call.Index)
	}
}

func TestStreamCompletionMultipleToolCallsInOrder(t *testing.T) {
	events := []string{
		toolDeltaChunk(0, "call_0", "function", "list_dir", `{"path":"."}`),
		toolDeltaChunk(1, "call_1", "function", "read_file", `{"path":"a.go"}`),
		toolFinishChunk("tool_calls"),
		"[DONE]",
	}
	srv, _ := fakeSSEServer(t, events)
	client := NewClientWithBaseURL("sk-test-123", srv.URL)

	res, err := StreamCompletion(context.Background(), client, "gpt-4o-mini", nil, toolsForTest(), io.Discard)
	if err != nil {
		t.Fatalf("StreamCompletion: %v", err)
	}
	if len(res.ToolCalls) != 2 {
		t.Fatalf("ToolCalls = %+v, want 2", res.ToolCalls)
	}
	if res.ToolCalls[0].Function.Name != "list_dir" || res.ToolCalls[1].Function.Name != "read_file" {
		t.Errorf("tool calls out of order: %+v", res.ToolCalls)
	}
}

// toolsForTest is a minimal non-empty tool set for request-building tests.
func toolsForTest() []openai.Tool {
	return []openai.Tool{{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name: "read_file", Parameters: map[string]any{"type": "object"},
		},
	}}
}

func TestStreamCompletionUsageParsed(t *testing.T) {
	usage := `{"id":"x","object":"chat.completion.chunk","created":1,"model":"gpt-4o-mini","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	events := []string{streamChunk("hi "), usage, "[DONE]"}
	srv, _ := fakeSSEServer(t, events)
	client := NewClientWithBaseURL("sk-test-123", srv.URL)

	res, err := StreamCompletion(context.Background(), client, "gpt-4o-mini", nil, nil, io.Discard)
	if err != nil {
		t.Fatalf("StreamCompletion: %v", err)
	}
	if res.TotalTokens != 15 {
		t.Errorf("result.TotalTokens = %d, want 15", res.TotalTokens)
	}
	if res.PromptTokens != 10 {
		t.Errorf("result.PromptTokens = %d, want 10", res.PromptTokens)
	}
	if res.Text != "hi " {
		t.Errorf("result.Text = %q, want %q", res.Text, "hi ")
	}
}

func TestStreamCompletionMidStreamError(t *testing.T) {
	// go-openai detects mid-stream errors from raw JSON lines (no "data:"
	// prefix) accumulated until EOF, so this test uses its own handler.
	var body strings.Builder
	body.WriteString("data: " + streamChunk("Par") + "\n\n")
	body.WriteString(`{"error":{"message":"stream interrupted","type":"server_error"}}` + "\n\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body.String()))
	}))
	defer srv.Close()
	client := NewClientWithBaseURL("sk-test-123", srv.URL)

	var out strings.Builder
	res, err := StreamCompletion(context.Background(), client, "gpt-4o-mini", nil, nil, &out)
	if err == nil {
		t.Fatal("expected mid-stream error, got nil")
	}
	if !strings.Contains(err.Error(), "stream interrupted") {
		t.Errorf("error = %q, want it to mention the provider message", err.Error())
	}
	// Partial text streamed before the error must be preserved.
	if res.Text != "Par" {
		t.Errorf("partial text = %q, want %q", res.Text, "Par")
	}
}

func TestStreamCompletionHTTPErrorBeforeStream(t *testing.T) {
	// A non-2xx response with an API error body surfaces at
	// CreateChatCompletionStream time and must be classified.
	srv, _ := fakeChatServer(t, http.StatusUnauthorized, apiErrorBody("bad key"))
	client := NewClientWithBaseURL("sk-test-123", srv.URL)
	_, err := StreamCompletion(context.Background(), client, "gpt-4o-mini", nil, nil, io.Discard)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "key rejected (401)") {
		t.Errorf("error = %q, want classified 401 message", err.Error())
	}
}

func TestRunCompletion(t *testing.T) {
	body := `{"id":"x","object":"chat.completion","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`
	srv, _ := fakeChatServer(t, http.StatusOK, body)
	client := NewClientWithBaseURL("sk-test-123", srv.URL)

	var out strings.Builder
	res, err := RunCompletion(context.Background(), client, "gpt-4o-mini",
		[]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "ping"}}, &out)
	if err != nil {
		t.Fatalf("RunCompletion: %v", err)
	}
	if res.Text != "pong" || out.String() != "pong" {
		t.Errorf("text = %q, out = %q, want pong", res.Text, out.String())
	}
	if res.TotalTokens != 4 {
		t.Errorf("result.TotalTokens = %d, want 4", res.TotalTokens)
	}
	if res.Model != "gpt-4o-mini" {
		t.Errorf("result.Model = %q", res.Model)
	}
}

func TestRunCompletionErrorClassified(t *testing.T) {
	srv, _ := fakeChatServer(t, http.StatusNotFound, apiErrorBody("model gone"))
	client := NewClientWithBaseURL("sk-test-123", srv.URL)
	_, err := RunCompletion(context.Background(), client, "old-model", nil, io.Discard)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "model not found (404)") {
		t.Errorf("error = %q, want classified 404 message", err.Error())
	}
}
