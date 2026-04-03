package docker

import (
	"1ctl/internal/utils"
	"1ctl/internal/validator"
	"archive/tar"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
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

// isPodman returns true if the docker CLI is actually Podman.
func isPodman() bool {
	out, err := exec.Command("docker", "--version").Output() // #nosec G204
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), "podman")
}

// getImageID returns the image ID (sha256:...) for a local image name.
func getImageID(imageName string) (string, error) {
	out, err := exec.Command("docker", "inspect", "--format", "{{.Id}}", imageName).Output() // #nosec G204
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// patchManifestRepoTags rewrites the manifest.json inside a Docker-archive tar so that
// every entry has RepoTags set to [tag:latest]. This is necessary when Podman embeds
// "localhost/<name>" as the repo tag, which the registry upload service cannot resolve.
func patchManifestRepoTags(tarPath, tag string) error {
	// Read the original tar into memory.
	origData, err := os.ReadFile(tarPath) // #nosec G304
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tr := tar.NewReader(bytes.NewReader(origData))

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return err
		}

		if hdr.Name == "manifest.json" {
			var manifests []map[string]interface{}
			if err := json.Unmarshal(data, &manifests); err != nil {
				return err
			}
			repoTag := tag
			if !strings.Contains(repoTag, ":") {
				repoTag = repoTag + ":latest"
			}
			for i := range manifests {
				manifests[i]["RepoTags"] = []string{repoTag}
			}
			data, err = json.Marshal(manifests)
			if err != nil {
				return err
			}
			hdr.Size = int64(len(data))
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(data); err != nil {
			return err
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}

	return os.WriteFile(tarPath, buf.Bytes(), 0600) // #nosec G306
}

// SaveImage saves the Docker image as a tar archive in Docker format.
// When running under Podman, it saves by image ID (to avoid OCI format) and then
// patches manifest.json to set the correct RepoTags so the registry can tag by name.
func SaveImage(projectName, outputPath string) error {
	ref := projectName
	args := []string{"save"}

	if isPodman() {
		// Podman stores unqualified images as "localhost/<name>:latest".
		// Save by image ID to get a valid Docker archive without OCI metadata.
		if id, err := getImageID(projectName); err == nil && id != "" {
			ref = id
		}
		args = append(args, "--format", "docker-archive")
	}

	args = append(args, "-o", outputPath, ref)
	cmd := exec.Command("docker", args...) // #nosec G204 -- executable is fixed "docker", args are validated by callers
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return utils.NewError(fmt.Sprintf("failed to save Docker image: %s", err.Error()), nil)
	}

	// Patch the manifest so the registry service can locate the image by name after load.
	if isPodman() {
		if err := patchManifestRepoTags(outputPath, projectName); err != nil {
			return utils.NewError(fmt.Sprintf("failed to patch image manifest: %s", err.Error()), nil)
		}
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
