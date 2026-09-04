package chat

import (
	"context"
	"errors"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// retryBackoffs is the delay between retry attempts after a transient
// provider error (HTTP 429 or 5xx): 500ms, 1s, 2s. Package-level so tests
// can shrink it; production code never mutates it.
var retryBackoffs = []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second}

// withRetry runs attempt, retrying transient provider errors (429 or 5xx)
// up to len(retryBackoffs) times with the default backoff schedule.
// Non-transient errors (401, 404, 400, network failures) fail immediately;
// the context cancels any in-flight retry sleep.
func withRetry(ctx context.Context, attempt func() error) error {
	return withRetryBackoff(ctx, attempt, retryBackoffs)
}

// withRetryBackoff is withRetry with an explicit backoff schedule. attempt
// is invoked once, then once more per backoff entry while it keeps failing
// with a retryable error. The last error is returned when the schedule is
// exhausted.
func withRetryBackoff(ctx context.Context, attempt func() error, backoffs []time.Duration) error {
	var lastErr error
	for i := 0; ; i++ {
		err := attempt()
		if err == nil {
			return nil
		}
		if !isRetryable(err) {
			return err
		}
		lastErr = err
		if i >= len(backoffs) {
			return lastErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoffs[i]):
		}
	}
}

// isRetryable reports whether a provider error is transient (429 rate
// limit or a 5xx server error). Anything else — auth, missing model, bad
// request, network — is failed immediately.
func isRetryable(err error) bool {
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		return apiErr.HTTPStatusCode == 429 || apiErr.HTTPStatusCode >= 500
	}
	return false
}
