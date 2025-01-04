package api

import (
	"1ctl/internal/config"
	"1ctl/internal/context"
	"1ctl/internal/docker"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func UploadDockerImage(imagePath, projectName string) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image file: %w", err)
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
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to copy image to form: %w", err)
	}

	// Add image tag
	if err := writer.WriteField("tag", projectName); err != nil {
		return "", fmt.Errorf("failed to add tag to form: %w", err)
	}

	// Add version control (currently using GenerateImageHash)
	version, err := docker.GenerateImageHash()
	if err != nil {
		return "", fmt.Errorf("failed to generate image hash: %w", err)
	}

	if err := writer.WriteField("version", version); err != nil {
		return "", fmt.Errorf("failed to add version control to form: %w", err)
	}

	writer.Close()

	// Create request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	token := context.GetToken()
	if token == "" {
		return "", fmt.Errorf("not authenticated. Please run '1ctl auth login' to authenticate")
	}

	userConfigKey := context.GetUserConfigKey()
	if userConfigKey == "" {
		return "", fmt.Errorf("not authenticated. Please run '1ctl auth login' to authenticate")
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-satusky-api-key", token)
	req.Header.Set("x-satusky-config", userConfigKey)

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to deploy Docker image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to deploy image: server returned %d", resp.StatusCode)
	}

	return version, nil
}
