package version

import (
	"fmt"
)

var (
	// Version is the current version of the CLI
	Version = "dev"

	// CommitHash is the git commit hash at build time
	CommitHash = ""

	// BuildDate is the date when the binary was built
	BuildDate = ""
)

// GetVersionInfo returns a formatted string with version information
func GetVersionInfo() string {
	version := Version
	if CommitHash != "" {
		version += fmt.Sprintf(" (commit: %s)", CommitHash)
	}
	if BuildDate != "" {
		version += fmt.Sprintf(" (built: %s)", BuildDate)
	}
	return version
}
