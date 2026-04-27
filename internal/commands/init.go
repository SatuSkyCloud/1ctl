package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"1ctl/internal/config"
	"1ctl/internal/utils"

	"github.com/urfave/cli/v2"
)

func InitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Create a satusky.toml config file in the current directory",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Config name (e.g. staging → creates satusky.staging.toml)",
			},
		},
		Action: handleInit,
	}
}

func handleInit(c *cli.Context) error {
	configArg := c.String("config")
	filename := config.DefaultConfigFile
	if configArg != "" && !strings.HasSuffix(configArg, ".toml") {
		filename = fmt.Sprintf("satusky.%s.toml", configArg)
	} else if configArg != "" {
		filename = configArg
	}

	if _, err := os.Stat(filename); err == nil {
		return utils.NewError(fmt.Sprintf("%s already exists", filename), nil)
	}

	var base config.ProjectConfig
	if filename != config.DefaultConfigFile {
		if existing, err := config.FindConfig(""); err == nil && existing != nil {
			base = *existing
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		return utils.NewError(fmt.Sprintf("failed to get working directory: %s", err.Error()), nil)
	}
	if base.App.Name == "" {
		base.App.Name = filepath.Base(dir)
	}
	if base.App.Dockerfile == "" {
		base.App.Dockerfile = "Dockerfile"
	}
	if base.App.Port == 0 {
		base.App.Port = 8080
	}

	base.Path = filename
	lines := []string{"[app]"}
	lines = append(lines, fmt.Sprintf("  name = %q", base.App.Name))
	if base.App.Port != 0 {
		lines = append(lines, fmt.Sprintf("  port = %d", base.App.Port))
	}
	if base.App.Dockerfile != "" && base.App.Dockerfile != "Dockerfile" {
		lines = append(lines, fmt.Sprintf("  dockerfile = %q", base.App.Dockerfile))
	}
	if base.App.CPU != "" {
		lines = append(lines, fmt.Sprintf("  cpu = %q", base.App.CPU))
	}
	if base.App.Memory != "" {
		lines = append(lines, fmt.Sprintf("  memory = %q", base.App.Memory))
	}
	if base.App.Replicas > 0 && base.App.Replicas != 1 {
		lines = append(lines, fmt.Sprintf("  replicas = %d", base.App.Replicas))
	}
	if base.App.Domain != "" {
		lines = append(lines, fmt.Sprintf("  domain = %q", base.App.Domain))
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		return utils.NewError(fmt.Sprintf("failed to write %s: %s", filename, err.Error()), nil)
	}

	utils.PrintSuccess("Created %s", filename)
	if filename != config.DefaultConfigFile {
		utils.PrintInfo("Edit %s to configure resources and domain for this target.", filename)
		if configArg != "" {
			utils.PrintInfo("Then run: 1ctl deploy --config %s", configArg)
		}
	} else {
		utils.PrintInfo("Edit satusky.toml, then run: 1ctl deploy")
	}
	return nil
}
