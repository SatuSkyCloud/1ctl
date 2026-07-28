package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

type diagnosticTestError struct {
	message     string
	statusCode  int
	retryAfter  time.Duration
	code        string
	details     interface{}
	retryable   *bool
	remediation []string
	requestID   string
}

func (e *diagnosticTestError) Error() string                     { return e.message }
func (e *diagnosticTestError) HTTPStatusCode() int               { return e.statusCode }
func (e *diagnosticTestError) RetryAfterDuration() time.Duration { return e.retryAfter }
func (e *diagnosticTestError) ErrorCode() string                 { return e.code }
func (e *diagnosticTestError) ErrorDetails() interface{}         { return e.details }
func (e *diagnosticTestError) ErrorRetryable() *bool             { return e.retryable }
func (e *diagnosticTestError) ErrorRemediation() []string        { return e.remediation }
func (e *diagnosticTestError) ErrorRequestID() string            { return e.requestID }

func TestHandleErrorRendersWrappedAPIDiagnostics(t *testing.T) {
	retryable := true
	err := fmt.Errorf("create deployment intent: %w", &diagnosticTestError{
		message:    "rate limit reached",
		statusCode: 429,
		retryAfter: 2 * time.Second,
		code:       "RATE_LIMITED",
		details: map[string]interface{}{
			"limit": 10,
			"scope": "organization",
		},
		retryable:   &retryable,
		remediation: []string{"wait before retrying", "reduce request rate"},
		requestID:   "req-123",
	})

	output := captureErrorOutput(t, "table", err)
	for _, want := range []string{
		"rate limit reached",
		"HTTP status: 429",
		"Code: RATE_LIMITED",
		`Details: {"limit":10,"scope":"organization"}`,
		"Retryable: true",
		"Retry after: 2s",
		"Remediation: wait before retrying; reduce request rate",
		"Request ID: req-123",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("table error output missing %q: %q", want, output)
		}
	}
}

func TestHandleErrorJSONIsOneStructuredDocument(t *testing.T) {
	retryable := false
	err := fmt.Errorf("create deployment intent: %w", &diagnosticTestError{
		message:     "invalid deployment",
		statusCode:  422,
		code:        "VALIDATION_ERROR",
		details:     "app_label is required",
		retryable:   &retryable,
		remediation: []string{"set --app"},
		requestID:   "req-456",
	})

	output := captureErrorOutput(t, "json", err)
	var got map[string]interface{}
	if decodeErr := json.Unmarshal([]byte(output), &got); decodeErr != nil {
		t.Fatalf("JSON error output is invalid: %q: %v", output, decodeErr)
	}
	if got["error"] != true || got["message"] != "invalid deployment" || got["code"] != "VALIDATION_ERROR" {
		t.Fatalf("JSON error output = %#v", got)
	}
	if got["status_code"] != float64(422) || got["details"] != "app_label is required" || got["retryable"] != false || got["request_id"] != "req-456" {
		t.Fatalf("JSON diagnostics = %#v", got)
	}
	if remediation, ok := got["remediation"].([]interface{}); !ok || len(remediation) != 1 || remediation[0] != "set --app" {
		t.Fatalf("JSON remediation = %#v", got["remediation"])
	}
}

func TestHandleErrorJSONForGenericFailureIsNonemptyObject(t *testing.T) {
	output := captureErrorOutput(t, "json", fmt.Errorf("plain failure"))
	var got map[string]interface{}
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("JSON error output is invalid: %q: %v", output, err)
	}
	if got["error"] != true || got["message"] != "plain failure" {
		t.Fatalf("JSON error output = %#v", got)
	}
}

func TestHandleErrorRendersLocalDiagnosticWithoutHTTPStatus(t *testing.T) {
	err := NewLocalDiagnosticError(
		"deployment is unavailable for \"wordpress\"",
		"PACKAGE_TRUST_INVALID",
		[]string{"Ask an authorized operator to re-sign the package before deploying."},
	)

	tableOutput := captureErrorOutput(t, "table", err)
	for _, want := range []string{"deployment is unavailable", "Code: PACKAGE_TRUST_INVALID", "Remediation: Ask an authorized operator"} {
		if !strings.Contains(tableOutput, want) {
			t.Fatalf("table error output missing %q: %q", want, tableOutput)
		}
	}
	if strings.Contains(tableOutput, "HTTP status") {
		t.Fatalf("local diagnostic included HTTP status: %q", tableOutput)
	}

	jsonOutput := captureErrorOutput(t, "json", err)
	var got map[string]interface{}
	if decodeErr := json.Unmarshal([]byte(jsonOutput), &got); decodeErr != nil {
		t.Fatalf("JSON error output is invalid: %q: %v", jsonOutput, decodeErr)
	}
	if got["code"] != "PACKAGE_TRUST_INVALID" || got["message"] != "deployment is unavailable for \"wordpress\"" {
		t.Fatalf("JSON error output = %#v", got)
	}
	if _, ok := got["status_code"]; ok {
		t.Fatalf("local diagnostic status_code = %#v, want omitted", got["status_code"])
	}
}

func TestHandleErrorJSONPreservesEmptyBackendDiagnostics(t *testing.T) {
	retryable := false
	err := &diagnosticTestError{
		message:     "invalid deployment",
		statusCode:  422,
		code:        "VALIDATION_ERROR",
		details:     map[string]interface{}{},
		retryable:   &retryable,
		remediation: []string{},
		requestID:   "req-empty",
	}

	output := captureErrorOutput(t, "json", err)
	var got map[string]interface{}
	if decodeErr := json.Unmarshal([]byte(output), &got); decodeErr != nil {
		t.Fatalf("JSON error output is invalid: %q: %v", output, decodeErr)
	}
	if details, ok := got["details"].(map[string]interface{}); !ok || len(details) != 0 {
		t.Fatalf("details = %#v, want empty object", got["details"])
	}
	if remediation, ok := got["remediation"].([]interface{}); !ok || len(remediation) != 0 {
		t.Fatalf("remediation = %#v, want empty array", got["remediation"])
	}
}

func captureErrorOutput(t *testing.T, format string, err error) string {
	t.Helper()
	originalPrinter := defaultPrinter
	originalFormat := outputFormat
	buffer := &bytes.Buffer{}
	defaultPrinter = NewPrinter(buffer)
	SetOutputFormat(format)
	t.Cleanup(func() {
		defaultPrinter = originalPrinter
		SetOutputFormat(originalFormat)
	})

	if handled := HandleError(err); handled != err {
		t.Fatalf("HandleError() = %v, want original error", handled)
	}
	return buffer.String()
}
