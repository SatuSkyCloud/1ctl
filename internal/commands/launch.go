package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"1ctl/internal/config"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v2"
)

// LaunchCommand is the onboarding wizard. It detects the project runtime,
// suggests sensible defaults, and writes a populated satusky.toml so the
// user can run `1ctl deploy` immediately. (#3 D-06)
func LaunchCommand() *cli.Command {
	return &cli.Command{
		Name:  "launch",
		Usage: "Interactive wizard: detect runtime, write satusky.toml, ready to deploy",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "non-interactive",
				Usage: "Skip prompts and accept all detected defaults",
			},
		},
		Action: handleLaunch,
	}
}

// Runtime is a detected project runtime. Each carries a suggested resource
// profile and a default port.
type Runtime struct {
	Name   string
	Marker string // file whose presence indicates this runtime
	CPU    string
	Memory string
	Port   int
	Notes  string
}

// detectableRuntimes is the ordered list of runtime probes. First match wins.
var detectableRuntimes = []Runtime{
	{Name: "Go", Marker: "go.mod", CPU: "0.5", Memory: "256Mi", Port: 8080},
	{Name: "Node.js / Bun", Marker: "package.json", CPU: "0.5", Memory: "512Mi", Port: 3000},
	{Name: "Python", Marker: "requirements.txt", CPU: "0.5", Memory: "512Mi", Port: 8000},
	{Name: "Python (Poetry)", Marker: "pyproject.toml", CPU: "0.5", Memory: "512Mi", Port: 8000},
	{Name: "Rust", Marker: "Cargo.toml", CPU: "0.5", Memory: "256Mi", Port: 8080},
	{Name: "Ruby", Marker: "Gemfile", CPU: "0.5", Memory: "512Mi", Port: 3000},
	{Name: "Java (Maven)", Marker: "pom.xml", CPU: "1", Memory: "1Gi", Port: 8080},
	{Name: "Java (Gradle)", Marker: "build.gradle", CPU: "1", Memory: "1Gi", Port: 8080},
	{Name: "PHP", Marker: "composer.json", CPU: "0.5", Memory: "512Mi", Port: 8080},
}

// detectRuntime returns the first matching runtime in detectableRuntimes,
// or an empty Runtime if none match.
func detectRuntime(dir string) Runtime {
	for _, r := range detectableRuntimes {
		if _, err := os.Stat(filepath.Join(dir, r.Marker)); err == nil {
			return r
		}
	}
	return Runtime{}
}

// hasDockerfile reports whether the directory contains a Dockerfile.
// Builds require either a Dockerfile or a --image flag, so the wizard
// flags missing Dockerfiles to the user.
func hasDockerfile(dir string) bool {
	for _, name := range []string{"Dockerfile", "Dockerfile.prod", "dockerfile"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func handleLaunch(c *cli.Context) error {
	dir, err := os.Getwd()
	if err != nil {
		return utils.NewError("failed to read working directory", err)
	}

	if _, err := os.Stat(filepath.Join(dir, config.DefaultConfigFile)); err == nil {
		return utils.NewError(fmt.Sprintf("%s already exists in %s — remove it or run `1ctl deploy` directly", config.DefaultConfigFile, dir), nil)
	}

	rt := detectRuntime(dir)
	appName := filepath.Base(dir)
	nonInteractive := c.Bool("non-interactive")

	utils.PrintHeader("1ctl launch")
	if rt.Name != "" {
		utils.PrintInfo("Detected runtime: %s (%s)", rt.Name, rt.Marker)
	} else {
		utils.PrintWarning("No runtime detected — using generic defaults. You can edit satusky.toml after.")
		rt = Runtime{CPU: "0.5", Memory: "256Mi", Port: 8080}
	}
	if !hasDockerfile(dir) {
		utils.PrintWarning("No Dockerfile in this directory — add one before running `1ctl deploy`, or pass `--image` to use a pre-built image.")
	}

	reader := bufio.NewReader(os.Stdin)
	appName = promptOrDefault(reader, "App name", appName, nonInteractive)
	port := promptIntOrDefault(reader, "Port", rt.Port, nonInteractive)
	cpu := promptOrDefault(reader, "CPU", rt.CPU, nonInteractive)
	memory := promptOrDefault(reader, "Memory", rt.Memory, nonInteractive)

	cfg := config.ProjectConfig{
		App: config.AppConfig{
			Name:   appName,
			Port:   port,
			CPU:    cpu,
			Memory: memory,
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

// writeLaunchConfig serialises a minimal satusky.toml. We don't reuse
// init.go's template because launch is the curated path: only the four
// fields the user just confirmed get written. Users discover the v2
// schema via 1ctl init.
func writeLaunchConfig(cfg *config.ProjectConfig) error {
	lines := []string{
		"[app]",
		fmt.Sprintf("  name = %q", cfg.App.Name),
		fmt.Sprintf("  port = %d", cfg.App.Port),
		fmt.Sprintf("  cpu = %q", cfg.App.CPU),
		fmt.Sprintf("  memory = %q", cfg.App.Memory),
		"",
		"# Run `1ctl init` to scaffold the full v2 schema",
		"# (volume, hpa, vpa, pdb, multicluster sections).",
	}
	return os.WriteFile(cfg.Path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}
