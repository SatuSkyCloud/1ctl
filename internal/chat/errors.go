package chat

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// classifyError maps a go-openai error (or a network error) to a friendly,
// actionable message. This is the single error mapper shared by the connect
// flow and the streaming loop — users never see stack traces.
func classifyError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		message := strings.TrimSpace(apiErr.Message)
		switch apiErr.HTTPStatusCode {
		case 401:
			return fmt.Errorf("key rejected (401)")
		case 429:
			return fmt.Errorf("rate limited or out of quota (429)")
		case 404:
			return fmt.Errorf("model not found (404)")
		default:
			if message != "" {
				return fmt.Errorf("provider error (%d): %s", apiErr.HTTPStatusCode, message)
			}
			return fmt.Errorf("provider error (%d)", apiErr.HTTPStatusCode)
		}
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		host := urlErr.URL
		if parsed, parseErr := url.Parse(urlErr.URL); parseErr == nil && parsed.Host != "" {
			host = parsed.Host
		}
		return fmt.Errorf("cannot reach %s: %v", host, urlErr.Err)
	}
	return err
}
