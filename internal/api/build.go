package api

import (
	"1ctl/internal/config"
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BuildResponse is returned by POST /v1/cli/builds.
type BuildResponse struct {
	BuildID string `json:"build_id"`
	Status  string `json:"status"`
}

// BuildStatusResponse is returned by GET /v1/cli/builds/{id}/status.
type BuildStatusResponse struct {
	BuildID      string `json:"build_id"`
	Status       string `json:"status"`        // queued | building | completed | failed
	ImageRef     string `json:"image_ref"`     // populated when Status == "completed"
	Logs         string `json:"logs"`          // accumulated Kaniko stdout
	ErrorMessage string `json:"error_message"` // populated when Status == "failed"
}

// SubmitBuild uploads the gzipped build context to the backend and returns the
// build ID. The backend spawns a Kaniko job that builds the image and pushes it
// to the internal registry.
func SubmitBuild(contextTarPath, projectName, dockerfilePath string, buildArgs map[string]string) (string, error) {
	token := context.GetToken()
	if token == "" {
		return "", utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate", nil)
	}

	f, err := os.Open(contextTarPath) // #nosec G304 -- caller-supplied temp file from PackageContext
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to open context archive: %s", err.Error()), nil)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	part, err := w.CreateFormFile("context", filepath.Base(contextTarPath))
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to create form file: %s", err.Error()), nil)
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to write context to form: %s", err.Error()), nil)
	}

	for _, pair := range []struct{ key, val string }{
		{"project", projectName},
		{"dockerfile", dockerfilePath},
	} {
		if err := w.WriteField(pair.key, pair.val); err != nil {
			return "", utils.NewError(fmt.Sprintf("failed to write %s field: %s", pair.key, err.Error()), nil)
		}
	}
	for k, v := range buildArgs {
		if err := w.WriteField("build_arg", k+"="+v); err != nil {
			return "", utils.NewError(fmt.Sprintf("failed to write build_arg: %s", err.Error()), nil)
		}
	}
	if err := w.Close(); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to close multipart writer: %s", err.Error()), nil)
	}

	cfg := config.GetConfig()
	req, err := http.NewRequest("POST", cfg.ApiURL+"/builds", body)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to create request: %s", err.Error()), nil)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("x-satusky-api-key", token)
	if email := context.GetEmail(); email != "" {
		req.Header.Set("x-satusky-user-email", email)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to submit build: %s", err.Error()), nil)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var apiErr APIError
		if jsonErr := json.Unmarshal(respBody, &apiErr); jsonErr == nil && apiErr.Message != "" {
			return "", utils.NewError(fmt.Sprintf("build submission failed: %s", apiErr.Message), nil)
		}
		return "", utils.NewError(fmt.Sprintf("build submission failed (HTTP %d): %s", resp.StatusCode, string(respBody)), nil)
	}

	var apiResp struct {
		Error bool          `json:"error"`
		Data  BuildResponse `json:"data"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to parse build response: %s", err.Error()), nil)
	}
	if apiResp.Error {
		return "", utils.NewError("build submission rejected by server", nil)
	}
	return apiResp.Data.BuildID, nil
}

// GetBuildStatus returns the current build status and any accumulated logs.
func GetBuildStatus(buildID string) (*BuildStatusResponse, error) {
	var resp struct {
		Error bool                `json:"error"`
		Data  BuildStatusResponse `json:"data"`
	}
	if err := makeRequest("GET", fmt.Sprintf("/builds/%s/status", buildID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// WaitForBuild polls until the build completes or fails, streaming any new log
// lines to progressWriter. Returns the full image reference on success.
func WaitForBuild(buildID string, progressWriter io.Writer) (string, error) {
	const (
		pollInterval = 3 * time.Second
		maxWait      = 15 * time.Minute
	)

	deadline := time.Now().Add(maxWait)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var logOffset int // byte offset into the accumulated Logs string

	for time.Now().Before(deadline) {
		<-ticker.C

		status, err := GetBuildStatus(buildID)
		if err != nil {
			// Transient network error — keep polling.
			continue
		}

		// Print any new log lines since the last poll.
		if len(status.Logs) > logOffset {
			newLogs := status.Logs[logOffset:]
			scanner := bufio.NewScanner(strings.NewReader(newLogs))
			for scanner.Scan() {
				fmt.Fprintf(progressWriter, "  %s\n", scanner.Text())
			}
			logOffset = len(status.Logs)
		}

		switch status.Status {
		case "completed":
			return status.ImageRef, nil
		case "failed":
			msg := status.ErrorMessage
			if msg == "" {
				msg = "unknown error"
			}
			return "", utils.NewError(fmt.Sprintf("cloud build failed: %s", msg), nil)
		}
		// queued | building → keep polling
	}

	return "", utils.NewError(fmt.Sprintf("cloud build timed out after %v", maxWait), nil)
}
