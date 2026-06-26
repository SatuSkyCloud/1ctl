package launch

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"1ctl/internal/config"
	"1ctl/internal/utils"
)

// Runtime is a detected project runtime. Each carries a suggested resource
// profile and a default port.
type Runtime struct {
	Name       string
	Marker     string // file whose presence indicates this runtime
	CPURequest string
	CPULimit   string
	Memory     string
	Port       int
	Notes      string
}

// detectableRuntimes is the ordered list of runtime probes. First match wins.
var detectableRuntimes = []Runtime{
	{Name: "Go", Marker: "go.mod", CPURequest: "250m", CPULimit: "1", Memory: "256Mi", Port: 8080},
	{Name: "Node.js / Bun", Marker: "package.json", CPURequest: "250m", CPULimit: "1", Memory: "512Mi", Port: 3000},
	{Name: "Python", Marker: "requirements.txt", CPURequest: "250m", CPULimit: "1", Memory: "512Mi", Port: 8000},
	{Name: "Python (Poetry)", Marker: "pyproject.toml", CPURequest: "250m", CPULimit: "1", Memory: "512Mi", Port: 8000},
	{Name: "Rust", Marker: "Cargo.toml", CPURequest: "250m", CPULimit: "1", Memory: "256Mi", Port: 8080},
	{Name: "Ruby", Marker: "Gemfile", CPURequest: "250m", CPULimit: "1", Memory: "512Mi", Port: 3000},
	{Name: "Java (Maven)", Marker: "pom.xml", CPURequest: "500m", CPULimit: "1", Memory: "1Gi", Port: 8080},
	{Name: "Java (Gradle)", Marker: "build.gradle", CPURequest: "500m", CPULimit: "1", Memory: "1Gi", Port: 8080},
	{Name: "PHP", Marker: "composer.json", CPURequest: "250m", CPULimit: "1", Memory: "512Mi", Port: 8080},
}

// detectRuntime returns the first matching runtime in detectableRuntimes.
func detectRuntime(dir string) Runtime {
	for _, r := range detectableRuntimes {
		if _, err := os.Stat(filepath.Join(dir, r.Marker)); err == nil {
			return r
		}
	}
	return Runtime{}
}

// hasDockerfile reports whether the directory contains a Dockerfile.
func hasDockerfile(dir string) bool {
	for _, name := range []string{"Dockerfile", "Dockerfile.prod", "dockerfile"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func handleLaunch(ctx context.Context, in launchInput) error {
	dir, err := os.Getwd()
	if err != nil {
		return utils.NewError("failed to read working directory", err)
	}

	if _, err := os.Stat(filepath.Join(dir, config.DefaultConfigFile)); err == nil {
		return utils.NewError(fmt.Sprintf("%s already exists in %s \u2014 remove it or run `1ctl deploy` directly", config.DefaultConfigFile, dir), nil)
	}

	rt := detectRuntime(dir)
	appName := filepath.Base(dir)

	utils.PrintHeader("1ctl launch")
	if rt.Name != "" {
		utils.PrintInfo("Detected runtime: %s (%s)", rt.Name, rt.Marker)
	} else {
		utils.PrintWarning("No runtime detected \u2014 using generic defaults. You can edit satusky.toml after.")
		rt = Runtime{CPURequest: "250m", CPULimit: "1", Memory: "256Mi", Port: 8080}
	}
	if !hasDockerfile(dir) {
		utils.PrintWarning("No Dockerfile in this directory \u2014 add one before running `1ctl deploy`, or pass `--image` to use a pre-built image.")
	}

	reader := bufio.NewReader(os.Stdin)
	appName = promptOrDefault(reader, "App name", appName, in.NonInteractive)
	port := promptIntOrDefault(reader, "Port", rt.Port, in.NonInteractive)
	cpuRequest := promptOrDefault(reader, "CPU request", rt.CPURequest, in.NonInteractive)
	cpuLimit := promptOrDefault(reader, "CPU limit", rt.CPULimit, in.NonInteractive)
	memory := promptOrDefault(reader, "Memory", rt.Memory, in.NonInteractive)

	cfg := config.ProjectConfig{
		App: config.AppConfig{
			Name:       appName,
			Port:       port,
			CPURequest: cpuRequest,
			CPULimit:   cpuLimit,
			Memory:     memory,
		},
		Path: filepath.Join(dir, config.DefaultConfigFile),
	}
	if err := writeLaunchConfig(&cfg); err != nil {
		return err
	}

	utils.PrintSuccess("Wrote %s", config.DefaultConfigFile)
	utils.PrintInfo("Next: 1ctl deploy")
	return nil
}

func promptOrDefault(r *bufio.Reader, label, def string, nonInteractive bool) string {
	if nonInteractive {
		return def
	}
	fmt.Printf("%s [%s]: ", label, def)
	line, err := r.ReadString('\n')
	if err != nil {
		return def
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func promptIntOrDefault(r *bufio.Reader, label string, def int, nonInteractive bool) int {
	s := promptOrDefault(r, label, fmt.Sprintf("%d", def), nonInteractive)
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n <= 0 {
		return def
	}
	return n
}

// writeLaunchConfig serialises a minimal satusky.toml.
func writeLaunchConfig(cfg *config.ProjectConfig) error {
	lines := []string{
		"[app]",
		fmt.Sprintf("  name = %q", cfg.App.Name),
		fmt.Sprintf("  port = %d", cfg.App.Port),
		fmt.Sprintf("  cpu_request = %q", cfg.App.CPURequest),
		fmt.Sprintf("  cpu_limit = %q", cfg.App.CPULimit),
		fmt.Sprintf("  memory = %q", cfg.App.Memory),
		"",
		"# Run `1ctl init` to scaffold the full v2 schema",
		"# (volume, hpa, vpa, pdb, multicluster sections).",
	}
	return os.WriteFile(cfg.Path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}
