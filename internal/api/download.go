package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"1ctl/internal/context"
	"1ctl/internal/utils"
)

func downloadMainAPIResponse(path string, headers http.Header) ([]byte, error) {
	requestURL := resolveMainAPIURL(path)
	if !utils.IsLocalhostURL(requestURL) && !strings.HasPrefix(requestURL, "https://") {
		return nil, utils.NewError(
			fmt.Sprintf("refusing to send auth token over insecure connection (%s). Use HTTPS or http://localhost for local development", requestURL),
			nil,
		)
	}

	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to create request: %s", err.Error()), nil)
	}
	token := context.GetToken()
	if token == "" {
		return nil, utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate", nil)
	}
	request.Header.Set("x-satusky-api-key", token)
	if email := context.GetEmail(); email != "" {
		request.Header.Set("x-satusky-user-email", email)
	}
	for key, values := range headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to make request: %s", err.Error()), nil)
	}
	defer func() { _ = response.Body.Close() }() //nolint:errcheck

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to read response body: %s", err.Error()), nil)
	}
	if response.StatusCode >= http.StatusBadRequest {
		var apiErr APIError
		if decodeErr := json.Unmarshal(body, &apiErr); decodeErr == nil && apiErr.Message != "" {
			return nil, &HTTPStatusError{StatusCode: response.StatusCode, Message: apiErr.Message}
		}
		return nil, &HTTPStatusError{
			StatusCode: response.StatusCode,
			Message:    fmt.Sprintf("request failed with status %d", response.StatusCode),
		}
	}
	return body, nil
}
