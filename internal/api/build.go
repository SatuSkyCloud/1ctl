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

// buildUploadClient is a dedicated client for build context uploads.
// Uses a longer timeout than the shared API client because build archives
// can be large and the upload may take time on slow connections.
var buildUploadClient = &http.Client{
	Timeout: 5 * time.Minute,
}

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
	ImageArch    string `json:"image_arch"`    // "amd64", "arm64", or "" if detection failed
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
	apiURL := cfg.ApiURL + "/builds"

	// Enforce HTTPS for non-localhost API URLs to prevent token leakage
	if !utils.IsLocalhostURL(apiURL) && !strings.HasPrefix(apiURL, "https://") {
		return "", utils.NewError(fmt.Sprintf("refusing to send auth token over insecure connection (%s). Use HTTPS or http://localhost for local development", cfg.ApiURL), nil)
	}

	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to create request: %s", err.Error()), nil)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("x-satusky-api-key", token)
	if email := context.GetEmail(); email != "" {
		req.Header.Set("x-satusky-user-email", email)
	}

	resp, err := buildUploadClient.Do(req)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to submit build: %s", err.Error()), nil)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to read build response: %s", err.Error()), nil)
	}
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

// BuildResult holds the outcome of a completed cloud build.
type BuildResult struct {
	ImageRef  string
	ImageArch string // "amd64", "arm64", or "" if detection failed
}

// WaitForBuildResult polls the cloud-build job until completion, streaming
// new log lines to progressWriter, and returns the BuildResult (image ref +
// detected image architecture).
func WaitForBuildResult(buildID string, progressWriter io.Writer) (*BuildResult, error) {
	const (
		pollInterval = 3 * time.Second
		maxWait      = 15 * time.Minute
	)

	deadline := time.Now().Add(maxWait)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var logOffset int

	for time.Now().Before(deadline) {
		<-ticker.C

		status, err := GetBuildStatus(buildID)
		if err != nil {
			continue
		}

		if len(status.Logs) > logOffset {
			newLogs := status.Logs[logOffset:]
			scanner := bufio.NewScanner(strings.NewReader(newLogs))
			for scanner.Scan() {
				if _, err := fmt.Fprintf(progressWriter, "  %s\n", scanner.Text()); err != nil {
					return nil, utils.NewError(fmt.Sprintf("failed to write build log: %s", err.Error()), nil)
				}
			}
			logOffset = len(status.Logs)
		}

		switch status.Status {
		case "completed":
			return &BuildResult{ImageRef: status.ImageRef, ImageArch: status.ImageArch}, nil
		case "failed":
			msg := status.ErrorMessage
			if msg == "" {
				msg = "unknown error"
			}
			return nil, utils.NewError(fmt.Sprintf("cloud build failed: %s", msg), nil)
		}
	}

	return nil, utils.NewError(fmt.Sprintf("cloud build timed out after %v", maxWait), nil)
}
