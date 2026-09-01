package chat

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

// capturedRequest records what the fake server received so tests can
// assert path/auth/body wiring.
type capturedRequest struct {
	Path        string
	AuthHeader  string
	Body        []byte
	ContentType string
}

// fakeChatServer spins up an OpenAI-compatible chat completions endpoint.
// It records the last request so tests can assert wiring, and replies with
// status + body.
func fakeChatServer(t *testing.T, status int, body string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	var captured capturedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		captured = capturedRequest{
			Path:        r.URL.Path,
			AuthHeader:  r.Header.Get("Authorization"),
			Body:        raw,
			ContentType: r.Header.Get("Content-Type"),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &captured
}

func apiErrorBody(message string) string {
	payload, _ := json.Marshal(map[string]any{
		"error": map[string]any{"message": message, "type": "invalid_request_error"},
	})
	return string(payload)
}

func TestConnectSuccess(t *testing.T) {
	srv, captured := fakeChatServer(t, http.StatusOK, `{"id":"x","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}]}`)
	client := NewClientWithBaseURL("sk-test-123", srv.URL)
	p, _ := ParseProvider("openai")

	if err := Connect(context.Background(), client, p); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if !strings.HasSuffix(captured.Path, "/chat/completions") {
		t.Errorf("request path = %q, want suffix /chat/completions", captured.Path)
	}
	if captured.AuthHeader != "Bearer sk-test-123" {
		t.Errorf("Authorization = %q, want Bearer sk-test-123", captured.AuthHeader)
	}
	var req openai.ChatCompletionRequest
	if err := json.Unmarshal(captured.Body, &req); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if req.Model != p.DefaultModel {
		t.Errorf("request model = %q, want %q", req.Model, p.DefaultModel)
	}
	if req.MaxTokens != 5 {
		t.Errorf("request MaxTokens = %d, want 5", req.MaxTokens)
	}
	if len(req.Messages) != 1 || req.Messages[0].Content != "ping" {
		t.Errorf("request messages = %+v, want a single user \"ping\"", req.Messages)
	}
}

func TestConnectBaseURLWiring(t *testing.T) {
	srv, captured := fakeChatServer(t, http.StatusOK, `{"id":"x","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"pong"}}]}`)
	// NewClientWithBaseURL must point at the injected URL, not the
	// provider default (api.openai.com), so the fake server is hit.
	client := NewClientWithBaseURL("sk-test-123", srv.URL)
	p, _ := ParseProvider("openai")
	if err := Connect(context.Background(), client, p); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if !strings.HasSuffix(captured.Path, "/chat/completions") {
		t.Errorf("request path = %q, want suffix /chat/completions", captured.Path)
	}
}

func TestConnectErrorClassification(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{name: "unauthorized 401", status: 401, body: apiErrorBody("Incorrect API key provided"), wantErr: "key rejected (401)"},
		{name: "rate limited 429", status: 429, body: apiErrorBody("Rate limit reached"), wantErr: "rate limited or out of quota (429)"},
		{name: "model missing 404", status: 404, body: apiErrorBody("The model does not exist"), wantErr: "model not found (404)"},
		{name: "server error 500", status: 500, body: apiErrorBody("internal server error"), wantErr: "provider error (500): internal server error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := fakeChatServer(t, tt.status, tt.body)
			client := NewClientWithBaseURL("sk-test-123", srv.URL)
			p, _ := ParseProvider("openai")
			err := Connect(context.Background(), client, p)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConnectNetworkRefused(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close() // free the port so the connect is refused

	client := NewClientWithBaseURL("sk-test-123", "http://"+addr)
	p, _ := ParseProvider("openai")
	err = Connect(context.Background(), client, p)
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot reach") {
		t.Errorf("error = %q, want friendly \"cannot reach <host>\" message", err.Error())
	}
	if !strings.Contains(err.Error(), addr) {
		t.Errorf("error = %q, want host %q mentioned", err.Error(), addr)
	}
}
