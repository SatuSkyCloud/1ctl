package docker

import (
	"1ctl/internal/utils"
	"1ctl/internal/validator"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type BuildOptions struct {
	DockerfilePath string
	Tag            string
	Context        string
}

const (
	RegistryURL = "registry.satusky.com/satusky-container-registry"
)

// Add input validation
func validateBuildInput(opts BuildOptions) error {
	if !filepath.IsAbs(opts.DockerfilePath) {
		opts.DockerfilePath = filepath.Clean(opts.DockerfilePath)
	}
	if !filepath.IsAbs(opts.Context) {
		opts.Context = filepath.Clean(opts.Context)
	}

	// Validate Dockerfile path
	if strings.Contains(opts.DockerfilePath, "..") {
		return utils.NewError("invalid dockerfile path: must not contain parent directory references", nil)
	}

	// Validate tag format
	if !regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9_./:]*$`).MatchString(opts.Tag) {
		return utils.NewError("invalid tag format", nil)
	}

	return nil
}

// Build builds a Docker image locally
func Build(opts BuildOptions) error {
	if err := validateBuildInput(opts); err != nil {
		return utils.NewError(fmt.Sprintf("invalid build options: %s", err.Error()), nil)
	}

	if opts.Context == "" {
		var err error
		opts.Context, err = os.Getwd()
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to get working directory: %s", err.Error()), nil)
		}
	}

	// Ensure Docker is installed
	if err := validator.ValidateDockerInstallation(); err != nil {
		return err
	}

	// Build the Docker image locally only
	// #nosec G204 -- User-provided build options are validated before use
	cmd := exec.Command("docker", "build",
		"--platform", "linux/amd64", // TODO: remove this once we have multi-arch support
		"-f", opts.DockerfilePath,
		"-t", opts.Tag,
		opts.Context,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to build Docker image: %s", err.Error()), nil)
	}

	return nil
}

// SaveImage saves the Docker image as a tar archive
func SaveImage(projectName, outputPath string) error {
	cmd := exec.Command("docker", "save", "-o", outputPath, projectName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to save Docker image: %s", err.Error()), nil)
	}

	return nil
}

// generate version control for image here
// right now we will just use has like this ""53a9cc9"
// but in future will have more sophisticated version control such as semver
// so we can easily track the version of the image and roll back to previous version if needed
// generateImageHash creates a unique hash based on timestamp and random string
func GenerateImageHash() (string, error) {
	// Create a random bytes slice
	randomBytes := make([]byte, 5)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// Convert to hex string
	hash := fmt.Sprintf("%x", randomBytes)
	return hash, nil
}

// GetProjectName attempts to get the project name from git remote or directory name
func GetProjectName() (string, error) {
	// Try to get from git remote first
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err == nil {
		remoteURL := strings.TrimSpace(string(output))
		// Extract project name from git URL
		parts := strings.Split(remoteURL, "/")
		if len(parts) > 0 {
			projectName := strings.TrimSuffix(parts[len(parts)-1], ".git")
			return projectName, nil
		}
	}

	// Fallback to directory name
	wd, err := os.Getwd()
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to get working directory: %s", err.Error()), nil)
	}
	return filepath.Base(wd), nil
}
