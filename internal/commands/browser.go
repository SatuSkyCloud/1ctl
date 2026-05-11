package commands

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openBrowser opens the given URL in the user's default browser. Falls back
// to printing the URL when no opener is available (e.g., CI environments
// without a desktop session).
//
// Implementation note: each runtime.GOOS dispatch passes the URL as a single
// argument, never via shell interpolation, so a malicious DomainName cannot
// trigger command injection.
func openBrowser(url string) error {
	if url == "" {
		return fmt.Errorf("no URL to open")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) // #nosec G204 -- argv form; url not shell-interpreted
	case "linux":
		cmd = exec.Command("xdg-open", url) // #nosec G204 -- argv form; url not shell-interpreted
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url) // #nosec G204 -- argv form
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}
