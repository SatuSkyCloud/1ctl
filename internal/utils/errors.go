package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CLIError represents an error that can be nicely formatted for CLI output
type CLIError struct {
	Message string
	Err     error
}

func (e *CLIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// NewError creates a new CLIError
func NewError(message string, err error) error {
	return &CLIError{
		Message: message,
		Err:     err,
	}
}

// HandleError prints the error and returns it
func HandleError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr apiDiagnosticError
	if errors.As(err, &apiErr) {
		if IsJSONOutput() {
			printJSONError(errorOutputFromAPIDiagnostic(apiErr))
		} else {
			printAPIDiagnosticError(apiErr)
		}
		return err
	}

	if IsJSONOutput() {
		printJSONError(errorOutput{Error: true, Message: err.Error()})
		return err
	}

	if cliErr, ok := err.(*CLIError); ok {
		PrintError("%s", cliErr.Error())
	} else {
		PrintError("%s", err.Error())
	}
	return err
}

type apiDiagnosticError interface {
	error
	HTTPStatusCode() int
	RetryAfterDuration() time.Duration
	ErrorCode() string
	ErrorDetails() interface{}
	ErrorRetryable() *bool
	ErrorRemediation() []string
	ErrorRequestID() string
}

type errorOutput struct {
	Error        bool        `json:"error"`
	Message      string      `json:"message"`
	StatusCode   *int        `json:"status_code,omitempty"`
	Code         string      `json:"code,omitempty"`
	Details      interface{} `json:"details"`
	RetryAfterMS *int64      `json:"retry_after_ms,omitempty"`
	Retryable    *bool       `json:"retryable,omitempty"`
	Remediation  []string    `json:"remediation"`
	RequestID    string      `json:"request_id,omitempty"`
}

func errorOutputFromAPIDiagnostic(err apiDiagnosticError) errorOutput {
	statusCode := err.HTTPStatusCode()
	output := errorOutput{
		Error:       true,
		Message:     err.Error(),
		StatusCode:  &statusCode,
		Code:        err.ErrorCode(),
		Details:     err.ErrorDetails(),
		Retryable:   err.ErrorRetryable(),
		Remediation: err.ErrorRemediation(),
		RequestID:   err.ErrorRequestID(),
	}
	if retryAfter := err.RetryAfterDuration(); retryAfter > 0 {
		retryAfterMS := retryAfter.Milliseconds()
		output.RetryAfterMS = &retryAfterMS
	}
	return output
}

func printJSONError(output errorOutput) {
	data, marshalErr := json.Marshal(output)
	if marshalErr != nil {
		data = []byte(`{"error":true,"message":"failed to encode error output"}`)
	}
	_, _ = fmt.Fprintln(defaultPrinter.out, string(data)) //nolint:errcheck
}

func printAPIDiagnosticError(err apiDiagnosticError) {
	PrintError("%s", err.Error())
	PrintError("HTTP status: %d", err.HTTPStatusCode())
	if code := err.ErrorCode(); code != "" {
		PrintError("Code: %s", code)
	}
	if details := formatErrorDetails(err.ErrorDetails()); details != "" {
		PrintError("Details: %s", details)
	}
	if retryable := err.ErrorRetryable(); retryable != nil {
		PrintError("Retryable: %t", *retryable)
	}
	if retryAfter := err.RetryAfterDuration(); retryAfter > 0 {
		PrintError("Retry after: %s", retryAfter)
	}
	if remediation := err.ErrorRemediation(); len(remediation) > 0 {
		PrintError("Remediation: %s", strings.Join(remediation, "; "))
	}
	if requestID := err.ErrorRequestID(); requestID != "" {
		PrintError("Request ID: %s", requestID)
	}
}

func formatErrorDetails(details interface{}) string {
	if details == nil {
		return ""
	}
	if message, ok := details.(string); ok {
		return message
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		return ""
	}
	return string(encoded)
}
