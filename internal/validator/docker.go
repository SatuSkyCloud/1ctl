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
	"MAINTAINER":  true, // deprecated, but Docker still accepts it
}

// heredocPattern finds heredoc markers such as <<EOF, <<-EOF and <<"EOF".
var heredocPattern = regexp.MustCompile(`<<(-?)(["']?)([A-Za-z_][A-Za-z0-9_]*)(["']?)`)

// ValidateDockerInstallation checks if Docker is installed and running
func ValidateDockerInstallation() error {
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		return utils.NewError("docker is not installed or not running. Please install docker and try again", err)
	}
	return nil
}

// logicalLine is one Dockerfile instruction after joining continuation lines.
// lineNo is the line in the file where the instruction starts.
type logicalLine struct {
	text   string
	lineNo int
}

// ValidateDockerfile validates the content of a Dockerfile at the given path
func ValidateDockerfile(path string) error {
	// Check if the Dockerfile exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return utils.NewError(fmt.Sprintf("dockerfile not found at %s", path), nil)
	}

	// Open the Dockerfile
	file, err := os.Open(path) // #nosec G304 -- User-provided Dockerfile path is intentional
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

	// Process lines and handle continuations. Each instruction remembers the
	// line in the file where it starts, so error messages point at real lines.
	var processedLines []logicalLine
	var currentLine string
	startLine := 0

	for i := 0; i < len(rawLines); i++ {
		line := strings.TrimRight(rawLines[i], " \t")

		if currentLine == "" {
			startLine = i + 1
		}

		if strings.HasSuffix(line, "\\") {
			// Line continuation - remove backslash and continue building the instruction
			currentLine += strings.TrimSuffix(line, "\\") + " "
		} else {
			// End of instruction
			currentLine += line
			if strings.TrimSpace(currentLine) != "" {
				processedLines = append(processedLines, logicalLine{text: currentLine, lineNo: startLine})
				// Heredoc bodies (RUN <<EOF ... EOF) are script or file content,
				// not instructions, so skip them up to their terminators.
				i = skipHeredocBodies(currentLine, rawLines, i+1) - 1
			}
			currentLine = ""
		}

		// Handle case where file ends with a continuation
		if i == len(rawLines)-1 && currentLine != "" {
			processedLines = append(processedLines, logicalLine{text: currentLine, lineNo: startLine})
		}
	}

	// Validate processed lines
	var errors []string
	hasFrom := false
	firstNonCommentLine := true

	for _, processed := range processedLines {
		lineNo := processed.lineNo
		line := strings.TrimSpace(processed.text)

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
			errors = append(errors, fmt.Sprintf("line %d: First instruction must be FROM or ARG", lineNo))
		}
		firstNonCommentLine = false

		// Check if the instruction is valid
		if !validInstructions[instruction] {
			errors = append(errors, fmt.Sprintf("line %d: invalid instruction '%s'", lineNo, instruction))
			continue
		}

		// Validate the FROM instruction
		if instruction == "FROM" {
			hasFrom = true
			// Drop flags such as --platform=$BUILDPLATFORM: they come before the
			// image name and are checked by the builder, not here.
			parts = append(parts[:1], stripFlags(parts[1:])...)
			if len(parts) < 2 {
				errors = append(errors, fmt.Sprintf("line %d: FROM instruction requires a base image", lineNo))
			} else {
				// Handle multistage syntax: FROM image AS stage_name
				baseImage := parts[1]
				if len(parts) == 4 && strings.ToUpper(parts[2]) == "AS" {
					// This is a multistage build with AS clause
					stageName := parts[3]
					// Validate stage name (alphanumeric, underscore, hyphen allowed)
					if !isValidStageName(stageName) {
						errors = append(errors, fmt.Sprintf("line %d: invalid stage name '%s'", lineNo, stageName))
					}
				} else if len(parts) > 2 {
					// FROM instruction has extra arguments that aren't AS clause
					errors = append(errors, fmt.Sprintf("line %d: FROM instruction has invalid syntax '%s'", lineNo, strings.Join(parts[1:], " ")))
					continue
				}

				// Images like node:${NODE_VERSION} use ARG variables that Docker
				// fills in at build time, so the name can't be checked here.
				if !strings.Contains(baseImage, "$") && !isValidBaseImage(baseImage) {
					errors = append(errors, fmt.Sprintf("line %d: invalid base image format '%s'", lineNo, baseImage))
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
		errors = append(errors, "dockerfile must contain a FROM instruction")
	}

	// Return every problem found, with its line number, so the user knows what to fix
	if len(errors) > 0 {
		return utils.NewError(fmt.Sprintf("dockerfile validation failed at %s:\n  %s", path, strings.Join(errors, "\n  ")), nil)
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

// stripFlags removes leading --flag arguments (for example --platform=linux/amd64).
func stripFlags(args []string) []string {
	for len(args) > 0 && strings.HasPrefix(args[0], "--") {
		args = args[1:]
	}
	return args
}

// skipHeredocBodies returns the index of the first raw line after the bodies of
// any heredocs that the instruction opens. It returns start when the
// instruction has no heredoc. An unterminated heredoc runs to the end of the
// file, as it does for the Docker builder.
func skipHeredocBodies(instruction string, rawLines []string, start int) int {
	trimmed := strings.TrimSpace(instruction)
	if strings.HasPrefix(trimmed, "#") {
		return start
	}
	next := start
	for _, match := range heredocPattern.FindAllStringSubmatch(instruction, -1) {
		stripTabs := match[1] == "-"
		word := match[3]
		for next < len(rawLines) {
			body := rawLines[next]
			next++
			if stripTabs {
				body = strings.TrimLeft(body, "\t")
			}
			if body == word {
				break
			}
		}
	}
	return next
}
