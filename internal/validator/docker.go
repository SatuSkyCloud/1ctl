package validator

import (
	"1ctl/internal/utils"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Common Dockerfile instructions to validate
var validInstructions = map[string]bool{
	"FROM":        true,
	"RUN":         true,
	"CMD":         true,
	"LABEL":       true,
	"EXPOSE":      true,
	"ENV":         true,
	"ADD":         true,
	"COPY":        true,
	"ENTRYPOINT":  true,
	"VOLUME":      true,
	"USER":        true,
	"WORKDIR":     true,
	"ARG":         true,
	"ONBUILD":     true,
	"STOPSIGNAL":  true,
	"HEALTHCHECK": true,
	"SHELL":       true,
}

// ValidateDockerInstallation checks if Docker is installed and running
func ValidateDockerInstallation() error {
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		return utils.NewError("docker is not installed or not running. Please install docker and try again", err)
	}
	return nil
}

// ValidateDockerfile validates the content of a Dockerfile at the given path
func ValidateDockerfile(path string) error {
	// Check if the Dockerfile exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return utils.NewError("dockerfile not found at %s", err)
	}

	// Open the Dockerfile
	file, err := os.Open(path)
	if err != nil {
		return utils.NewError("failed to open Dockerfile: %w", err)
	}
	defer file.Close()

	// Validate file properties
	info, err := file.Stat()
	if err != nil {
		return utils.NewError("failed to get Dockerfile info: %w", err)
	}

	if info.IsDir() {
		return utils.NewError("%s is a directory, not a Dockerfile", nil)
	}

	if info.Size() == 0 {
		return utils.NewError("dockerfile is empty", nil)
	}

	// Validate file content
	scanner := bufio.NewScanner(file)
	var errors []string
	lineNum := 0
	hasFrom := false
	firstNonCommentLine := true

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split the line into instruction and arguments
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		instruction := strings.ToUpper(parts[0])

		// Validate the first non-comment line
		if firstNonCommentLine && instruction != "FROM" && instruction != "ARG" {
			errors = append(errors, fmt.Sprintf("line %d: First instruction must be FROM or ARG", lineNum))
		}
		firstNonCommentLine = false

		// Check if the instruction is valid
		if !validInstructions[instruction] {
			errors = append(errors, fmt.Sprintf("line %d: invalid instruction '%s'", lineNum, instruction))
			continue
		}

		// Validate the FROM instruction
		if instruction == "FROM" {
			hasFrom = true
			if len(parts) < 2 {
				errors = append(errors, fmt.Sprintf("line %d: FROM instruction requires a base image", lineNum))
			} else if !isValidBaseImage(parts[1]) {
				errors = append(errors, fmt.Sprintf("line %d: invalid base image format '%s'", lineNum, parts[1]))
			}
		}
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		return utils.NewError("error reading Dockerfile: %w", err)
	}

	// Ensure a FROM instruction exists
	if !hasFrom {
		errors = append(errors, "Dockerfile must start with a FROM instruction")
	}

	// Return validation errors if any
	if len(errors) > 0 {
		return utils.NewError("dockerfile validation failed", nil)
	}

	return nil
}

// FindDockerfile searches for a valid Dockerfile in common locations within the specified directory
func FindDockerfile(dir string) (string, error) {
	dockerfilePaths := []string{
		"Dockerfile",
		"dockerfile",
		"docker/Dockerfile",
		".docker/Dockerfile",
		"build/Dockerfile",
		".build/Dockerfile",
	}

	// Try to validate each potential Dockerfile
	for _, fileName := range dockerfilePaths {
		path := filepath.Join(dir, fileName)
		if err := ValidateDockerfile(path); err == nil {
			return path, nil
		}
	}

	// Gather existing but invalid Dockerfile paths
	var invalidFiles []string
	for _, fileName := range dockerfilePaths {
		path := filepath.Join(dir, fileName)
		if _, err := os.Stat(path); err == nil {
			invalidFiles = append(invalidFiles, path)
		}
	}

	if len(invalidFiles) > 0 {
		return "", utils.NewError("found Dockerfile(s) at %v but they failed validation. Please check the file content", nil)
	}

	return "", utils.NewError("no valid Dockerfile found in the current directory or common locations", nil)
}

// isValidBaseImage checks if the base image name is valid
func isValidBaseImage(image string) bool {
	if image == "scratch" {
		return true
	}

	// Regex for valid image names
	const validImagePattern = `^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*(?::[a-z0-9]+(?:[._-][a-z0-9]+)*)?(?:@sha256:[a-f0-9]{64})?$`
	matched, err := regexp.MatchString(validImagePattern, image)
	if err != nil {
		return false
	}
	return matched
}
