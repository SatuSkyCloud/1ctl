package api

import (
	"errors"
	"net/http"
	"testing"
)

// A deployment is already accepted and reconciling when the wait begins, so a
// transport blip must not be reported as a deployment failure. This exact case
// aborted a healthy rollout when the API restarted mid-wait.
func TestTransientWaitErrorsAreRetryable(t *testing.T) {
	for _, message := range []string{
		`Get "http://localhost:8080/v1/deployments/x/status/live": dial tcp [::1]:8080: connect: connection refused`,
		"read tcp 127.0.0.1:1234->127.0.0.1:8080: connection reset by peer",
		"unexpected EOF",
		"dial tcp: lookup api.satusky.com: no such host",
		"Get \"https://api.satusky.com/v1/x\": net/http: request canceled (i/o timeout)",
	} {
		if !isTransientWaitError(errors.New(message)) {
			t.Errorf("isTransientWaitError(%q) = false, want true", message)
		}
	}
}

// A typed HTTP status means the server answered. Its answer is the outcome and
// must never be retried away, or a genuine failure would spin until timeout.
func TestHTTPStatusErrorsAreNotTransient(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusConflict, http.StatusInternalServerError} {
		err := &HTTPStatusError{StatusCode: status}
		if isTransientWaitError(err) {
			t.Errorf("isTransientWaitError(HTTP %d) = true, want false", status)
		}
	}
}

func TestUnknownErrorsAreNotTransient(t *testing.T) {
	if isTransientWaitError(errors.New("deployment configuration is invalid")) {
		t.Fatal("an unrecognised error must not be treated as transient")
	}
	if isTransientWaitError(nil) {
		t.Fatal("nil must not be transient")
	}
}
