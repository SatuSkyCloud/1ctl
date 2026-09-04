package chat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// countingServer returns an httptest server whose handler fails with the
// given status (an OpenAI-style error body) for the first failTimes
// requests and then serves the okBody. It counts requests.
func countingServer(failStatus, failTimes int, okBody string) (*httptest.Server, *int) {
	var count int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		n := count
		mu.Unlock()
		if n <= failTimes {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(failStatus)
			_, _ = w.Write([]byte(apiErrorBody("transient failure")))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(okBody))
	}))
	return srv, &count
}

// shortBackoff shrinks the retry schedule for the duration of the test so
// retry assertions run without real sleeps.
func shortBackoff(t *testing.T) {
	t.Helper()
	old := retryBackoffs
	retryBackoffs = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { retryBackoffs = old })
}

func TestStreamCompletionRetriesTransientErrors(t *testing.T) {
	shortBackoff(t)
	okBody := "data: " + streamChunk("ok") + "\n\ndata: [DONE]\n\n"
	srv, count := countingServer(http.StatusTooManyRequests, 2, okBody)
	defer srv.Close()
	client := NewClientWithBaseURL("sk-test-123", srv.URL)

	res, err := StreamCompletion(context.Background(), client, "gpt-4o-mini", nil, nil, io.Discard)
	if err != nil {
		t.Fatalf("StreamCompletion: %v", err)
	}
	if res.Text != "ok" {
		t.Errorf("text = %q, want %q", res.Text, "ok")
	}
	if *count != 3 {
		t.Errorf("request count = %d, want 3 (2 transient failures + 1 success)", *count)
	}
}

func TestStreamCompletionRetriesServerError(t *testing.T) {
	shortBackoff(t)
	okBody := "data: " + streamChunk("ok") + "\n\ndata: [DONE]\n\n"
	srv, count := countingServer(http.StatusInternalServerError, 1, okBody)
	defer srv.Close()
	client := NewClientWithBaseURL("sk-test-123", srv.URL)

	if _, err := StreamCompletion(context.Background(), client, "gpt-4o-mini", nil, nil, io.Discard); err != nil {
		t.Fatalf("StreamCompletion after 5xx retry: %v", err)
	}
	if *count != 2 {
		t.Errorf("request count = %d, want 2", *count)
	}
}

func TestStreamCompletionNoRetryOnAuthError(t *testing.T) {
	shortBackoff(t)
	srv, count := countingServer(http.StatusUnauthorized, 999, "")
	defer srv.Close()
	client := NewClientWithBaseURL("sk-test-123", srv.URL)

	_, err := StreamCompletion(context.Background(), client, "gpt-4o-mini", nil, nil, io.Discard)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "key rejected (401)") {
		t.Errorf("error = %q, want classified 401 message", err.Error())
	}
	if *count != 1 {
		t.Errorf("request count = %d, want 1 (401 must not retry)", *count)
	}
}

func TestWithRetryGivesUpAfterBackoffsExhausted(t *testing.T) {
	shortBackoff(t)
	var attempts int
	err := withRetry(context.Background(), func() error {
		attempts++
		return &openai.APIError{HTTPStatusCode: http.StatusTooManyRequests}
	})
	if err == nil {
		t.Fatal("expected error after backoffs exhausted, got nil")
	}
	if attempts != 4 {
		t.Errorf("attempts = %d, want 4 (1 initial + 3 retries)", attempts)
	}
}

func TestWithRetryContextCancelStopsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: the sleep must not block
	start := time.Now()
	err := withRetryBackoff(ctx, func() error {
		return &openai.APIError{HTTPStatusCode: http.StatusTooManyRequests}
	}, []time.Duration{time.Hour})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("retry sleep blocked for %v after context cancel", elapsed)
	}
}

func TestWithRetryNonRetryableFailsImmediately(t *testing.T) {
	var attempts int
	err := withRetry(context.Background(), func() error {
		attempts++
		return &openai.APIError{HTTPStatusCode: http.StatusNotFound}
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}
