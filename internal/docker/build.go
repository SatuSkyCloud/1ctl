package docker

import (
	"1ctl/internal/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// RegistryURL is the SatuSky container registry prefix.
	// Cloud-built images are pushed here: RegistryURL/<project>:<sha>.
	RegistryURL = "registry.satusky.com/satusky-container-registry"
)

// GetProjectName returns the project name derived from the git remote URL,
// falling back to the current directory name.
func GetProjectName() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url") // #nosec G204
	output, err := cmd.Output()
	if err == nil {
		remoteURL := strings.TrimSpace(string(output))
		parts := strings.Split(remoteURL, "/")
		if len(parts) > 0 {
			projectName := strings.TrimSuffix(parts[len(parts)-1], ".git")
			return projectName, nil
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", utils.NewError(fmt.Sprintf("failed to get working directory: %s", err.Error()), nil)
	}
	return filepath.Base(wd), nil
}
