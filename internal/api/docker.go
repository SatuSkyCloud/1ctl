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
		return utils.NewError(fmt.Sprintf("image file not found: %s", err.Error()), nil)
	}

	return nil
}

func UploadDockerImage(imagePath, projectName string) (string, error) {
	if err := validateImagePath(imagePath); err != nil {
		return "", utils.NewError(fmt.Sprintf("invalid image path: %s", err.Error()), nil)
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to open image file: %s", err.Error()), nil)
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
		return "", utils.NewError(fmt.Sprintf("failed to create form file: %s", err.Error()), nil)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to copy image to form: %s", err.Error()), nil)
	}

	// Add image tag
	if err := writer.WriteField("tag", projectName); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to add tag to form: %s", err.Error()), nil)
	}

	// Add version control (currently using GenerateImageHash)
	version, err := docker.GenerateImageHash()
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to generate image hash: %s", err.Error()), nil)
	}

	if err := writer.WriteField("version", version); err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to add version control to form: %s", err.Error()), nil)
	}

	writer.Close()

	// Create request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to create request: %s", err.Error()), nil)
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
		return "", utils.NewError(fmt.Sprintf("failed to deploy Docker image: %s", err.Error()), nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", utils.NewError(fmt.Sprintf("failed to deploy image: server returned %d", resp.StatusCode), nil)
	}

	return version, nil
}
