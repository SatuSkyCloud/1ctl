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
		return utils.NewError(fmt.Sprintf("dockerfile not found at %s", path), nil)
	}

	// Open the Dockerfile
	file, err := os.Open(path)
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to open Dockerfile: %s", err.Error()), nil)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck

	// Validate file properties
	info, err := file.Stat()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get Dockerfile info: %s", err.Error()), nil)
	}

	if info.IsDir() {
		return utils.NewError(fmt.Sprintf("%s is a directory, not a Dockerfile", path), nil)
	}

	if info.Size() == 0 {
		return utils.NewError(fmt.Sprintf("dockerfile is empty at %s", path), nil)
	}

	// Read all lines first and handle line continuations
	scanner := bufio.NewScanner(file)
	var rawLines []string
	for scanner.Scan() {
		rawLines = append(rawLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return utils.NewError(fmt.Sprintf("error reading Dockerfile: %s", err.Error()), nil)
	}

	// Process lines and handle continuations
	var processedLines []string
	var currentLine string

	for i, rawLine := range rawLines {
		line := strings.TrimRight(rawLine, " \t")

		if strings.HasSuffix(line, "\\") {
			// Line continuation - remove backslash and continue building the instruction
			currentLine += strings.TrimSuffix(line, "\\") + " "
		} else {
			// End of instruction
			currentLine += line
			if strings.TrimSpace(currentLine) != "" {
				processedLines = append(processedLines, currentLine)
			}
			currentLine = ""
		}

		// Handle case where file ends with a continuation
		if i == len(rawLines)-1 && currentLine != "" {
			processedLines = append(processedLines, currentLine)
		}
	}

	// Validate processed lines
	var errors []string
	hasFrom := false
	firstNonCommentLine := true

	for lineIndex, processedLine := range processedLines {
		line := strings.TrimSpace(processedLine)

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
			errors = append(errors, fmt.Sprintf("line %d: First instruction must be FROM or ARG", lineIndex+1))
		}
		firstNonCommentLine = false

		// Check if the instruction is valid
		if !validInstructions[instruction] {
			errors = append(errors, fmt.Sprintf("line %d: invalid instruction '%s'", lineIndex+1, instruction))
			continue
		}

		// Validate the FROM instruction
		if instruction == "FROM" {
			hasFrom = true
			if len(parts) < 2 {
				errors = append(errors, fmt.Sprintf("line %d: FROM instruction requires a base image", lineIndex+1))
			} else {
				// Handle multistage syntax: FROM image AS stage_name
				baseImage := parts[1]
				if len(parts) >= 4 && strings.ToUpper(parts[2]) == "AS" {
					// This is a multistage build with AS clause
					stageName := parts[3]
					// Validate stage name (alphanumeric, underscore, hyphen allowed)
					if !isValidStageName(stageName) {
						errors = append(errors, fmt.Sprintf("line %d: invalid stage name '%s'", lineIndex+1, stageName))
					}
				} else if len(parts) > 2 {
					// FROM instruction has extra arguments that aren't AS clause
					errors = append(errors, fmt.Sprintf("line %d: FROM instruction has invalid syntax '%s'", lineIndex+1, strings.Join(parts[1:], " ")))
					continue
				}

				if !isValidBaseImage(baseImage) {
					errors = append(errors, fmt.Sprintf("line %d: invalid base image format '%s'", lineIndex+1, baseImage))
				}
			}
		}

		// Validate COPY --from syntax for multistage builds
		if instruction == "COPY" && len(parts) >= 3 && strings.HasPrefix(parts[1], "--from=") { //nolint:staticcheck // return value is used in condition
			// This is fine - COPY --from syntax for multistage builds
			continue
		}
	}

	// Ensure a FROM instruction exists
	if !hasFrom {
		errors = append(errors, fmt.Sprintf("dockerfile must start with a FROM instruction at %s", path))
	}

	// Return validation errors if any
	if len(errors) > 0 {
		return utils.NewError(fmt.Sprintf("dockerfile validation failed at %s", path), nil)
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
		return "", utils.NewError(fmt.Sprintf("found Dockerfile(s) at %v but they failed validation. Please check the file content", invalidFiles), nil)
	}

	return "", utils.NewError(fmt.Sprintf("no valid Dockerfile found in the current directory or common locations at %s", dir), nil)
}

// isValidStageName checks if the stage name in multistage builds is valid
func isValidStageName(name string) bool {
	if name == "" {
		return false
	}
	// Stage names can contain alphanumeric characters, underscores, and hyphens
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	if err != nil {
		return false
	}
	return matched
}

// isValidBaseImage checks if the base image name is valid
func isValidBaseImage(image string) bool {
	if image == "scratch" {
		return true
	}

	// Enhanced regex for valid image names that supports:
	// - Registry hostnames (optional)
	// - Namespaces/organization names
	// - Repository names
	// - Tags
	// - Digest references
	// - Case insensitive matching for practical use
	const validImagePattern = `^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?)*(?::[0-9]+)?/)?[a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*(?:/[a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*)*(?::[a-zA-Z0-9]+(?:[._-][a-zA-Z0-9]+)*)?(?:@sha256:[a-f0-9]{64})?$`
	matched, err := regexp.MatchString(validImagePattern, image)
	if err != nil {
		return false
	}
	return matched
}
