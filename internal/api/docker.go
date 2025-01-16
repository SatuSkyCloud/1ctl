package api

import (
	"1ctl/internal/config"
	"1ctl/internal/context"
	"1ctl/internal/docker"
	"1ctl/internal/utils"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func validateImagePath(path string) error {
	// Clean and validate the path
	cleanPath := filepath.Clean(path)

	// Check for directory traversal
	if strings.Contains(cleanPath, "..") {
		return utils.NewError("invalid path: must not contain parent directory references", nil)
	}

	// Verify the file exists
	if _, err := os.Stat(cleanPath); err != nil {
		return utils.NewError("image file not found: %w", err)
	}

	return nil
}

func UploadDockerImage(imagePath, projectName string) (string, error) {
	if err := validateImagePath(imagePath); err != nil {
		return "", utils.NewError("invalid image path: %w", err)
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return "", utils.NewError("failed to open image file: %w", err)
	}
	defer file.Close()

	config := config.GetConfig()
	url := fmt.Sprintf("%s/docker/images/upload", config.ApiURL)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add image file
	part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		return "", utils.NewError("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", utils.NewError("failed to copy image to form: %w", err)
	}

	// Add image tag
	if err := writer.WriteField("tag", projectName); err != nil {
		return "", utils.NewError("failed to add tag to form: %w", err)
	}

	// Add version control (currently using GenerateImageHash)
	version, err := docker.GenerateImageHash()
	if err != nil {
		return "", utils.NewError("failed to generate image hash: %w", err)
	}

	if err := writer.WriteField("version", version); err != nil {
		return "", utils.NewError("failed to add version control to form: %w", err)
	}

	writer.Close()

	// Create request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", utils.NewError("failed to create request: %w", err)
	}

	token := context.GetToken()
	if token == "" {
		return "", utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate", nil)
	}

	userConfigKey := context.GetUserConfigKey()
	if userConfigKey == "" {
		return "", utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate", nil)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-satusky-api-key", token)
	req.Header.Set("x-satusky-config", userConfigKey)

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", utils.NewError("failed to deploy Docker image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", utils.NewError(fmt.Sprintf("failed to deploy image: server returned %d", resp.StatusCode), nil)
	}

	return version, nil
}
